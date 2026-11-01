package app

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/JorisJonkers-dev/findability-explorer/internal/insight/domain"
	llm "github.com/JorisJonkers-dev/findability-explorer/internal/llm/domain"
	"github.com/JorisJonkers-dev/findability-explorer/internal/shared/work"
)

// The Recommendation run (the spec's section 10): the rules find and score the gaps, the Claude step
// writing recommendations places and words them, a batch at a time, and a run that ends replaces
// the open recommendations an earlier run wrote, keeping the done and edited ones. A run stopped,
// cut short by the budget, or failed changes nothing.

// TribeltPages is the port to the analysis context: the pages the newest finished scan of
// tribelt.nl read, with their text, and when that scan finished; none before such a scan.
type TribeltPages interface {
	Pages(ctx context.Context) ([]domain.Page, time.Time, error)
}

// Usage is what requests to the model used.
type Usage struct {
	InputTokens  int64
	OutputTokens int64
}

// Writer is the Claude step writing recommendations: its model with its instructions, given the
// gaps and tribelt.nl's pages as data. On an error the usage still counts what was used.
type Writer interface {
	Write(ctx context.Context, gaps []domain.Gap, pages []domain.Page) ([]domain.Written, Usage, error)
}

// Writers hands over the writer the step names now, with its model; a nil writer without an
// error when no Anthropic key is set up.
type Writers interface {
	Current(ctx context.Context) (Writer, string, error)
}

// BudgetSpent reports whether the month's Anthropic budget is spent.
type BudgetSpent func(ctx context.Context) (bool, error)

// WriteBatch is how many gaps the model writes in one request.
const WriteBatch = 10

// RecentRuns is how many past runs the page lists.
const RecentRuns = 20

var (
	// ErrNoWriter is returned when no Anthropic key is set up to write recommendations.
	ErrNoWriter = errors.New("no Anthropic key is set up to write recommendations")
	// ErrRunning is returned when a Recommendation run runs already.
	ErrRunning = errors.New("a Recommendation run is running already")
	// ErrNothingToStop is returned when no Recommendation run runs.
	ErrNothingToStop = errors.New("no Recommendation run runs")
	// errStopped and errBudget end a run early.
	errStopped = errors.New("stopped by hand")
	errBudget  = errors.New("the month's Anthropic budget is spent")
)

// recState is whether a run runs in the background, and whether it was asked to stop.
type recState struct {
	mu       sync.Mutex
	running  bool
	stopping bool
}

// WithWriting lets the Claude step writing recommendations run, on tribelt.nl's pages, within the
// month's budget; journal records each run for Activity, report hears that it failed, and spawn
// runs a run in the background (tests run it in place).
func (s *Service) WithWriting(writers Writers, pages TribeltPages, budget BudgetSpent, journal work.Journal, report func(error), spawn func(func())) {
	s.writing = writing{writers: writers, pages: pages, budget: budget, journal: journal, report: report, spawn: spawn}
}

// RecRuns is the Recommendation run as Manage shows it: whether one runs, what starting one would
// do and cost, and the past runs.
type RecRuns struct {
	Running bool
	// NoKey is true when no Anthropic key is set up.
	NoKey bool
	Model string
	// Replaces is how many open recommendations a run would replace.
	Replaces int
	// Stale is true when tribelt.nl was last scanned before the newest Prompt run started.
	Stale       bool
	PromptRunAt time.Time
	ScanAt      time.Time
	Terms       int
	Gaps        int
	Estimate    llm.Micros
	Runs        []domain.RecRun
}

// inputs are what a run reads now: the gaps, tribelt.nl's pages, and the facts behind them.
type inputs struct {
	gaps   []domain.Gap
	pages  []domain.Page
	scanAt time.Time
	run    *domain.Run
	terms  int
	// done are the facts the rules that decide when a recommendation is done read.
	done domain.DoneFacts
}

func (s *Service) gather(ctx context.Context) (inputs, error) {
	terms, err := s.ports.Terms.TermMap(ctx)
	if err != nil {
		return inputs{}, fmt.Errorf("the terminology map: %w", err)
	}
	companies, f, err := s.facts(ctx)
	if err != nil {
		return inputs{}, err
	}
	pages, scanAt, err := s.writing.pages.Pages(ctx)
	if err != nil {
		return inputs{}, fmt.Errorf("tribelt.nl's pages: %w", err)
	}
	gaps := domain.Gaps(domain.GapInputs{Companies: companies, Run: f.Run, Terms: terms, Tribelt: tribeltReadiness(f), Pages: pages})
	return inputs{
		gaps: gaps, pages: pages, scanAt: scanAt, run: f.Run, terms: len(terms.Terms),
		done: domain.DoneFacts{Pages: pages, Run: f.Run, Tribelt: tribeltReadiness(f)},
	}, nil
}

// RecommendationRuns is what starting a run would do and cost, and the past runs.
func (s *Service) RecommendationRuns(ctx context.Context) (RecRuns, error) {
	in, err := s.gather(ctx)
	if err != nil {
		return RecRuns{}, err
	}
	existing, err := s.ports.Store.Recommendations(ctx)
	if err != nil {
		return RecRuns{}, fmt.Errorf("the written recommendations: %w", err)
	}
	runs, err := s.ports.Store.RecRuns(ctx, RecentRuns)
	if err != nil {
		return RecRuns{}, fmt.Errorf("the past runs: %w", err)
	}
	writer, model, err := s.writing.writers.Current(ctx)
	if err != nil {
		return RecRuns{}, fmt.Errorf("the language model: %w", err)
	}
	out := RecRuns{
		Running: s.recRunning(), NoKey: writer == nil, Model: model, Replaces: domain.Replaces(domain.KeepDone(existing, in.done, s.now())), Stale: domain.Stale(in.scanAt, in.run),
		ScanAt: in.scanAt, Terms: in.terms, Gaps: len(in.gaps), Estimate: domain.EstimateRun(in.gaps, in.pages, WriteBatch, model), Runs: runs,
	}
	if in.run != nil {
		out.PromptRunAt = in.run.StartedAt
	}
	return out, nil
}

