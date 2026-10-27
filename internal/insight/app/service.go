// Package app holds the insight context's use cases: the standings (Catching up), the products
// and the plan. They compose what the other contexts own, reached only through the ports below,
// and are the only entry points the inbound adapters call.
package app

import (
	"context"
	"fmt"
	"time"

	"github.com/JorisJonkers-dev/findability-explorer/internal/insight/domain"
)

// Directory is the port to the companies context: every company that is not archived.
type Directory interface {
	Companies(ctx context.Context) ([]domain.Company, error)
}

// PromptRuns is the port to the prompt-test context: its newest runs with answers that count,
// newest first, each with only those answers.
type PromptRuns interface {
	Runs(ctx context.Context, limit int) ([]domain.Run, error)
}

// Readiness is the port to the analysis context: each company's newest AI readiness, by id.
type Readiness interface {
	Readiness(ctx context.Context) (map[string]domain.Readiness, error)
}

// Terms is the port to the terminology context: how many map terms each company's newest scan
// found, by id, for every company a scan read.
type Terms interface {
	TermsUsed(ctx context.Context) (map[string]int, error)
	// MapTerms are the terms of the map, each with how many companies use it and whether
	// Tribelt's site does.
	MapTerms(ctx context.Context) ([]domain.MapTerm, error)
}

// Ports are the other contexts the insight context reads.
type Ports struct {
	Directory  Directory
	PromptRuns PromptRuns
	Readiness  Readiness
	Terms      Terms
}

// Service runs the insight use cases.
type Service struct {
	ports Ports
	now   func() time.Time
}

// New returns a Service.
func New(ports Ports, now func() time.Time) *Service {
	return &Service{ports: ports, now: now}
}

// targets are the plan's targets now.
func (s *Service) targets() map[domain.Target]float64 {
	return domain.DefaultTargets()
}

// Standings is how Tribelt stands against its competitors now.
type Standings struct {
	Standings []domain.Standing
	// Trailing is how many measures Tribelt trails the leader on.
	Trailing int
	// Run is the Prompt run the AI measures come from; nil before any.
	Run       *domain.Run
	Companies []domain.Company
}

// facts gathers what the other contexts know now.
func (s *Service) facts(ctx context.Context) ([]domain.Company, domain.Facts, error) {
	companies, err := s.ports.Directory.Companies(ctx)
	if err != nil {
		return nil, domain.Facts{}, fmt.Errorf("list the companies: %w", err)
	}
	runs, err := s.ports.PromptRuns.Runs(ctx, 1)
	if err != nil {
		return nil, domain.Facts{}, fmt.Errorf("the newest Prompt run: %w", err)
	}
	readiness, err := s.ports.Readiness.Readiness(ctx)
	if err != nil {
		return nil, domain.Facts{}, fmt.Errorf("the AI readiness: %w", err)
	}
	terms, err := s.ports.Terms.TermsUsed(ctx)
	if err != nil {
		return nil, domain.Facts{}, fmt.Errorf("the terms used: %w", err)
	}
	f := domain.Facts{Readiness: readiness, TermsUsed: terms}
	if len(runs) > 0 {
		f.Run = &runs[0]
	}
	return companies, f, nil
}

// Standings compares Tribelt with its competitors on every measure, from the newest data.
func (s *Service) Standings(ctx context.Context) (Standings, error) {
	companies, f, err := s.facts(ctx)
	if err != nil {
		return Standings{}, err
	}
	st := domain.Standings(companies, f)
	return Standings{Standings: st, Trailing: domain.Trailing(st), Run: f.Run, Companies: companies}, nil
}
