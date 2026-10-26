package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain"
)

// maxFailure is the longest failure stored with an answer or a run.
const maxFailure = 300

// execution is one run as it is carried out: what is left to ask, and what stops it.
type execution struct {
	s         *Service
	run       domain.Run
	gw        Gateway
	directory []domain.DirectoryCompany
	budget    domain.Micros

	mu      sync.Mutex
	spent   domain.Micros
	stopped domain.RunStatus // set when the budget is reached; empty otherwise
	lost    bool             // set when another instance took the run over
	failure error            // the first answer that could not be stored
}

// Execute carries out a run held by this instance: it asks every task not answered yet, each
// provider's model in a goroutine of its own, reads every answer not read yet, and finishes the
// run done, or stopped when the month's budget is reached. A run another instance took over is
// left to it. Execute returns an error only when the run could not be recorded.
func (s *Service) Execute(ctx context.Context, run domain.Run) error {
	e, tasks, unread, err := s.resume(ctx, run)
	if err != nil {
		return s.fail(ctx, run.ID, err)
	}
	for _, a := range unread {
		if e.halted() {
			break
		}
		e.read(ctx, a)
	}
	var wg sync.WaitGroup
	for _, m := range run.Models {
		mine := make([]domain.Task, 0, len(tasks))
		for _, t := range tasks {
			if t.Provider == m.Provider {
				mine = append(mine, t)
			}
		}
		wg.Go(func() { e.provider(ctx, m, mine) })
	}
	wg.Wait()
	return e.finish(ctx)
}

// resume loads what a run has done: the tasks still to ask, and the answers stored without being
// read, as when a restart came between the two.
func (s *Service) resume(ctx context.Context, run domain.Run) (*execution, []domain.Task, []domain.Answer, error) {
	settings, err := s.settings(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	gw, err := s.gateways.Gateway(ctx, settings.CredentialID)
	if err != nil {
		return nil, nil, nil, err
	}
	directory, err := s.directory.Companies(ctx)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("list the companies: %w", err)
	}
	spent, err := s.repo.SpentSince(ctx, domain.MonthStart(s.now()))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("this month's spend: %w", err)
	}
	set := domain.CurrentPromptSet()
	if run.PromptSetVersion != set.Version {
		return nil, nil, nil, fmt.Errorf("the run asked prompt set %d, which this release no longer has", run.PromptSetVersion)
	}
	done, err := s.repo.Answers(ctx, run.ID, true)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("the run's answers: %w", err)
	}
	answered := map[domain.TaskKey]bool{}
	var unread []domain.Answer
	for _, a := range done {
		answered[a.Key()] = true
		if a.Failure == "" && a.Extraction == nil {
			unread = append(unread, a)
		}
	}
	var tasks []domain.Task
	for _, t := range domain.Tasks(set, run.Models) {
		if !answered[t.Key()] {
			tasks = append(tasks, t)
		}
	}
	e := &execution{s: s, run: run, gw: gw, directory: directory, budget: settings.MonthlyBudget, spent: spent}
	return e, tasks, unread, nil
}

// fail finishes a run that cannot go on as failed, with why.
func (s *Service) fail(ctx context.Context, id uuid.UUID, cause error) error {
	if err := s.repo.FinishRun(ctx, id, domain.RunFailed, clip(capitalise(cause.Error())), s.now()); err != nil {
		return fmt.Errorf("fail the run: %w", err)
	}
	return nil
}

// provider asks one provider's tasks in turn, its free model paced and, when the free model
// cannot finish, its cheapest paid model for the rest.
func (e *execution) provider(ctx context.Context, m domain.ModelChoice, tasks []domain.Task) {
	model, free := m.Model, m.Free
	for _, t := range tasks {
		if e.halted() {
			return
		}
		reply, err := e.ask(ctx, model, free, t)
		if errors.Is(err, domain.ErrRateLimited) && free && m.Fallback != "" {
			model, free = m.Fallback, false
			reply, err = e.ask(ctx, model, free, t)
		}
		if ctx.Err() != nil {
			return
		}
		a := domain.Answer{
			ID: uuid.New(), RunID: e.run.ID, Prompt: t.Prompt, PromptSetVersion: e.run.PromptSetVersion, Provider: t.Provider,
			Model: model, Mode: t.Mode, Reply: reply, At: e.s.now(),
		}
		if err != nil {
			a.Failure = clip(capitalise(err.Error()))
		}
		e.store(ctx, a)
	}
}

