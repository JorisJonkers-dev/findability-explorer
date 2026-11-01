package app

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain"
	"github.com/JorisJonkers-dev/findability-explorer/internal/shared/work"
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

var (
	// ErrNoReviewer is returned by Review when no Anthropic key is set up.
	ErrNoReviewer = errors.New("no Anthropic key is set up")
	// ErrBudgetSpent is returned by Review when the month's Anthropic budget is spent.
	ErrBudgetSpent = errors.New("this month's language-model budget is spent")
	// ErrStopped is returned by Review and CheckList when an administrator stopped them.
	ErrStopped = errors.New("stopped by hand")
	// ErrNothingToStop is returned by a stop that finds nothing running.
	ErrNothingToStop = errors.New("there is nothing running to stop")
)

// ReviewProblem is why the last review stopped short.
type ReviewProblem string

const (
	ReviewProblemNone   ReviewProblem = ""
	ReviewProblemNoKey  ReviewProblem = "noKey"
	ReviewProblemBudget ReviewProblem = "budget"
	ReviewProblemFails  ReviewProblem = "failed"
	// ReviewProblemStopped is a review an administrator stopped; what it decided before is kept.
	ReviewProblemStopped ReviewProblem = "stopped"
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
	// candidates is the review of the candidates, list the check of the companies on the list.
	candidates, list runState
}

// runState is whether a review runs in the background, whether an administrator asked it to stop,
// and why its last run stopped short.
type runState struct {
	mu       sync.Mutex
	running  bool
	stopping bool
	problem  ReviewProblem
	// counts and in, out are what the run did and used, for the work journal.
	counts  work.Counts
	in, out int64
}

// planned sets how many companies the run checks.
func (r *runState) planned(n int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.counts.Total = n
}

// did counts n companies checked.
func (r *runState) did(n int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.counts.Done += n
}

// used counts what a request used.
func (r *runState) used(u domain.ReviewUsage) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.in, r.out = r.in+u.InputTokens, r.out+u.OutputTokens
}

// outcome is the run's outcome, ended as ending.
func (r *runState) outcome(ending work.Ending) work.Outcome {
	r.mu.Lock()
	defer r.mu.Unlock()
	return work.Outcome{Ending: ending, Counts: r.counts, InputTokens: r.in, OutputTokens: r.out}
}

func (r *runState) state() ReviewState {
	r.mu.Lock()
	defer r.mu.Unlock()
	return ReviewState{Running: r.running, Problem: r.problem}
}

// begin marks a run started, unless one runs already; it reports whether it started.
func (r *runState) begin() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.running {
		return false
	}
	r.running, r.stopping = true, false
	r.counts, r.in, r.out = work.Counts{}, 0, 0
	return true
}

// end marks the run ended, with the problem it ran into.
func (r *runState) end(problem ReviewProblem) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.running, r.stopping, r.problem = false, false, problem
}

// stop asks the run to stop before its next step; ErrNothingToStop when none runs.
func (r *runState) stop() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.running {
		return ErrNothingToStop
	}
	r.stopping = true
	return nil
}

// goOn is ErrStopped once an administrator asked the run to stop.
func (r *runState) goOn() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.stopping {
		return ErrStopped
	}
	return nil
}

// ReviewState says whether the model is reviewing the candidates, and why its last review stopped short.
func (s *Service) ReviewState() ReviewState { return s.review.candidates.state() }

// StartReview starts a review in the background, unless one is running; it reports whether it started.
func (s *Service) StartReview(ctx context.Context) bool {
	return s.startIn(ctx, work.KindCandidateReview, &s.review.candidates, s.Review)
}

// StopReview stops the review of the candidates that runs: it asks nothing after the step it is
// on, which it finishes and keeps, as it keeps every decision made before. The review's state says
// it was stopped by hand. ErrNothingToStop when no review runs.
func (s *Service) StopReview(context.Context) error { return s.review.candidates.stop() }

