package app

import (
	"context"
	"fmt"

	"github.com/JorisJonkers-dev/findability-explorer/internal/insight/domain"
)

// Products are Tribelt's belt families against the field, what stands out among them, and the
// companies they name.
type Products struct {
	Products  []domain.Product
	Standouts []domain.Standout
	Companies []domain.Company
}

// Products compares each belt family from the companies' names for it, the terminology map and
// the newest Prompt run.
func (s *Service) Products(ctx context.Context) (Products, error) {
	companies, err := s.ports.Directory.Companies(ctx)
	if err != nil {
		return Products{}, fmt.Errorf("list the companies: %w", err)
	}
	terms, err := s.ports.Terms.MapTerms(ctx)
	if err != nil {
		return Products{}, fmt.Errorf("the terminology map: %w", err)
	}
	runs, err := s.ports.PromptRuns.Runs(ctx, 1)
	if err != nil {
		return Products{}, fmt.Errorf("the newest Prompt run: %w", err)
	}
	var run *domain.Run
	if len(runs) > 0 {
		run = &runs[0]
	}
	products := domain.Products(companies, terms, run)
	return Products{Products: products, Standouts: domain.Standouts(products), Companies: companies}, nil
}
