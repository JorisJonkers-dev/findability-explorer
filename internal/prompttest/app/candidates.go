package app

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain"
)

// CandidateCounts is how many candidates have each status, and how many waiting ones the model has
// not checked.
type CandidateCounts struct {
	Pending, Added, Dismissed, Unchecked int
}

// Candidates lists the competitor candidates with status, or all of them for "", with the counts.
// Names on one website, and names merged, are one candidate.
func (s *Service) Candidates(ctx context.Context, status domain.CandidateStatus) ([]domain.Candidate, CandidateCounts, error) {
	all, err := s.candidates(ctx)
	if err != nil {
		return nil, CandidateCounts{}, err
	}
	var counts CandidateCounts
	out := []domain.Candidate{}
	for _, c := range all {
		switch c.Status() {
		case domain.CandidatePending:
			counts.Pending++
		case domain.CandidateAdded:
			counts.Added++
		case domain.CandidateDismissed:
			counts.Dismissed++
		}
		if status == "" || c.Status() == status {
			out = append(out, c)
		}
	}
	counts.Unchecked = len(domain.Unchecked(all))
	return out, counts, nil
}

func (s *Service) candidates(ctx context.Context) ([]domain.Candidate, error) {
	directory, err := s.directory.Companies(ctx)
	if err != nil {
		return nil, fmt.Errorf("list the companies: %w", err)
	}
	return s.candidatesOn(ctx, directory)
}

// candidatesOn are the candidates against directory, with the model's verdicts.
func (s *Service) candidatesOn(ctx context.Context, directory []domain.DirectoryCompany) ([]domain.Candidate, error) {
	sightings, err := s.repo.Sightings(ctx)
	if err != nil {
		return nil, fmt.Errorf("unmatched names: %w", err)
	}
	decisions, err := s.repo.CandidateDecisions(ctx)
	if err != nil {
		return nil, fmt.Errorf("candidate decisions: %w", err)
	}
	merges, err := s.repo.Merges(ctx)
	if err != nil {
		return nil, fmt.Errorf("candidate merges: %w", err)
	}
	verdicts, err := s.repo.CompanyVerdicts(ctx)
	if err != nil {
		return nil, fmt.Errorf("candidate verdicts: %w", err)
	}
	return domain.WithVerdicts(domain.Candidates(sightings, decisions, directory, merges), verdicts), nil
}

// find is the candidate one of whose names has id.
func find(all []domain.Candidate, id string) (domain.Candidate, bool) {
	i := slices.IndexFunc(all, func(c domain.Candidate) bool { return slices.Contains(c.MemberIDs(), id) })
	if i < 0 {
		return domain.Candidate{}, false
	}
	return all[i], true
}

// candidate returns the pending candidate one of whose names has id, domain.ErrNotFound, or ErrDecided.
func (s *Service) candidate(ctx context.Context, id string) (domain.Candidate, error) {
	all, err := s.candidates(ctx)
	if err != nil {
		return domain.Candidate{}, err
	}
	return pending(all, id)
}

func pending(all []domain.Candidate, id string) (domain.Candidate, error) {
	c, ok := find(all, id)
	switch {
	case !ok:
		return domain.Candidate{}, domain.ErrNotFound
	case c.Decision != nil:
		return domain.Candidate{}, ErrDecided
	}
	return c, nil
}

// ErrDecided is returned for a decision on a candidate that was decided already; undo it first.
var ErrDecided = fmt.Errorf("%w: the candidate was decided already; undo that first", domain.ErrInvalid)

// decide records a decision on every name of a candidate, and returns it decided.
func (s *Service) decide(ctx context.Context, c domain.Candidate, status domain.CandidateStatus, competitorID, actor string) (domain.Candidate, error) {
	d := domain.CandidateDecision{ID: c.ID, Name: c.Name, Status: status, CompetitorID: competitorID, By: actor, At: s.now()}
	for _, m := range c.Members {
		md := d
		md.ID, md.Name = m.ID, m.Name
		if err := s.repo.Decide(ctx, md); err != nil {
			return domain.Candidate{}, fmt.Errorf("record the decision: %w", err)
		}
	}
	c.Decision = &d
	return c, nil
}

// AddCandidate adds a pending candidate to the competitor list as an added company with name and
// website, which can then be scanned, and records the decision on each of its names.
func (s *Service) AddCandidate(ctx context.Context, actor, id, name, website string) (domain.Candidate, error) {
	c, err := s.candidate(ctx, id)
	if err != nil {
		return domain.Candidate{}, err
	}
	competitorID, err := s.directory.Add(ctx, actor, name, website)
	if err != nil {
		return domain.Candidate{}, err
	}
	return s.decide(ctx, c, domain.CandidateAdded, competitorID, actor)
}

// DismissCandidate dismisses a pending candidate, every name of it; it stays dismissed in later runs.
func (s *Service) DismissCandidate(ctx context.Context, actor, id string) (domain.Candidate, error) {
	c, err := s.candidate(ctx, id)
	if err != nil {
		return domain.Candidate{}, err
	}
	return s.decide(ctx, c, domain.CandidateDismissed, "", actor)
}