// startIn runs work in the background on st, unless it runs already, within reviewTimeout; it
// reports whether it started.
func (s *Service) startIn(ctx context.Context, kind work.Kind, st *runState, run func(context.Context) error) bool {
	if !st.begin() {
		return false
	}
	s.spawn(func() {
		bg, cancel := context.WithTimeout(context.WithoutCancel(ctx), reviewTimeout)
		defer cancel()
		entry := work.Start(bg, s.journal, kind, journalBy, s.now(), s.reportJournal)
		err := run(bg)
		problem, ending := ReviewProblemNone, work.EndingDone
		switch {
		case err == nil:
		case errors.Is(err, ErrStopped):
			problem, ending = ReviewProblemStopped, work.EndingStoppedByHand
		case errors.Is(err, ErrNoReviewer):
			problem, ending = ReviewProblemNoKey, work.EndingFailed
		case errors.Is(err, ErrBudgetSpent):
			problem, ending = ReviewProblemBudget, work.EndingStoppedByBudget
		default:
			problem, ending = ReviewProblemFails, work.EndingFailed
			s.review.onError(err)
		}
		entry.Finish(context.WithoutCancel(bg), st.outcome(ending), s.now(), s.reportJournal)
		st.end(problem)
	})
	return true
}

// journalBy is who the work journal names as having started a review: an administrator asks for
// it, and the language model does it.
const journalBy = "language model"

// reportJournal reports that the work journal failed, when anyone hears it.
func (s *Service) reportJournal(err error) {
	if s.review.onError != nil {
		s.review.onError(err)
	}
}

// Review has the model merge the candidates that are one company, then check each waiting company
// it has not checked, and records the decisions its sure verdicts make. ErrNoReviewer when no
// Anthropic key is set up; ErrBudgetSpent when the month's Anthropic budget is spent before
// the merge or a batch, and ErrStopped when an administrator stopped it, with what was checked
// before kept either way.
func (s *Service) Review(ctx context.Context) error {
	r, model, err := s.reviewer(ctx)
	if err != nil {
		return err
	}
	if err := s.goOn(ctx, &s.review.candidates); err != nil {
		return err
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
	s.review.candidates.planned(len(unchecked))
	for len(unchecked) > 0 {
		if err := s.goOn(ctx, &s.review.candidates); err != nil {
			return err
		}
		batch := unchecked[:min(CheckBatch, len(unchecked))]
		unchecked = unchecked[len(batch):]
		if err := s.check(ctx, r, model, batch); err != nil {
			return err
		}
		s.review.candidates.did(len(batch))
	}
	return nil
}

// reviewer is the language model and its name, or ErrNoReviewer when no Anthropic key is set up.
func (s *Service) reviewer(ctx context.Context) (Reviewer, string, error) {
	if s.review.reviewers == nil {
		return nil, "", ErrNoReviewer
	}
	r, model, err := s.review.reviewers.Current(ctx)
	if err != nil {
		return nil, "", fmt.Errorf("the language model: %w", err)
	}
	if r == nil {
		return nil, "", ErrNoReviewer
	}
	return r, model, nil
}

// goOn is ErrStopped once an administrator stopped the run on st, ErrBudgetSpent once the month's
// Anthropic budget is spent.
func (s *Service) goOn(ctx context.Context, st *runState) error {
	if err := st.goOn(); err != nil {
		return err
	}
	return s.tokensLeft(ctx)
}

// tokensLeft is ErrBudgetSpent once the month's Anthropic budget is spent.
func (s *Service) tokensLeft(ctx context.Context) error {
	spent, err := s.tokensSpent(ctx)
	if err != nil {
		return err
	}
	if spent {
		return ErrBudgetSpent
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
	s.review.candidates.used(usage)
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
	for _, asked := range batch {
		i := slices.IndexFunc(verdicts, func(v domain.CompanyVerdict) bool { return v.CandidateID == asked.ID })
		if i < 0 {
			continue // the model said nothing about it: it is asked about again
		}
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
	s.review.candidates.used(usage)
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
