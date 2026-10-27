package app

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain"
)

// The language model's review of the competitor candidates (ADR-0006): Claude Haiku, on an
// Anthropic key, first finds which candidates are one company, or a company on the list, and
// merges them; then it checks each company it has not checked, a batch at a time, for whether it
// is a competitor of Tribelt. Its check is a suggestion: a person adds or dismisses.

const (
	// CheckBatch is how many candidates the model checks at a time.
	CheckBatch = 40
	// MaxMerge is the most candidates, the most named first, the model compares in one go.
	MaxMerge = 1000
	// reviewTimeout bounds one review.
	reviewTimeout = 10 * time.Minute
)

// ErrNoReviewer is returned by Review when no Anthropic key is set up.
var ErrNoReviewer = errors.New("no Anthropic key is set up")

// ReviewProblem is why the last review stopped short.
type ReviewProblem string

const (
	ReviewProblemNone  ReviewProblem = ""
	ReviewProblemNoKey ReviewProblem = "noKey"
	ReviewProblemFails ReviewProblem = "failed"
)

// ReviewState is whether the model is reviewing, and why its last review stopped short.
type ReviewState struct {
	Running bool
	Problem ReviewProblem
}

type review struct {
	reviewers Reviewers
	onError   func(error)
	running   atomic.Bool
	problem   atomic.Value // ReviewProblem
}

// ReviewState says whether the model is reviewing the candidates, and why its last review stopped short.
func (s *Service) ReviewState() ReviewState {
	problem, _ := s.review.problem.Load().(ReviewProblem)
	return ReviewState{Running: s.review.running.Load(), Problem: problem}
}

// StartReview starts a review in the background, unless one is running; it reports whether it started.
func (s *Service) StartReview(ctx context.Context) bool {
	if !s.review.running.CompareAndSwap(false, true) {
		return false
	}
	s.spawn(func() {
		defer s.review.running.Store(false)
		bg, cancel := context.WithTimeout(context.WithoutCancel(ctx), reviewTimeout)
		defer cancel()
		err := s.Review(bg)
		switch {
		case err == nil:
			s.review.problem.Store(ReviewProblemNone)
		case errors.Is(err, ErrNoReviewer):
			s.review.problem.Store(ReviewProblemNoKey)
		default:
			s.review.problem.Store(ReviewProblemFails)
			s.review.onError(err)
		}
	})
	return true
}

// Review has the model merge the candidates that are one company, then check each company it has
// not checked. ErrNoReviewer when no Anthropic key is set up.
func (s *Service) Review(ctx context.Context) error {
	if s.review.reviewers == nil {
		return ErrNoReviewer
	}
	r, model, err := s.review.reviewers.Current(ctx)
	if err != nil {
		return fmt.Errorf("the language model: %w", err)
	}
	if r == nil {
		return ErrNoReviewer
	}
	directory, err := s.directory.Companies(ctx)
	if err != nil {
		return fmt.Errorf("list the companies: %w", err)
	}
	if err := s.merge(ctx, r, directory); err != nil {
		return err
	}
	all, err := s.candidatesOn(ctx, directory)
	if err != nil {
		return err
	}
	unchecked := domain.Unchecked(all)
	for len(unchecked) > 0 {
		batch := unchecked[:min(CheckBatch, len(unchecked))]
		unchecked = unchecked[len(batch):]
		answered, err := r.Check(ctx, batch)
		if err != nil {
			return fmt.Errorf("check the candidates: %w", err)
		}
		if err := s.repo.SaveCompanyVerdicts(ctx, domain.CompleteVerdicts(batch, answered, model, s.now())); err != nil {
			return fmt.Errorf("store the checks: %w", err)
		}
	}
	return nil
}

// merge has the model find the waiting candidates that are one company, and stores what it found.
func (s *Service) merge(ctx context.Context, r Reviewer, directory []domain.DirectoryCompany) error {
	merges, err := s.repo.Merges(ctx)
	if err != nil {
		return fmt.Errorf("candidate merges: %w", err)
	}
	all, err := s.candidatesOn(ctx, directory)
	if err != nil {
		return err
	}
	var asked []domain.Candidate
	for _, c := range all {
		if c.Status() == domain.CandidatePending && len(asked) < MaxMerge {
			asked = append(asked, c)
		}
	}
	if len(asked) == 0 {
		return nil
	}
	clusters, err := r.Merge(ctx, asked, directory)
	if err != nil {
		return fmt.Errorf("merge the candidates: %w", err)
	}
	if found := domain.MergesOf(clusters, asked, directory, merges, s.now()); len(found) > 0 {
		if err := s.repo.SaveMerges(ctx, found); err != nil {
			return fmt.Errorf("store the merges: %w", err)
		}
	}
	return nil
}
