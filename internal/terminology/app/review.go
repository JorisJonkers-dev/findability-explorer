package app

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/JorisJonkers-dev/findability-explorer/internal/terminology/domain"
)

// Verdicts stores the language model's verdicts on candidates, and what asking for them cost.
type Verdicts interface {
	// Verdicts returns every verdict.
	Verdicts(ctx context.Context) ([]domain.Verdict, error)
	// SaveReview stores a batch's verdicts, the decisions its sure ones make, and what asking used,
	// in one transaction; a decision on a candidate decided meanwhile is left out.
	SaveReview(ctx context.Context, verdicts []domain.Verdict, decisions []domain.Decision, usage []domain.Usage) error
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
	// ErrBudgetSpent is returned by Review when the month's language-model tokens are spent.
	ErrBudgetSpent = errors.New("this month's language-model budget is spent")
	// ErrBadAnswer is what a Judge returns when the model answered but not usably: it declined, its
	// answer was cut off, or it could not be read. Asking again the same way gets the same.
	ErrBadAnswer = errors.New("the language model gave no usable answer")
)

// Problem is why the last review stopped short.
type Problem string

const (
	ProblemNone    Problem = ""
	ProblemNoModel Problem = "noModel"
	ProblemBudget  Problem = "budget"
	ProblemFailed  Problem = "failed"
)

// BudgetSpent reports whether the month's language-model tokens, of every job together, are spent.
type BudgetSpent func(ctx context.Context) (bool, error)

// ReviewState is how far the language model is with the waiting candidates.
type ReviewState struct {
	Running bool
	// Waiting is how many waiting candidates the model has not reviewed yet.
	Waiting int
	Problem Problem
}

// examplesPerTier is how many map terms of each tier the model gets as examples.
const examplesPerTier = 8

// reviewer is the language model's part of the review queue.
type reviewer struct {
	verdicts Verdicts
	judges   Judges
	kick     chan struct{}
	running  atomic.Bool
	problem  atomic.Value // Problem
}

// WithReviewer lets a language model review the waiting candidates: verdicts stores what it
// decided, judges names the model to ask.
func (s *Service) WithReviewer(verdicts Verdicts, judges Judges) {
	s.review = &reviewer{verdicts: verdicts, judges: judges, kick: make(chan struct{}, 1)}
	s.review.problem.Store(ProblemNone)
}

// WithBudget stops the review once the month's language-model tokens are spent; without it, the
// review does not stop.
func (s *Service) WithBudget(spent BudgetSpent) { s.budget = spent }

// affordable is ErrBudgetSpent once the month's tokens are spent.
func (s *Service) affordable(ctx context.Context) error {
	if s.budget == nil {
		return nil
	}
	spent, err := s.budget(ctx)
	if err != nil {
		return fmt.Errorf("this month's tokens: %w", err)
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
// ErrNoModel when no model is set up, ErrBudgetSpent when the month's tokens are spent before a batch.
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
	terms := s.Terms()
	reviewed := 0
	for len(waiting) > 0 {
		if err := s.affordable(ctx); err != nil {
			return reviewed, err
		}
		batch := waiting[:min(domain.ReviewBatch, len(waiting))]
		waiting = waiting[len(batch):]
		var usage []domain.Usage
		verdicts, err := s.ask(ctx, judge, model, batch, terms, &usage)
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

// ReviewState says whether the model is reviewing, how many candidates wait for it (from counts),
// and why its last review stopped short. That no model was set up, or the budget was spent, is
// checked again, so a key added since, or a new month, shows at once.
func (s *Service) ReviewState(ctx context.Context, counts domain.Counts) ReviewState {
	if s.review == nil {
		return ReviewState{Waiting: counts.Unreviewed, Problem: ProblemNoModel}
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
	case ProblemNone, ProblemFailed:
	}
	return ReviewState{Running: s.review.running.Load() || len(s.review.kick) > 0, Waiting: counts.Unreviewed, Problem: problem}
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

// ReviewLoop reviews the waiting candidates now, every interval, and whenever StartReview asks,
// until ctx ends. onError hears why a review stopped short, except that no model is set up or the
// month's budget is spent.
func (s *Service) ReviewLoop(ctx context.Context, every time.Duration, onError func(error)) {
	if s.review == nil {
		return
	}
	tick := time.NewTicker(every)
	defer tick.Stop()
	for {
		s.reviewOnce(ctx, onError)
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		case <-s.review.kick:
		}
	}
}

func (s *Service) reviewOnce(ctx context.Context, onError func(error)) {
	s.review.running.Store(true)
	defer s.review.running.Store(false)
	_, err := s.Review(ctx)
	switch {
	case err == nil:
		s.review.problem.Store(ProblemNone)
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