func (s *Service) recRunning() bool {
	s.rec.mu.Lock()
	defer s.rec.mu.Unlock()
	return s.rec.running
}

// StartRecommendationRun starts a run in the background, by user. ErrNoWriter without an Anthropic
// key; ErrRunning when one runs.
func (s *Service) StartRecommendationRun(ctx context.Context, user string) error {
	writer, model, err := s.writing.writers.Current(ctx)
	if err != nil {
		return fmt.Errorf("the language model: %w", err)
	}
	if writer == nil {
		return ErrNoWriter
	}
	s.rec.mu.Lock()
	if s.rec.running {
		s.rec.mu.Unlock()
		return ErrRunning
	}
	s.rec.running, s.rec.stopping = true, false
	s.rec.mu.Unlock()
	bg := context.WithoutCancel(ctx)
	s.writing.spawn(func() {
		defer func() {
			s.rec.mu.Lock()
			s.rec.running, s.rec.stopping = false, false
			s.rec.mu.Unlock()
		}()
		s.recommend(bg, writer, model, user)
	})
	return nil
}

// StopRecommendationRun stops the run that runs after the batch it writes; it changes nothing.
// ErrNothingToStop when none runs.
func (s *Service) StopRecommendationRun(context.Context) error {
	s.rec.mu.Lock()
	defer s.rec.mu.Unlock()
	if !s.rec.running {
		return ErrNothingToStop
	}
	s.rec.stopping = true
	return nil
}

// goOn is errStopped once an administrator stopped the run, errBudget once the budget is spent.
func (s *Service) goOn(ctx context.Context) error {
	s.rec.mu.Lock()
	stopping := s.rec.stopping
	s.rec.mu.Unlock()
	if stopping {
		return errStopped
	}
	if s.writing.budget == nil {
		return nil
	}
	spent, err := s.writing.budget(ctx)
	if err != nil {
		return fmt.Errorf("this month's budget: %w", err)
	}
	if spent {
		return errBudget
	}
	return nil
}

// recommend runs one Recommendation run and records it, and what it did, in the store and the journal.
func (s *Service) recommend(ctx context.Context, writer Writer, model, user string) {
	run := domain.RecRun{ID: s.newID(), StartedAt: s.now(), By: user, Model: model}
	entry := work.Start(ctx, s.writing.journal, work.KindRecommendationRun, user, run.StartedAt, s.writing.report)
	if err := s.ports.Store.BeginRecRun(ctx, run); err != nil {
		s.writing.report(fmt.Errorf("record the Recommendation run: %w", err))
	}
	replacement, used, done, total, err := s.write(ctx, writer, run)
	run.InputTokens, run.OutputTokens, run.Gaps = used.InputTokens, used.OutputTokens, total
	run.FinishedAt, run.Ending = s.now(), string(recEnding(err))
	if replacement != nil {
		run.Added, run.Replaced, run.Kept = len(replacement.Added), len(replacement.Replaced), len(replacement.Kept)
	}
	if err != nil && !errors.Is(err, errStopped) && !errors.Is(err, errBudget) {
		s.writing.report(err)
	}
	if err := s.ports.Store.FinishRecRun(ctx, run, replacement); err != nil {
		s.writing.report(fmt.Errorf("record how the Recommendation run ended: %w", err))
	}
	entry.Finish(ctx, work.Outcome{
		Ending: recEnding(err), Counts: work.Counts{Done: done, Total: total}, InputTokens: used.InputTokens, OutputTokens: used.OutputTokens,
		Model: run.Model,
	}, run.FinishedAt, s.writing.report)
}

func recEnding(err error) work.Ending {
	switch {
	case err == nil:
		return work.EndingDone
	case errors.Is(err, errStopped):
		return work.EndingStoppedByHand
	case errors.Is(err, errBudget):
		return work.EndingStoppedByBudget
	default:
		return work.EndingFailed
	}
}

// write has the model write every gap a batch at a time, and decides what the run replaces; nil
// when it ended early. It returns what asking used, and how many gaps of how many it wrote.
func (s *Service) write(ctx context.Context, writer Writer, run domain.RecRun) (*domain.Replacement, Usage, int, int, error) {
	in, err := s.gather(ctx)
	if err != nil {
		return nil, Usage{}, 0, 0, err
	}
	var used Usage
	var written []domain.Written
	done := 0
	for start := 0; start < len(in.gaps); start += WriteBatch {
		if err := s.goOn(ctx); err != nil {
			return nil, used, done, len(in.gaps), err
		}
		batch := in.gaps[start:min(start+WriteBatch, len(in.gaps))]
		w, u, err := writer.Write(ctx, batch, in.pages)
		used.InputTokens += u.InputTokens
		used.OutputTokens += u.OutputTokens
		if err != nil {
			return nil, used, done, len(in.gaps), fmt.Errorf("write the recommendations: %w", err)
		}
		written = append(written, w...)
		done += len(batch)
	}
	existing, err := s.ports.Store.Recommendations(ctx)
	if err != nil {
		return nil, used, done, len(in.gaps), fmt.Errorf("the written recommendations: %w", err)
	}
	// What the rules find done is kept, as what an administrator ticked off or edited is.
	r := domain.Replace(domain.KeepDone(existing, in.done, s.now()), domain.Place(in.gaps, written, in.pages, run.ID, s.now(), s.newID))
	return &r, used, done, len(in.gaps), nil
}
