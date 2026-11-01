package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/JorisJonkers-dev/findability-explorer/internal/insight/domain"
)

// Plan is the plan as it stands: the improvements in order and which are done, the numbers to
// watch against their targets, and the competitors to keep an eye on.
type Plan struct {
	Items   []domain.PlanItem
	Watched []domain.Watched
	EyeOn   []domain.EyeOn
	// TermsFrom says where Tribelt's search terms come from.
	TermsFrom domain.TermsSource
}

// Plan reads the plan from the ticks, the targets, Tribelt's latest scan and the standings.
func (s *Service) Plan(ctx context.Context) (Plan, error) {
	companies, f, err := s.facts(ctx)
	if err != nil {
		return Plan{}, err
	}
	ticks, err := s.ports.Store.Ticks(ctx)
	if err != nil {
		return Plan{}, fmt.Errorf("the ticked improvements: %w", err)
	}
	targets, err := s.targets(ctx)
	if err != nil {
		return Plan{}, err
	}
	st := domain.Standings(companies, f)
	items, err := s.items(ctx, ticks, f, st)
	if err != nil {
		return Plan{}, err
	}
	return Plan{
		Items:     items,
		Watched:   domain.Watch(targets, f),
		EyeOn:     domain.KeepAnEyeOn(companies, f, st),
		TermsFrom: f.TermsFrom,
	}, nil
}

// items are the plan's items: the written-by-hand improvements, and the recommendations the runs
// wrote, best value first.
func (s *Service) items(ctx context.Context, ticks []domain.Tick, f domain.Facts, st []domain.Standing) ([]domain.PlanItem, error) {
	recs, err := s.ports.Store.Recommendations(ctx)
	if err != nil {
		return nil, fmt.Errorf("the written recommendations: %w", err)
	}
	runs, err := s.ports.Store.RecRuns(ctx, RecentRuns)
	if err != nil {
		return nil, fmt.Errorf("the past runs: %w", err)
	}
	return domain.WithWritten(domain.PlanItems(ticks, tribeltReadiness(f), st), recs, domain.NewestDone(runs), st), nil
}

// Tick marks an improvement only an administrator can check as done, or, with done false, as not
// done again; a recommendation a run wrote is ticked off by hand too. An improvement a scan checks
// is domain.ErrCheckedByScan.
func (s *Service) Tick(ctx context.Context, actor, id string, done bool) error {
	imp, ok := domain.ImprovementByID(id)
	if !ok {
		return s.tickWritten(ctx, actor, id, done)
	}
	if !imp.Manual() {
		return domain.ErrCheckedByScan
	}
	if !done {
		if err := s.ports.Store.Untick(ctx, id); err != nil {
			return fmt.Errorf("undo the tick: %w", err)
		}
		return nil
	}
	if err := s.ports.Store.Tick(ctx, domain.Tick{ImprovementID: id, Version: domain.ImprovementsVersion, By: actor, At: s.now()}); err != nil {
		return fmt.Errorf("record the tick: %w", err)
	}
	return nil
}

// SetTargets sets the targets given, each checked first; nothing is stored when one is refused.
func (s *Service) SetTargets(ctx context.Context, actor string, targets map[domain.Target]float64) error {
	for t, v := range targets {
		if _, err := domain.CheckTarget(t, v); err != nil {
			return err
		}
	}
	for _, t := range domain.Targets() {
		v, ok := targets[t]
		if !ok {
			continue
		}
		if err := s.ports.Store.SetTarget(ctx, t, v, actor, s.now()); err != nil {
			return fmt.Errorf("store the target: %w", err)
		}
	}
	return nil
}

// IsInputError reports whether err is the caller's to fix (422) rather than the service's.
func IsInputError(err error) bool {
	return errors.Is(err, domain.ErrInvalid)
}

// tickWritten ticks a recommendation a run wrote off, or undoes the tick.
func (s *Service) tickWritten(ctx context.Context, actor, id string, done bool) error {
	by := ""
	if done {
		by = actor
	}
	found, err := s.ports.Store.SetRecDone(ctx, id, by, s.now())
	if err != nil {
		return fmt.Errorf("record the tick: %w", err)
	}
	if !found {
		return domain.ErrUnknownImprovement
	}
	return nil
}
