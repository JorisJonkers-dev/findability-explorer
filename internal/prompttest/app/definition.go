package app

import (
	"context"
	"errors"
	"fmt"

	llm "github.com/JorisJonkers-dev/findability-explorer/internal/llm/domain"
	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain"
)

// ErrUnchanged is returned for a definition saved or restored with the text already in use.
var ErrUnchanged = errors.New("the competitor definition in use has this text already")

// DefinitionView is the competitor definition in use with every version, the newest first, and
// what checking again against it would take: the competitors on the list and the waiting
// candidates checked against an older version, and what checking them again should cost.
type DefinitionView struct {
	Current             domain.Definition
	Versions            []domain.Definition
	OutdatedCompetitors int
	OutdatedCandidates  int
	RecheckCost         llm.Micros
}

// definition is the competitor definition in use: the newest version.
func (s *Service) definition(ctx context.Context) (domain.Definition, error) {
	_, d, err := s.definitions(ctx)
	return d, err
}

// definitions are every version of the competitor definition, the newest first, and the one in use.
func (s *Service) definitions(ctx context.Context) ([]domain.Definition, domain.Definition, error) {
	versions, err := s.repo.Definitions(ctx)
	if err != nil {
		return nil, domain.Definition{}, fmt.Errorf("the competitor definition: %w", err)
	}
	d, ok := domain.NewestDefinition(versions)
	if !ok {
		return nil, domain.Definition{}, fmt.Errorf("the competitor definition: %w", domain.ErrNotFound)
	}
	return versions, d, nil
}

// CompetitorDefinition is the definition in use, its versions, and what checking again would take.
func (s *Service) CompetitorDefinition(ctx context.Context) (DefinitionView, error) {
	versions, current, err := s.definitions(ctx)
	if err != nil {
		return DefinitionView{}, err
	}
	directory, err := s.directory.Companies(ctx)
	if err != nil {
		return DefinitionView{}, fmt.Errorf("list the companies: %w", err)
	}
	checks, err := s.repo.ListChecks(ctx)
	if err != nil {
		return DefinitionView{}, fmt.Errorf("the checks of the list: %w", err)
	}
	verdicts, err := s.repo.CompanyVerdicts(ctx)
	if err != nil {
		return DefinitionView{}, fmt.Errorf("candidate verdicts: %w", err)
	}
	all, err := s.candidatesOn(ctx, directory)
	if err != nil {
		return DefinitionView{}, err
	}
	used, checked, err := s.repo.CheckUsage(ctx)
	if err != nil {
		return DefinitionView{}, fmt.Errorf("what the checks used: %w", err)
	}
	competitors, candidates := domain.Outdated(directory, checks, all, verdicts, current.Version)
	return DefinitionView{
		Current: current, Versions: versions, OutdatedCompetitors: competitors, OutdatedCandidates: candidates,
		RecheckCost: domain.EstimateChecks(competitors+candidates, used, checked, s.checkModel(ctx)),
	}, nil
}

// CheckEstimate is what checking n companies should cost now on the Anthropic key: from what the
// checks so far used per company, at the model the Claude step checking companies uses now.
func (s *Service) CheckEstimate(ctx context.Context, n int) (llm.Micros, error) {
	used, checked, err := s.repo.CheckUsage(ctx)
	if err != nil {
		return 0, fmt.Errorf("what the checks used: %w", err)
	}
	return domain.EstimateChecks(n, used, checked, s.checkModel(ctx)), nil
}

// checkModel is the model the Claude step checking companies uses now, as the reviewer names it,
// for an estimate; Claude Haiku 5.5 when there is no reviewer to ask.
func (s *Service) checkModel(ctx context.Context) string {
	if _, model, err := s.reviewer(ctx); err == nil {
		return model
	}
	return llm.DefaultModel
}

// SaveDefinition stores text as a new version of the competitor definition, saved by user, and
// returns the view of it. It starts no check: what was checked against an older version is checked
// again by the next check. domain.ErrInvalid for an empty or too long text; ErrUnchanged for the
// text in use.
func (s *Service) SaveDefinition(ctx context.Context, user, text string) (DefinitionView, error) {
	text, err := domain.CheckDefinition(text)
	if err != nil {
		return DefinitionView{}, err
	}
	return s.addDefinition(ctx, user, text, 0)
}

// RestoreDefinition saves the text of version as a new version, restored by user. domain.ErrNotFound
// for an unknown version; ErrUnchanged when its text is the one in use.
func (s *Service) RestoreDefinition(ctx context.Context, user string, version int) (DefinitionView, error) {
	versions, _, err := s.definitions(ctx)
	if err != nil {
		return DefinitionView{}, err
	}
	d, ok := domain.DefinitionVersion(versions, version)
	if !ok {
		return DefinitionView{}, fmt.Errorf("version %d of the competitor definition: %w", version, domain.ErrNotFound)
	}
	return s.addDefinition(ctx, user, d.Text, version)
}

func (s *Service) addDefinition(ctx context.Context, user, text string, restoredFrom int) (DefinitionView, error) {
	current, err := s.definition(ctx)
	if err != nil {
		return DefinitionView{}, err
	}
	if current.Text == text {
		return DefinitionView{}, ErrUnchanged
	}
	if _, err := s.repo.AddDefinition(ctx, text, user, restoredFrom, s.now()); err != nil {
		return DefinitionView{}, fmt.Errorf("store the competitor definition: %w", err)
	}
	return s.CompetitorDefinition(ctx)
}
