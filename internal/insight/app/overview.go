package app

import (
	"context"
	"fmt"

	"github.com/JorisJonkers-dev/findability-explorer/internal/insight/domain"
)

// Overview counts the home page's figures from the companies, the terminology map and the newest
// Prompt run with answers that count.
func (s *Service) Overview(ctx context.Context) (domain.Overview, error) {
	companies, err := s.ports.Directory.Companies(ctx)
	if err != nil {
		return domain.Overview{}, fmt.Errorf("list the companies: %w", err)
	}
	terms, err := s.ports.Terms.TermMap(ctx)
	if err != nil {
		return domain.Overview{}, fmt.Errorf("the terminology map: %w", err)
	}
	runs, err := s.ports.PromptRuns.Runs(ctx, 1)
	if err != nil {
		return domain.Overview{}, fmt.Errorf("the newest Prompt run: %w", err)
	}
	var run *domain.Run
	if len(runs) > 0 {
		run = &runs[0]
	}
	return domain.OverviewOf(companies, terms, run), nil
}
