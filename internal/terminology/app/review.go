package app

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/JorisJonkers-dev/findability-explorer/internal/shared/work"
	"github.com/JorisJonkers-dev/findability-explorer/internal/terminology/domain"
)

// Verdicts stores the language model's verdicts on candidates, and what asking for them cost.
type Verdicts interface {
	// Verdicts returns every verdict.
	Verdicts(ctx context.Context) ([]domain.Verdict, error)
	// SaveReview stores a batch's verdicts, the decisions its sure ones make, and what asking used,
	// in one transaction; a decision on a candidate decided meanwhile is left out.
	SaveReview(ctx context.Context, verdicts []domain.Verdict, decisions []domain.Decision, usage []domain.Usage) error
	// UsageSince sums what asking used since t, per model that answered.
	UsageSince(ctx context.Context, t time.Time) (domain.Spend, error)
	// ReviewOn reports whether the review runs by itself; it does until an administrator switches it off.
	ReviewOn(ctx context.Context) (bool, error)
	// SwitchReview stores whether the review runs by itself, with the administrator who switched it, when.
	SwitchReview(ctx context.Context, on bool, by string, at time.Time) error
}

// Judge is a language model that reviews candidates, given terms the map holds as examples. On an
// error the usage still counts what was consumed.
type Judge interface {
	Judge(ctx context.Context, candidates []domain.Candidate, examples []domain.Term) ([]domain.Verdict, domain.Usage, error)
}

// Judges hands over the judge the language-model settings name now, with its model; a nil judge
// without an error when no model is set up.
type Judges interface {
	Current(ctx context.Context) (Judge, string, error)
}

var (
	// ErrNoModel is returned by Review when there is no model to ask: no Anthropic key is set up.
	ErrNoModel = errors.New("no Anthropic key is set up")
	// ErrBudgetSpent is returned by Review when the month's Anthropic budget is spent.
	ErrBudgetSpent = errors.New("this month's language-model budget is spent")
	// ErrStopped is returned by Review when an administrator stopped it.
	ErrStopped = errors.New("stopped by hand")
	// ErrNothingToStop is returned by StopReview when no review runs or is asked for.
	ErrNothingToStop = errors.New("no term review runs")
	// ErrBadAnswer is what a Judge returns when the model answered but not usably: it declined, its
	// answer was cut off, or it could not be read. Asking again the same way gets the same.
	ErrBadAnswer = errors.New("the language model gave no usable response")
)

// Problem is why the last review stopped short.
type Problem string

const (
	ProblemNone    Problem = ""
	ProblemNoModel Problem = "noModel"
	ProblemBudget  Problem = "budget"
	ProblemFailed  Problem = "failed"
	// ProblemStopped is a review an administrator stopped; what it decided before is kept.
	ProblemStopped Problem = "stopped"
)

// BudgetSpent reports whether the month's Anthropic budget, of every job together, is spent.
type BudgetSpent func(ctx context.Context) (bool, error)

// ReviewState is how far the language model is with the waiting candidates.
type ReviewState struct {
	// On is whether the review runs by itself, every interval.
	On      bool
	Running bool
	// Waiting is how many waiting candidates the model has not reviewed yet.
	Waiting int
	Problem Problem
	// Month is what the review used this month.
	Month domain.Spend
}

// examplesPerTier is how many map terms of each tier the model gets as examples.
const examplesPerTier = 8

// reviewer is the language model's part of the review queue.
type reviewer struct {
	verdicts Verdicts
	judges   Judges
	kick     chan struct{}
	problem  atomic.Value // Problem

	mu sync.Mutex
	// running is true while a review runs, stopping once an administrator asked it to stop.
	running, stopping bool
}

// begin marks a review started.
func (r *reviewer) begin() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.running, r.stopping = true, false
}

// end marks the review ended.
func (r *reviewer) end() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.running, r.stopping = false, false
}