// ask asks one task, paced for a free model, retrying while the model is rate limited.
func (e *execution) ask(ctx context.Context, model string, free bool, t domain.Task) (domain.Reply, error) {
	var reply domain.Reply
	var err error
	for attempt := 0; attempt <= e.s.cfg.Retries; attempt++ {
		wait := e.s.cfg.RetryWait * time.Duration(attempt) // none before the first try
		if free {
			wait += e.s.cfg.Pace
		}
		if wait > 0 {
			if err := e.s.sleep(ctx, wait); err != nil {
				return domain.Reply{}, err
			}
		}
		var r domain.Reply
		r, err = e.gw.Ask(ctx, model, t.Prompt.Text, t.Mode == domain.ModeWeb)
		reply.Text, reply.Citations, reply.InputTokens, reply.OutputTokens = r.Text, r.Citations, reply.InputTokens+r.InputTokens, reply.OutputTokens+r.OutputTokens
		reply.Cost += r.Cost
		if !errors.Is(err, domain.ErrRateLimited) {
			return reply, err
		}
	}
	return reply, err
}

// store reads an answer, stores it with how it was read, and counts what it cost.
func (e *execution) store(ctx context.Context, a domain.Answer) {
	if a.Failure == "" {
		x := e.extract(ctx, a)
		a.Extraction = &x
	}
	if err := e.s.repo.AddAnswer(ctx, a); err != nil {
		e.halt(fmt.Errorf("store an answer: %w", err))
		return
	}
	e.count(ctx, a)
}

// read reads an answer stored without being read, and stores how.
func (e *execution) read(ctx context.Context, a domain.Answer) {
	x := e.extract(ctx, a)
	if err := e.s.repo.SetExtraction(ctx, a.ID, x, e.s.now()); err != nil {
		e.halt(fmt.Errorf("store an extraction: %w", err))
		return
	}
	e.count(ctx, domain.Answer{RunID: a.RunID, Extraction: &x})
}

// extract has the extractor read an answer, trying twice; an answer it cannot read is marked
// failed, never dropped.
func (e *execution) extract(ctx context.Context, a domain.Answer) domain.Extraction {
	in := ExtractInput{Prompt: a.Prompt.Text, Answer: a.Text, Citations: a.Citations, FactSheet: e.run.FactSheet}
	var used domain.Reply
	for range 2 {
		x, usage, err := e.gw.Extract(ctx, domain.ExtractorModel, in)
		used.InputTokens += usage.InputTokens
		used.OutputTokens += usage.OutputTokens
		used.Cost += usage.Cost
		if err == nil {
			read := domain.Read(x, a.Citations, e.directory)
			read.InputTokens, read.OutputTokens, read.Cost = used.InputTokens, used.OutputTokens, used.Cost
			return read
		}
	}
	return domain.Extraction{Failed: true, InputTokens: used.InputTokens, OutputTokens: used.OutputTokens, Cost: used.Cost}
}

// count adds an answer's cost to the month's spend, stops the run at the budget, and tells the
// repository this instance still runs it.
func (e *execution) count(ctx context.Context, a domain.Answer) {
	cost := a.Cost
	if a.Extraction != nil {
		cost += a.Extraction.Cost
	}
	e.mu.Lock()
	e.spent += cost
	if e.spent >= e.budget && e.stopped == "" {
		e.stopped = domain.RunStopped
	}
	e.mu.Unlock()
	held, err := e.s.repo.Heartbeat(ctx, e.run.ID, e.s.cfg.Owner, e.s.now())
	switch {
	case err != nil:
		e.halt(fmt.Errorf("record the run's progress: %w", err))
	case !held:
		e.mu.Lock()
		e.lost = true
		e.mu.Unlock()
	}
}

// halt stops the run because something could not be recorded.
func (e *execution) halt(err error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.failure == nil {
		e.failure = err
	}
}

// halted reports whether the run should ask nothing more: the budget is reached, another
// instance took it over, or something could not be recorded.
func (e *execution) halted() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.stopped != "" || e.lost || e.failure != nil || e.spent >= e.budget
}

// finish records how the run ended: done, stopped by the budget, or failed when something could
// not be recorded. A run another instance took over is its to finish.
func (e *execution) finish(ctx context.Context) error {
	e.mu.Lock()
	lost, failure, stopped, over := e.lost, e.failure, e.stopped, e.spent >= e.budget
	e.mu.Unlock()
	switch {
	case lost:
		return nil
	case failure != nil:
		_ = e.s.fail(ctx, e.run.ID, failure)
		return failure
	case stopped != "" || over:
		return e.s.repo.FinishRun(ctx, e.run.ID, domain.RunStopped, "The month's budget for Prompt runs was reached; the answers so far are kept.", e.s.now())
	default:
		return e.s.repo.FinishRun(ctx, e.run.ID, domain.RunDone, "", e.s.now())
	}
}

func clip(s string) string {
	if r := []rune(s); len(r) > maxFailure {
		return string(r[:maxFailure-1]) + "…"
	}
	return s
}

// capitalise makes a sentence of an error message.
func capitalise(s string) string {
	r := []rune(strings.TrimSuffix(s, ".") + ".")
	r[0] = []rune(strings.ToUpper(string(r[0])))[0]
	return string(r)
}
