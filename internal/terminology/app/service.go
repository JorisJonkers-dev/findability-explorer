// Package app holds the terminology context's use cases: searching the map, and the review queue
// through which phrases scans found join it. They are the only entry points the inbound adapters
// call.
package app

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/JorisJonkers-dev/findability-explorer/internal/terminology/domain"
)

const (
	// DefaultLimit is how many terms Search returns when the caller names no limit.
	DefaultLimit = 50
	// MaxLimit is the most terms one Search returns.
	MaxLimit = 200
)

// Catalogue is the read-only terminology map of the desk research. The seed adapter implements it.
type Catalogue interface {
	Terms() []domain.Term
	Companies() []domain.Company
}

// Reviews stores the administrators' decisions on candidates and the terms accepted ones became.
type Reviews interface {
	// Decisions returns every decision, an accepted one with its term.
	Decisions(ctx context.Context) ([]domain.Decision, error)
	// Decide stores d, with its term when it is accepted, or returns domain.ErrDecided when the
	// candidate was decided meanwhile.
	Decide(ctx context.Context, d domain.Decision) error
	// Undo deletes the decision on a candidate, and its term; it reports whether there was one.
	Undo(ctx context.Context, candidateID string) (bool, error)
}

// Evidence returns the phrases finished scans found outside the map; the analysis context
// provides it.
type Evidence func(ctx context.Context) ([]domain.Observation, error)

// Service runs the terminology use cases against the catalogue and the accepted terms.
type Service struct {
	catalogue Catalogue
	reviews   Reviews
	evidence  Evidence
	now       func() time.Time

	mu       sync.RWMutex
	accepted []domain.Term
}

// New returns a Service reading catalogue and, through reviews, the terms accepted from the
// review queue. Call Load before serving.
func New(catalogue Catalogue, reviews Reviews, evidence Evidence, now func() time.Time) *Service {
	return &Service{catalogue: catalogue, reviews: reviews, evidence: evidence, now: now}
}

// Load reads the accepted terms into the map.
func (s *Service) Load(ctx context.Context) error {
	decisions, err := s.reviews.Decisions(ctx)
	if err != nil {
		return fmt.Errorf("load decisions: %w", err)
	}
	var accepted []domain.Term
	for _, d := range decisions {
		if d.Term != nil {
			accepted = append(accepted, *d.Term)
		}
	}
	s.mu.Lock()
	s.accepted = accepted
	s.mu.Unlock()
	return nil
}

// Search returns the terms that match q. A limit below 1 means DefaultLimit; above MaxLimit, MaxLimit.
func (s *Service) Search(q domain.Query) domain.Result {
	switch {
	case q.Limit < 1:
		q.Limit = DefaultLimit
	case q.Limit > MaxLimit:
		q.Limit = MaxLimit
	}
	return domain.Search(s.Terms(), q)
}

// Companies returns every company in the map, competitors first, by name.
func (s *Service) Companies() []domain.Company {
	return domain.SortCompanies(s.catalogue.Companies())
}

// Terms returns the whole map: the research's terms, then the accepted ones, for search and for
// other contexts that match text against it.
func (s *Service) Terms() []domain.Term {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return slices.Concat(s.catalogue.Terms(), s.accepted)
}

// Candidates returns the candidates with status, and how many have each status.
func (s *Service) Candidates(ctx context.Context, status domain.Status) ([]domain.Candidate, domain.Counts, error) {
	all, err := s.candidates(ctx)
	if err != nil {
		return nil, domain.Counts{}, err
	}
	list, counts := domain.Count(all, status)
	return list, counts, nil
}

func (s *Service) candidates(ctx context.Context) ([]domain.Candidate, error) {
	obs, err := s.evidence(ctx)
	if err != nil {
		return nil, fmt.Errorf("read the scans' findings: %w", err)
	}
	decisions, err := s.reviews.Decisions(ctx)
	if err != nil {
		return nil, fmt.Errorf("load decisions: %w", err)
	}
	return domain.Candidates(obs, decisions, s.Terms()), nil
}

// candidate returns the candidate with id, or domain.ErrCandidateNotFound.
func (s *Service) candidate(ctx context.Context, id string) (domain.Candidate, error) {
	all, err := s.candidates(ctx)
	if err != nil {
		return domain.Candidate{}, err
	}
	i := slices.IndexFunc(all, func(c domain.Candidate) bool { return c.ID == id })
	if i < 0 {
		return domain.Candidate{}, domain.ErrCandidateNotFound
	}
	return all[i], nil
}

// Accept adds the term a pending candidate becomes to the map, recording user as its reviewer.
func (s *Service) Accept(ctx context.Context, user, id string, a domain.Acceptance) (domain.Candidate, error) {
	c, err := s.candidate(ctx, id)
	if err != nil {
		return domain.Candidate{}, err
	}
	if c.Status != domain.StatusPending {
		return domain.Candidate{}, domain.ErrDecided
	}
	term, err := domain.NewTerm(a, c.Source, s.Terms())
	if err != nil {
		return domain.Candidate{}, err
	}
	return s.decide(ctx, domain.Decision{
		CandidateID: id, Phrase: c.Phrase, Status: domain.StatusAccepted, By: user, At: s.now(), Term: &term,
	})
}

// Reject marks a pending candidate rejected, recording user as its reviewer.
func (s *Service) Reject(ctx context.Context, user, id string) (domain.Candidate, error) {
	c, err := s.candidate(ctx, id)
	if err != nil {
		return domain.Candidate{}, err
	}
	if c.Status != domain.StatusPending {
		return domain.Candidate{}, domain.ErrDecided
	}
	return s.decide(ctx, domain.Decision{CandidateID: id, Phrase: c.Phrase, Status: domain.StatusRejected, By: user, At: s.now()})
}

func (s *Service) decide(ctx context.Context, d domain.Decision) (domain.Candidate, error) {
	if err := s.reviews.Decide(ctx, d); err != nil {
		return domain.Candidate{}, err
	}
	if err := s.Load(ctx); err != nil {
		return domain.Candidate{}, err
	}
	return s.candidate(ctx, d.CandidateID)
}

// Undo puts a decided candidate back in the queue; an accepted term leaves the map again.
func (s *Service) Undo(ctx context.Context, id string) (domain.Candidate, error) {
	c, err := s.candidate(ctx, id)
	if err != nil {
		return domain.Candidate{}, err
	}
	if c.Status == domain.StatusPending {
		return domain.Candidate{}, domain.ErrPending
	}
	if _, err := s.reviews.Undo(ctx, id); err != nil {
		return domain.Candidate{}, fmt.Errorf("undo decision: %w", err)
	}
	if err := s.Load(ctx); err != nil {
		return domain.Candidate{}, err
	}
	return s.candidate(ctx, id)
}