// UndoCandidate undoes the decision on a candidate, on every name of it: an added candidate's
// company is removed again, with its scans; the candidate is pending once more.
func (s *Service) UndoCandidate(ctx context.Context, actor, id string) error {
	decisions, err := s.repo.CandidateDecisions(ctx)
	if err != nil {
		return fmt.Errorf("candidate decisions: %w", err)
	}
	all, err := s.candidates(ctx)
	if err != nil {
		return err
	}
	ids := []string{id}
	if c, ok := find(all, id); ok {
		ids = c.MemberIDs()
	}
	mine := slices.DeleteFunc(decisions, func(d domain.CandidateDecision) bool { return !slices.Contains(ids, d.ID) })
	if len(mine) == 0 {
		return domain.ErrNotFound
	}
	var removed []string
	for _, d := range mine {
		if d.Status == domain.CandidateAdded && !slices.Contains(removed, d.CompetitorID) {
			if err := s.directory.Remove(ctx, actor, d.CompetitorID); err != nil {
				return err
			}
			removed = append(removed, d.CompetitorID)
		}
	}
	for _, d := range mine {
		if err := s.repo.Undecide(ctx, d.ID); err != nil {
			return err
		}
	}
	return nil
}

// RecordAdded records a pending candidate as added, as the company competitorID that was added
// another way: from a list of several at once. It returns domain.ErrNotFound for a name the
// answers never gave, and ErrDecided for a candidate decided already.
func (s *Service) RecordAdded(ctx context.Context, actor, id, competitorID string) error {
	// Without the company list: the company just added would hide the candidate.
	all, err := s.candidatesOn(ctx, nil)
	if err != nil {
		return err
	}
	c, err := pending(all, id)
	if err != nil {
		return err
	}
	_, err = s.decide(ctx, c, domain.CandidateAdded, competitorID, actor)
	return err
}

// Outcome is what deciding on one of several candidates did.
type Outcome string

const (
	OutcomeDone    Outcome = "done"
	OutcomeSkipped Outcome = "skipped"
)

// Decided is one candidate of several decided at once.
type Decided struct {
	ID      string
	Name    string
	Outcome Outcome
	// Reason says why one was skipped.
	Reason string
}

// ErrUnknownAction is returned for an action on several candidates other than add or dismiss.
var ErrUnknownAction = fmt.Errorf("%w: the action is add or dismiss", domain.ErrInvalid)

// DecideMany adds (status added) or dismisses (status dismissed) several pending candidates at
// once. One added goes on the list with its website; one without a website, one decided already
// or gone, and one the list refuses are skipped with the reason.
func (s *Service) DecideMany(ctx context.Context, actor string, status domain.CandidateStatus, ids []string) ([]Decided, error) {
	if status != domain.CandidateAdded && status != domain.CandidateDismissed {
		return nil, ErrUnknownAction
	}
	all, err := s.candidates(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Decided, 0, len(ids))
	var done []string
	for _, id := range ids {
		c, err := pending(all, id)
		switch {
		case err != nil:
			out = append(out, Decided{ID: id, Outcome: OutcomeSkipped, Reason: "It was decided already, or the answers no longer give it."})
			continue
		case slices.Contains(done, c.ID):
			continue
		}
		d, err := s.decideOne(ctx, actor, status, c)
		if err != nil {
			return out, err
		}
		out = append(out, d)
		done = append(done, c.ID)
	}
	return out, nil
}

func (s *Service) decideOne(ctx context.Context, actor string, status domain.CandidateStatus, c domain.Candidate) (Decided, error) {
	r := Decided{ID: c.ID, Name: c.Name, Outcome: OutcomeDone}
	competitorID := ""
	if status == domain.CandidateAdded {
		if c.Website == "" {
			r.Outcome, r.Reason = OutcomeSkipped, "The answers gave no website for it; add it on its own."
			return r, nil
		}
		id, err := s.directory.Add(ctx, actor, c.Name, c.Website)
		if errors.Is(err, domain.ErrInvalid) || errors.Is(err, domain.ErrCompanyKnown) {
			r.Outcome, r.Reason = OutcomeSkipped, err.Error()
			return r, nil
		}
		if err != nil {
			return r, err
		}
		competitorID = id
	}
	_, err := s.decide(ctx, c, status, competitorID, actor)
	return r, err
}

// ErrNotMerged is returned when splitting off a name that was not merged into the candidate.
var ErrNotMerged = fmt.Errorf("%w: only a name merged into a company can be split off", domain.ErrInvalid)

// Split takes a name the language model or a person merged into a candidate out of it again; it
// is not merged again. Names on one website stay one company.
func (s *Service) Split(ctx context.Context, actor, id string) error {
	all, err := s.candidates(ctx)
	if err != nil {
		return err
	}
	c, ok := find(all, id)
	if !ok {
		return domain.ErrNotFound
	}
	if i := slices.IndexFunc(c.Members, func(m domain.Member) bool { return m.ID == id }); c.Members[i].Joined != domain.JoinedByModel {
		return ErrNotMerged
	}
	return s.repo.SaveMerges(ctx, []domain.Merge{{CandidateID: id, Separate: true, By: actor, At: s.now()}})
}

// matcher matches answers' names to the list as it is now; with the list.
func (s *Service) matcher(ctx context.Context) (domain.Matcher, []domain.DirectoryCompany, error) {
	directory, err := s.directory.Companies(ctx)
	if err != nil {
		return domain.Matcher{}, nil, fmt.Errorf("list the companies: %w", err)
	}
	decisions, err := s.repo.CandidateDecisions(ctx)
	if err != nil {
		return domain.Matcher{}, nil, fmt.Errorf("candidate decisions: %w", err)
	}
	merges, err := s.repo.Merges(ctx)
	if err != nil {
		return domain.Matcher{}, nil, fmt.Errorf("candidate merges: %w", err)
	}
	return domain.NewMatcher(directory, decisions, merges), directory, nil
}

// rematched are the answers with each named company matched to the list as it is now.
func rematched(answers []domain.Answer, m domain.Matcher) []domain.Answer {
	for i, a := range answers {
		if a.Extraction != nil {
			x := m.Rematch(*a.Extraction)
			answers[i].Extraction = &x
		}
	}
	return answers
}