// goOn is ErrStopped once an administrator asked the review to stop.
func (r *reviewer) goOn() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.stopping {
		return ErrStopped
	}
	return nil
}

// isRunning reports whether a review runs, or is asked for.
func (r *reviewer) isRunning() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.running || len(r.kick) > 0
}

// WithReviewer lets a language model review the waiting candidates: verdicts stores what it
// decided, judges names the model to ask.
func (s *Service) WithReviewer(verdicts Verdicts, judges Judges) {
	s.review = &reviewer{verdicts: verdicts, judges: judges, kick: make(chan struct{}, 1)}
	s.review.problem.Store(ProblemNone)
}

// WithBudget stops the review once the month's Anthropic budget is spent; without it, the
// review does not stop.
func (s *Service) WithBudget(spent BudgetSpent) { s.budget = spent }

// affordable is ErrBudgetSpent once the month's Anthropic budget is spent.
func (s *Service) affordable(ctx context.Context) error {
	if s.budget == nil {
		return nil
	}
	spent, err := s.budget(ctx)
	if err != nil {
		return fmt.Errorf("this month's Anthropic budget: %w", err)
	}
	if spent {
		return ErrBudgetSpent
	}
	return nil
}

// verdicts are the model's verdicts; none without a reviewer.
func (s *Service) verdicts(ctx context.Context) ([]domain.Verdict, error) {
	if s.review == nil {
		return nil, nil
	}
	v, err := s.review.verdicts.Verdicts(ctx)
	if err != nil {
		return nil, fmt.Errorf("load the language model's verdicts: %w", err)
	}
	return v, nil
}

// Review asks the language model about every waiting candidate it has not reviewed, a batch at a
// time, and records its sure verdicts as its decisions. It returns how many it reviewed;
// ErrNoModel when no model is set up, ErrBudgetSpent when the month's Anthropic budget is spent
// before a batch, ErrStopped when an administrator stopped it, keeping what it decided before.
func (s *Service) Review(ctx context.Context) (int, error) {
	n, err := s.reviewAll(ctx)
	if n > 0 { // the accepted terms join the map
		if loadErr := s.Load(ctx); loadErr != nil {
			return n, errors.Join(err, loadErr)
		}
	}
	return n, err
}

func (s *Service) reviewAll(ctx context.Context) (int, error) {
	if s.review == nil {
		return 0, ErrNoModel
	}
	judge, model, err := s.review.judges.Current(ctx)
	if err != nil {
		return 0, fmt.Errorf("the language model: %w", err)
	}
	if judge == nil {
		return 0, ErrNoModel
	}
	all, err := s.candidates(ctx)
	if err != nil {
		return 0, err
	}
	waiting := domain.Unreviewed(all)
	if len(waiting) == 0 {
		return 0, nil
	}
	// A review the budget allows no batch of asks nothing: there is no work to record.
	if err := s.affordable(ctx); err != nil {
		return 0, err
	}
	entry := work.Start(ctx, s.journal, work.KindTermReview, domain.ModelReviewer, s.now(), s.report)
	var used domain.Usage
	reviewed, err := s.reviewBatches(ctx, judge, model, waiting, &used)
	entry.Finish(context.WithoutCancel(ctx), work.Outcome{
		Ending: ending(err), Counts: work.Counts{Done: reviewed, Total: len(waiting)},
		InputTokens: used.InputTokens, OutputTokens: used.OutputTokens, Model: used.Model,
	}, s.now(), s.report)
	return reviewed, err
}

