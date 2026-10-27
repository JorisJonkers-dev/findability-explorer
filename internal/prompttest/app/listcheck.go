package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain"
)

// The language model's check of the companies on the competitor list (ADR-0010): Claude Haiku
// checks each competitor on the list as it checks a candidate, against Tribelt's target market,
// from its names, its website, its home page and what the answers say of it. It runs only when an
// administrator asks. A sure "not a competitor" archives the company through the Directory, by
// domain.ModelDecider, unless a person kept it; the rest stay on the list.

// ListCheckState says whether the model is checking the companies on the list, and why its last
// check stopped short.
func (s *Service) ListCheckState() ReviewState { return s.review.list.state() }

// StartListCheck starts a check of the list in the background, unless one is running; it reports
// whether it started.
func (s *Service) StartListCheck(ctx context.Context) bool {
	return s.startIn(ctx, &s.review.list, s.CheckList)
}

// ListChecks returns the model's checks of the competitors on the list to show a person, and how
// many competitors the next check takes.
func (s *Service) ListChecks(ctx context.Context) ([]domain.ListCheck, int, error) {
	directory, err := s.directory.Companies(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("list the companies: %w", err)
	}
	checks, err := s.repo.ListChecks(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("the checks of the list: %w", err)
	}
	return domain.ShownChecks(directory, checks), len(domain.ToCheck(directory, checks)), nil
}

// CheckList has the model check each competitor on the list it has not checked with the current
// domain.CheckVersion, a batch at a time, and archives each it is sure does not compete.
// ErrNoReviewer when no Anthropic key is set up; ErrBudgetSpent when the month's language-model
// tokens are spent before a batch, with what was checked before kept.
func (s *Service) CheckList(ctx context.Context) error {
	r, model, err := s.reviewer(ctx)
	if err != nil {
		return err
	}
	directory, err := s.directory.Companies(ctx)
	if err != nil {
		return fmt.Errorf("list the companies: %w", err)
	}
	checks, err := s.repo.ListChecks(ctx)
	if err != nil {
		return fmt.Errorf("the checks of the list: %w", err)
	}
	todo := domain.ToCheck(directory, checks)
	if len(todo) == 0 {
		return nil
	}
	quotes, err := s.namedIn(ctx, directory)
	if err != nil {
		return err
	}
	for len(todo) > 0 {
		if err := s.tokensLeft(ctx); err != nil {
			return err
		}
		batch := todo[:min(CheckBatch, len(todo))]
		todo = todo[len(batch):]
		if err := s.checkListBatch(ctx, r, model, batch, quotes); err != nil {
			return err
		}
	}
	return nil
}

// namedIn is, for each company on the list, the answers that name it.
func (s *Service) namedIn(ctx context.Context, directory []domain.DirectoryCompany) (map[string]domain.Quotes, error) {
	namings, err := s.repo.Namings(ctx)
	if err != nil {
		return nil, fmt.Errorf("the names the answers gave: %w", err)
	}
	decisions, err := s.repo.CandidateDecisions(ctx)
	if err != nil {
		return nil, fmt.Errorf("candidate decisions: %w", err)
	}
	merges, err := s.repo.Merges(ctx)
	if err != nil {
		return nil, fmt.Errorf("candidate merges: %w", err)
	}
	return domain.NewMatcher(directory, decisions, merges).NamedIn(namings), nil
}

// checkListBatch has the model check batch from each company's dossier, stores its checks and what
// asking used, and archives each company it is sure does not compete.
func (s *Service) checkListBatch(ctx context.Context, r Reviewer, model string, batch []domain.DirectoryCompany, quotes map[string]domain.Quotes) error {
	asked := make([]domain.Candidate, 0, len(batch))
	for _, c := range batch {
		asked = append(asked, domain.AsCandidate(c, quotes[c.ID]))
	}
	dossiers, err := s.dossiers(ctx, asked)
	if err != nil {
		return err
	}
	answered, usage, err := r.Check(ctx, dossiers)
	if saveErr := s.saveUsage(ctx, usage); err != nil || saveErr != nil {
		return errors.Join(wrap("check the companies on the list", err), saveErr)
	}
	checks := domain.ListChecksOf(domain.CompleteVerdicts(asked, answered, model, s.now()))
	if err := s.repo.SaveListChecks(ctx, checks); err != nil {
		return fmt.Errorf("store the checks of the list: %w", err)
	}
	for _, c := range checks {
		if !c.Archives() {
			continue
		}
		if _, err := s.directory.ArchiveUnlessKept(ctx, domain.ModelDecider, c.CompanyID); err != nil {
			return fmt.Errorf("archive %s: %w", c.CompanyID, err)
		}
	}
	return nil
}
