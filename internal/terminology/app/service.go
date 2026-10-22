// Package app holds the terminology context's use cases: the only entry points the inbound adapters call.
package app

import (
	"github.com/JorisJonkers-dev/findability-explorer/internal/terminology/domain"
)

const (
	// DefaultLimit is how many terms Search returns when the caller names no limit.
	DefaultLimit = 50
	// MaxLimit is the most terms one Search returns.
	MaxLimit = 200
)

// Catalogue is the read-only terminology map the use cases search. The seed adapter implements it.
type Catalogue interface {
	Terms() []domain.Term
	Companies() []domain.Company
}

// Service runs the terminology use cases against a catalogue.
type Service struct {
	catalogue Catalogue
}

// New returns a Service reading catalogue.
func New(catalogue Catalogue) *Service {
	return &Service{catalogue: catalogue}
}

// Search returns the terms that match q. A limit below 1 means DefaultLimit; above MaxLimit, MaxLimit.
func (s *Service) Search(q domain.Query) domain.Result {
	switch {
	case q.Limit < 1:
		q.Limit = DefaultLimit
	case q.Limit > MaxLimit:
		q.Limit = MaxLimit
	}
	return domain.Search(s.catalogue.Terms(), q)
}

// Companies returns every company in the map, competitors first, by name.
func (s *Service) Companies() []domain.Company {
	return domain.SortCompanies(s.catalogue.Companies())
}

// Terms returns the whole map, for other contexts that match text against it.
func (s *Service) Terms() []domain.Term {
	return s.catalogue.Terms()
}