// reviewBatches asks the model about waiting a batch at a time, adding what asking used to used.
func (s *Service) reviewBatches(ctx context.Context, judge Judge, model string, waiting []domain.Candidate, used *domain.Usage) (int, error) {
	terms := s.Terms()
	reviewed := 0
	for len(waiting) > 0 {
		if err := s.review.goOn(); err != nil {
			return reviewed, err
		}
		if err := s.affordable(ctx); err != nil {
			return reviewed, err
		}
		batch := waiting[:min(domain.ReviewBatch, len(waiting))]
		waiting = waiting[len(batch):]
		var usage []domain.Usage
		verdicts, err := s.ask(ctx, judge, model, batch, terms, &usage)
		for _, u := range usage {
			used.InputTokens += u.InputTokens
			used.OutputTokens += u.OutputTokens
			used.Model = u.Model
		}
		if err != nil {
			// What the failed requests consumed still counts.
			return reviewed, errors.Join(fmt.Errorf("ask the language model: %w", err), s.review.verdicts.SaveReview(ctx, nil, nil, usage))
		}
		var decisions []domain.Decision
		decisions, terms = decide(batch, verdicts, terms)
		if err := s.review.verdicts.SaveReview(ctx, verdicts, decisions, usage); err != nil {
			return reviewed, fmt.Errorf("store the language model's review: %w", err)
		}
		reviewed += len(batch)
	}
	return reviewed, nil
}

// ending is how a review that returned err ended.
func ending(err error) work.Ending {
	switch {
	case err == nil:
		return work.EndingDone
	case errors.Is(err, ErrStopped):
		return work.EndingStoppedByHand
	case errors.Is(err, ErrBudgetSpent):
		return work.EndingStoppedByBudget
	default:
		return work.EndingFailed
	}
}

// WithJournal records each review that has candidates to review in j, as Activity lists it;
// report hears that j failed, which leaves the review alone.
func (s *Service) WithJournal(j work.Journal, report func(error)) {
	s.journal, s.report = j, report
}

// decide is the decisions the sure verdicts on batch make, and the map with the terms they accept;
// a term accepted earlier in the review counts, so the same term is not accepted twice.
func decide(batch []domain.Candidate, verdicts []domain.Verdict, terms []domain.Term) ([]domain.Decision, []domain.Term) {
	var decisions []domain.Decision
	for i, v := range verdicts {
		if d, ok := v.Decision(batch[i], terms); ok {
			decisions = append(decisions, d)
			if d.Term != nil {
				terms = append(terms, *d.Term)
			}
		}
	}
	return decisions, terms
}

// ask asks the model about batch and returns a verdict per candidate, in order. When it gives no
// usable answer, each half is asked apart, so one phrase it cannot answer does not hold up the
// rest; that phrase gets an unsure verdict and waits for a person. Any other failure stops it.
func (s *Service) ask(ctx context.Context, judge Judge, model string, batch []domain.Candidate, terms []domain.Term, usage *[]domain.Usage) ([]domain.Verdict, error) {
	answered, u, err := judge.Judge(ctx, batch, domain.Examples(terms, examplesPerTier))
	u.At = s.now()
	*usage = append(*usage, u)
	switch {
	case err == nil:
		return domain.Complete(batch, answered, model, s.now()), nil
	case !errors.Is(err, ErrBadAnswer):
		return nil, err
	case len(batch) == 1:
		return domain.Complete(batch, nil, model, s.now()), nil
	}
	half := len(batch) / 2
	first, err := s.ask(ctx, judge, model, batch[:half], terms, usage)
	if err != nil {
		return nil, err
	}
	second, err := s.ask(ctx, judge, model, batch[half:], terms, usage)
	if err != nil {
		return nil, err
	}
	return append(first, second...), nil
}

// ReviewState says whether the review runs by itself and whether it is reviewing now, how many
// candidates wait for it (from counts), why its last review stopped short, and what it used this
// month. That no model was set up, or the budget was spent, is checked again, so a key added
// since, or a new month, shows at once.
func (s *Service) ReviewState(ctx context.Context, counts domain.Counts) (ReviewState, error) {
	if s.review == nil {
		return ReviewState{Waiting: counts.Unreviewed, Problem: ProblemNoModel}, nil
	}
	on, err := s.review.verdicts.ReviewOn(ctx)
	if err != nil {
		return ReviewState{}, fmt.Errorf("whether the term review is on: %w", err)
	}
	now := s.now().UTC()
	month, err := s.review.verdicts.UsageSince(ctx, time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		return ReviewState{}, fmt.Errorf("what the term review used this month: %w", err)
	}
	problem, _ := s.review.problem.Load().(Problem)
	switch problem {
	case ProblemNoModel:
		if judge, _, err := s.review.judges.Current(ctx); err == nil && judge != nil {
			problem = ProblemNone
		}
	case ProblemBudget:
		if s.affordable(ctx) == nil {
			problem = ProblemNone
		}
	case ProblemNone, ProblemFailed, ProblemStopped:
	}
	return ReviewState{On: on, Running: s.review.isRunning(), Waiting: counts.Unreviewed, Problem: problem, Month: month}, nil
}

