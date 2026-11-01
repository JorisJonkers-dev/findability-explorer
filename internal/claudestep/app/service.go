// Package app holds the Claude steps' use cases: listing the steps, reading one with its versions
// and the models it may use, saving a new version, and handing a call site the version it uses.
package app

import (
	"context"
	"fmt"
	"time"

	"github.com/JorisJonkers-dev/findability-explorer/internal/claudestep/domain"
	llm "github.com/JorisJonkers-dev/findability-explorer/internal/llm/domain"
)

// Repository stores the saved versions of the steps.
type Repository interface {
	// Versions returns a step's saved versions, oldest first; none until the first save.
	Versions(ctx context.Context, step domain.ID) ([]domain.Version, error)
	// Save stores v. The first save of a step stores its original, version 1, before it, in the
	// same transaction, so the history keeps it.
	Save(ctx context.Context, v domain.Version, original *domain.Version) error
}

// Base is the tool's own estimate of one job of a step with Claude Haiku 5.5, such as the average
// scan's matching; 0 when there is none yet.
type Base func(ctx context.Context, step domain.ID) (llm.Micros, error)

// Model is a Claude model a step may use, with what one job of the step costs with it.
type Model struct {
	Info    llm.ClaudeInfo
	JobCost llm.Micros
}

// Step is one step as the pages show it.
type Step struct {
	Definition domain.Definition
	Current    domain.Version
	// Versions are every version, oldest first: the original, then the saved ones.
	Versions []domain.Version
	// Models are the Claude models it may use, cheapest first, each with its job's cost.
	Models []Model
	// JobCost is what one job costs with the current model.
	JobCost llm.Micros
}

// Service runs the Claude steps' use cases.
type Service struct {
	repo Repository
	base Base
	now  func() time.Time
}

// New returns a Service storing versions in repo and estimating jobs from base.
func New(repo Repository, base Base, now func() time.Time) *Service {
	return &Service{repo: repo, base: base, now: now}
}

// Steps returns every step with its current version and job cost.
func (s *Service) Steps(ctx context.Context) ([]Step, error) {
	defs := domain.Definitions()
	out := make([]Step, 0, len(defs))
	for _, d := range defs {
		st, err := s.step(ctx, d)
		if err != nil {
			return nil, err
		}
		out = append(out, st)
	}
	return out, nil
}

// Step returns one step, or domain.ErrNotFound.
func (s *Service) Step(ctx context.Context, id domain.ID) (Step, error) {
	d, ok := domain.Lookup(id)
	if !ok {
		return Step{}, domain.ErrNotFound
	}
	return s.step(ctx, d)
}

// Current returns the version a step's next job uses: model and instructions.
func (s *Service) Current(ctx context.Context, id domain.ID) (domain.Version, error) {
	d, ok := domain.Lookup(id)
	if !ok {
		return domain.Version{}, domain.ErrNotFound
	}
	versions, err := s.repo.Versions(ctx, id)
	if err != nil {
		return domain.Version{}, fmt.Errorf("read the versions of %s: %w", id, err)
	}
	return domain.CurrentOf(d, versions), nil
}

// Save stores a new version of a step, which its next job uses, and returns the step with it;
// domain.ErrNotFound for an unknown step, domain.ErrInvalid for a version that cannot be saved.
func (s *Service) Save(ctx context.Context, id domain.ID, actor string, ch domain.Change) (Step, error) {
	d, ok := domain.Lookup(id)
	if !ok {
		return Step{}, domain.ErrNotFound
	}
	versions, err := s.repo.Versions(ctx, id)
	if err != nil {
		return Step{}, fmt.Errorf("read the versions of %s: %w", id, err)
	}
	next, err := domain.Next(d, domain.CurrentOf(d, versions), ch, actor, s.now())
	if err != nil {
		return Step{}, err
	}
	var original *domain.Version
	if len(versions) == 0 {
		o := domain.Original(d)
		o.At = next.At
		original = &o
	}
	if err := s.repo.Save(ctx, next, original); err != nil {
		return Step{}, fmt.Errorf("save version %d of %s: %w", next.Number, id, err)
	}
	return s.step(ctx, d)
}

func (s *Service) step(ctx context.Context, d domain.Definition) (Step, error) {
	versions, err := s.repo.Versions(ctx, d.ID)
	if err != nil {
		return Step{}, fmt.Errorf("read the versions of %s: %w", d.ID, err)
	}
	base, err := s.base(ctx, d.ID)
	if err != nil {
		return Step{}, fmt.Errorf("estimate %s: %w", d.ID, err)
	}
	current := domain.CurrentOf(d, versions)
	if len(versions) == 0 {
		versions = []domain.Version{current}
	}
	claude := llm.ClaudeModels()
	models := make([]Model, 0, len(claude))
	for _, m := range claude {
		models = append(models, Model{Info: m, JobCost: domain.JobCost(d, base, m.ID)})
	}
	return Step{Definition: d, Current: current, Versions: versions, Models: models, JobCost: domain.JobCost(d, base, current.Model)}, nil
}
