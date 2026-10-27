package app

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain"
)

// The language model's review of the competitor candidates (ADR-0006, ADR-0009): Claude Haiku, on
// an Anthropic key, first finds which candidates are one company, or a company on the list, and
// merges them; then it checks each waiting company it has not checked with the current
// domain.CheckVersion, a batch at a time, against Tribelt's target market, from what the answers
// say of it and its home page. A sure verdict decides the company; an unsure one leaves it to a
// person.

const (
	// CheckBatch is how many candidates the model checks at a time.
	CheckBatch = 40
	// MaxMerge is the most candidates, the most named first, the model compares in one go.
	MaxMerge = 1000
	// pageFetchers is how many home pages, each on another company's website, are fetched at once.
	pageFetchers = 6
	// reviewTimeout bounds one review; what it checked before then is kept.
	reviewTimeout = 30 * time.Minute
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
	pages     HomePages
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

// Review has the model merge the candidates that are one company, then check each waiting company
// it has not checked, and records the decisions its sure verdicts make. ErrNoReviewer when no
// Anthropic key is set up.
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
		if err := s.check(ctx, r, model, batch); err != nil {
			return err
		}
	}
	return nil
}

// check has the model check batch, stores its verdicts and what asking used, and records the
// decisions its sure verdicts make.
func (s *Service) check(ctx context.Context, r Reviewer, model string, batch []domain.Candidate) error {
	dossiers, err := s.dossiers(ctx, batch)
	if err != nil {
		return err
	}
	answered, usage, err := r.Check(ctx, dossiers)
	if saveErr := s.saveUsage(ctx, usage); err != nil || saveErr != nil {
		return errors.Join(wrap("check the candidates", err), saveErr)
	}
	verdicts := domain.CompleteVerdicts(batch, answered, model, s.now())
	if err := s.repo.SaveCompanyVerdicts(ctx, verdicts); err != nil {
		return fmt.Errorf("store the checks: %w", err)
	}
	return s.decideByModel(ctx, batch, verdicts)
}

// decideByModel records the decision each sure verdict makes on its candidate, by the language
// model, unless a person decided the candidate meanwhile. A company the list refuses waits for a
// person, with the model's verdict as the suggestion.
func (s *Service) decideByModel(ctx context.Context, batch []domain.Candidate, verdicts []domain.CompanyVerdict) error {
	all, err := s.candidates(ctx)
	if err != nil {
		return err
	}
	for i, asked := range batch {
		status, ok := verdicts[i].Decides(asked)
		if !ok {
			continue
		}
		c, err := pending(all, asked.ID)
		if err != nil {
			continue // decided meanwhile, or gone
		}
		if err := s.decideOneByModel(ctx, c, status, asked.Website); err != nil {
			return err
		}
	}
	return nil
}

// decideOneByModel records the language model's decision on c: an added one goes on the list
// with website first. One the list refuses is left waiting.
func (s *Service) decideOneByModel(ctx context.Context, c domain.Candidate, status domain.CandidateStatus, website string) error {
	competitorID := ""
	if status == domain.CandidateAdded {
		id, err := s.directory.Add(ctx, domain.ModelDecider, c.Name, website)
		if errors.Is(err, domain.ErrInvalid) || errors.Is(err, domain.ErrCompanyKnown) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("add %s: %w", c.Name, err)
		}
		competitorID = id
	}
	_, err := s.decide(ctx, c, status, competitorID, domain.ModelDecider)
	return err
}

// dossiers is what the model reads of each company of batch: what the answers say around its
// names, and its home page, fetched a few websites at a time.
func (s *Service) dossiers(ctx context.Context, batch []domain.Candidate) ([]domain.Dossier, error) {
	var ids []uuid.UUID
	for _, c := range batch {
		ids = append(ids, c.Quoted...)
	}
	texts, err := s.repo.AnswerTexts(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("the answers' text: %w", err)
	}
	out := make([]domain.Dossier, len(batch))
	var wg sync.WaitGroup
	slots := make(chan struct{}, pageFetchers)
	for i, c := range batch {
		var answers []string
		for _, id := range c.Quoted {
			answers = append(answers, texts[id])
		}
		var names []string
		for _, m := range c.Members {
			names = append(names, m.Name)
		}
		out[i] = domain.Dossier{Candidate: c, Snippets: domain.Snippets(answers, names)}
		if c.Website == "" {
			continue
		}
		wg.Go(func() {
			slots <- struct{}{}
			defer func() { <-slots }()
			page := s.review.pages.Fetch(ctx, c.Website)
			out[i].Home = &page
		})
	}
	wg.Wait()
	return out, nil
}

// saveUsage stores what the requests used, if they used anything.
func (s *Service) saveUsage(ctx context.Context, usage ...domain.ReviewUsage) error {
	var used []domain.ReviewUsage
	for _, u := range usage {
		if u.InputTokens+u.OutputTokens > 0 {
			u.At = s.now()
			used = append(used, u)
		}
	}
	if len(used) == 0 {
		return nil
	}
	if err := s.repo.SaveReviewUsage(ctx, used); err != nil {
		return fmt.Errorf("store what the review used: %w", err)
	}
	return nil
}

// wrap is err with what failed, or nil.
func wrap(what string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", what, err)
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
	clusters, usage, err := r.Merge(ctx, asked, directory)
	if saveErr := s.saveUsage(ctx, usage); err != nil || saveErr != nil {
		return errors.Join(wrap("merge the candidates", err), saveErr)
	}
	if found := domain.MergesOf(clusters, asked, directory, merges, s.now()); len(found) > 0 {
		if err := s.repo.SaveMerges(ctx, found); err != nil {
			return fmt.Errorf("store the merges: %w", err)
		}
	}
	return nil
}