// SwitchReview turns the review that runs by itself on or off, recording user as who switched it.
// Off, nothing is reviewed until an administrator asks (StartReview); a review that runs goes on
// until it ends or is stopped. ErrNoModel without a reviewer.
func (s *Service) SwitchReview(ctx context.Context, user string, on bool) error {
	if s.review == nil {
		return ErrNoModel
	}
	if err := s.review.verdicts.SwitchReview(ctx, on, user, s.now()); err != nil {
		return fmt.Errorf("switch the term review: %w", err)
	}
	return nil
}

// StopReview stops the review that runs after the batch it asks about, which it keeps, as it keeps
// every decision made before; a review asked for that has not started does not start. The review's
// state says it was stopped by hand. ErrNothingToStop when no review runs or is asked for.
func (s *Service) StopReview(context.Context) error {
	if s.review == nil {
		return ErrNothingToStop
	}
	r := s.review
	r.mu.Lock()
	defer r.mu.Unlock()
	asked := false
	select {
	case <-r.kick:
		asked = true
	default:
	}
	switch {
	case r.running:
		r.stopping = true
	case asked:
		r.problem.Store(ProblemStopped)
	default:
		return ErrNothingToStop
	}
	return nil
}

// StartReview asks the review loop to review the waiting candidates now.
func (s *Service) StartReview() {
	if s.review == nil {
		return
	}
	select {
	case s.review.kick <- struct{}{}:
	default: // a review is asked for already
	}
}

// ReviewLoop reviews the waiting candidates now and every interval while the review is on, and
// whenever StartReview asks, on or off, until ctx ends. onError hears why a review stopped short,
// except that no model is set up, the month's budget is spent, or an administrator stopped it; and
// that whether the review is on could not be read, which counts as off.
func (s *Service) ReviewLoop(ctx context.Context, every time.Duration, onError func(error)) {
	if s.review == nil {
		return
	}
	tick := time.NewTicker(every)
	defer tick.Stop()
	asked := false
	for {
		if asked || s.reviewOn(ctx, onError) {
			s.reviewOnce(ctx, onError)
		}
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			asked = false
		case <-s.review.kick:
			asked = true
		}
	}
}

// reviewOn reports whether the review runs by itself; not when that cannot be read, which onError hears.
func (s *Service) reviewOn(ctx context.Context, onError func(error)) bool {
	on, err := s.review.verdicts.ReviewOn(ctx)
	if err != nil {
		if ctx.Err() == nil {
			onError(fmt.Errorf("whether the term review is on: %w", err))
		}
		return false
	}
	return on
}

func (s *Service) reviewOnce(ctx context.Context, onError func(error)) {
	s.review.begin()
	defer s.review.end()
	_, err := s.Review(ctx)
	switch {
	case err == nil:
		s.review.problem.Store(ProblemNone)
	case errors.Is(err, ErrStopped):
		s.review.problem.Store(ProblemStopped)
	case errors.Is(err, ErrNoModel):
		s.review.problem.Store(ProblemNoModel)
	case errors.Is(err, ErrBudgetSpent):
		s.review.problem.Store(ProblemBudget)
	case ctx.Err() != nil:
	default:
		s.review.problem.Store(ProblemFailed)
		onError(err)
	}
}
