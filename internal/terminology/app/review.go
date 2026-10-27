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
	// SaveVerdicts stores the verdicts of one request and what it consumed.
	SaveVerdicts(ctx context.Context, verdicts []domain.Verdict, usage domain.Usage) error
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

// ErrNoModel is returned by Review when no language model is set up.
var ErrNoModel = errors.New("no language model is set up")

// Problem is why the last review stopped short.
type Problem string

const (
	ProblemNone    Problem = ""
	ProblemNoModel Problem = "noModel"
	ProblemFailed  Problem = "failed"
)

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
// ErrNoModel when no model is set up.
func (s *Service) Review(ctx context.Context) (int, error) {
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
	reviewed := 0
	for {
		all, err := s.candidates(ctx)
		if err != nil {
			return reviewed, err
		}
		batch := domain.Unreviewed(all)
		if len(batch) == 0 {
			return reviewed, nil
		}
		batch = batch[:min(domain.ReviewBatch, len(batch))]
		if err := s.judge(ctx, judge, model, batch); err != nil {
			return reviewed, err
		}
		reviewed += len(batch)
	}
}

// judge asks about one batch, stores the verdicts and records the sure ones as decisions.
func (s *Service) judge(ctx context.Context, judge Judge, model string, batch []domain.Candidate) error {
	answered, usage, err := judge.Judge(ctx, batch, domain.Examples(s.Terms(), examplesPerTier))
	usage.At = s.now()
	if err != nil {
		// What a failed request consumed still counts.
		if saveErr := s.review.verdicts.SaveVerdicts(ctx, nil, usage); saveErr != nil {
			return errors.Join(err, saveErr)
		}
		return fmt.Errorf("ask the language model: %w", err)
	}
	verdicts := domain.Complete(batch, answered, model, s.now())
	if err := s.review.verdicts.SaveVerdicts(ctx, verdicts, usage); err != nil {
		return fmt.Errorf("store the language model's verdicts: %w", err)
	}
	terms := s.Terms()
	for i, v := range verdicts {
		d, ok := v.Decision(batch[i], terms)
		if !ok {
			continue
		}
		if err := s.reviews.Decide(ctx, d); err != nil {
			if errors.Is(err, domain.ErrDecided) { // a person decided meanwhile
				continue
			}
			return fmt.Errorf("record the language model's decision: %w", err)
		}
		if d.Term != nil {
			terms = append(terms, *d.Term)
		}
	}
	return s.Load(ctx)
}

// ReviewState says whether the model is reviewing, how many candidates wait for it, and why its
// last review stopped short.
func (s *Service) ReviewState(ctx context.Context) (ReviewState, error) {
	if s.review == nil {
		return ReviewState{Problem: ProblemNoModel}, nil
	}
	all, err := s.candidates(ctx)
	if err != nil {
		return ReviewState{}, err
	}
	problem, _ := s.review.problem.Load().(Problem)
	return ReviewState{
		Running: s.review.running.Load() || len(s.review.kick) > 0,
		Waiting: len(domain.Unreviewed(all)),
		Problem: problem,
	}, nil
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
// until ctx ends. onError hears why a review stopped short, except that no model is set up.
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
	case ctx.Err() != nil:
	default:
		s.review.problem.Store(ProblemFailed)
		onError(err)
	}
}
