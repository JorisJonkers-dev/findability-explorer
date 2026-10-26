package app

import (
	"context"
	"fmt"
	"slices"

	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain"
)

// Candidates lists the competitor candidates with status, or all of them for "", with how many
// there are of each status. It follows the terminology review queue's pattern (docs/adr/0003).
func (s *Service) Candidates(ctx context.Context, status domain.CandidateStatus) ([]domain.Candidate, map[domain.CandidateStatus]int, error) {
	all, err := s.candidates(ctx)
	if err != nil {
		return nil, nil, err
	}
	counts := map[domain.CandidateStatus]int{domain.CandidatePending: 0, domain.CandidateAdded: 0, domain.CandidateDismissed: 0}
	out := []domain.Candidate{}
	for _, c := range all {
		counts[c.Status()]++
		if status == "" || c.Status() == status {
			out = append(out, c)
		}
	}
	return out, counts, nil
}

func (s *Service) candidates(ctx context.Context) ([]domain.Candidate, error) {
	sightings, err := s.repo.Sightings(ctx)
	if err != nil {
		return nil, fmt.Errorf("unmatched names: %w", err)
	}
	decisions, err := s.repo.CandidateDecisions(ctx)
	if err != nil {
		return nil, fmt.Errorf("candidate decisions: %w", err)
	}
	directory, err := s.directory.Companies(ctx)
	if err != nil {
		return nil, fmt.Errorf("list the companies: %w", err)
	}
	return domain.Candidates(sightings, decisions, directory), nil
}

// candidate returns the pending candidate with id, domain.ErrNotFound, or ErrDecided.
func (s *Service) candidate(ctx context.Context, id string) (domain.Candidate, error) {
	all, err := s.candidates(ctx)
	if err != nil {
		return domain.Candidate{}, err
	}
	i := slices.IndexFunc(all, func(c domain.Candidate) bool { return c.ID == id })
	switch {
	case i < 0:
		return domain.Candidate{}, domain.ErrNotFound
	case all[i].Decision != nil:
		return domain.Candidate{}, ErrDecided
	}
	return all[i], nil
}

// ErrDecided is returned for a decision on a candidate that was decided already; undo it first.
var ErrDecided = fmt.Errorf("%w: the candidate was decided already; undo that first", domain.ErrInvalid)

// AddCandidate adds a pending candidate to the competitor list as an added company with name and
// website, which can then be scanned, and records the decision.
func (s *Service) AddCandidate(ctx context.Context, actor, id, name, website string) (domain.Candidate, error) {
	c, err := s.candidate(ctx, id)
	if err != nil {
		return domain.Candidate{}, err
	}
	competitorID, err := s.directory.Add(ctx, actor, name, website)
	if err != nil {
		return domain.Candidate{}, err
	}
	d := domain.CandidateDecision{ID: id, Name: c.Name, Status: domain.CandidateAdded, CompetitorID: competitorID, By: actor, At: s.now()}
	if err := s.repo.Decide(ctx, d); err != nil {
		return domain.Candidate{}, fmt.Errorf("record the decision: %w", err)
	}
	c.Decision = &d
	return c, nil
}

// DismissCandidate dismisses a pending candidate; it stays dismissed in later runs.
func (s *Service) DismissCandidate(ctx context.Context, actor, id string) (domain.Candidate, error) {
	c, err := s.candidate(ctx, id)
	if err != nil {
		return domain.Candidate{}, err
	}
	d := domain.CandidateDecision{ID: id, Name: c.Name, Status: domain.CandidateDismissed, By: actor, At: s.now()}
	if err := s.repo.Decide(ctx, d); err != nil {
		return domain.Candidate{}, fmt.Errorf("record the decision: %w", err)
	}
	c.Decision = &d
	return c, nil
}

// UndoCandidate undoes a decision: an added candidate's company is removed again, with its
// scans; the candidate is pending once more.
func (s *Service) UndoCandidate(ctx context.Context, actor, id string) error {
	decisions, err := s.repo.CandidateDecisions(ctx)
	if err != nil {
		return fmt.Errorf("candidate decisions: %w", err)
	}
	i := slices.IndexFunc(decisions, func(d domain.CandidateDecision) bool { return d.ID == id })
	if i < 0 {
		return domain.ErrNotFound
	}
	if d := decisions[i]; d.Status == domain.CandidateAdded {
		if err := s.directory.Remove(ctx, actor, d.CompetitorID); err != nil {
			return err
		}
	}
	return s.repo.Undecide(ctx, id)
}
