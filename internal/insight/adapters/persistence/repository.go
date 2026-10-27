// Package persistence implements the insight Store port on the sqlc-generated queries: the plan's
// ticks and targets.
package persistence

import (
	"context"
	"fmt"
	"time"

	"github.com/JorisJonkers-dev/findability-explorer/internal/insight/app"
	"github.com/JorisJonkers-dev/findability-explorer/internal/insight/domain"
	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/pg/queries"
)

// Repository maps rows to ticks and targets.
type Repository struct {
	q queries.Querier
}

var _ app.Store = (*Repository)(nil)

// New returns a Repository over q.
func New(q queries.Querier) *Repository {
	return &Repository{q: q}
}

// Ticks returns every improvement ticked off by hand.
func (r *Repository) Ticks(ctx context.Context) ([]domain.Tick, error) {
	rows, err := r.q.ListPlanTicks(ctx)
	if err != nil {
		return nil, fmt.Errorf("select plan ticks: %w", err)
	}
	out := make([]domain.Tick, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.Tick{ImprovementID: row.ImprovementID, Version: int(row.CatalogueVersion), By: row.TickedBy, At: row.TickedAt})
	}
	return out, nil
}

// Tick stores a tick, in place of an earlier one of the same improvement.
func (r *Repository) Tick(ctx context.Context, t domain.Tick) error {
	return r.q.TickPlanImprovement(ctx, queries.TickPlanImprovementParams{
		ImprovementID: t.ImprovementID, CatalogueVersion: int64(t.Version), TickedBy: t.By, TickedAt: t.At,
	})
}

// Untick deletes an improvement's tick; there being none is no error.
func (r *Repository) Untick(ctx context.Context, improvementID string) error {
	return r.q.UntickPlanImprovement(ctx, improvementID)
}

// Targets returns the targets an administrator set.
func (r *Repository) Targets(ctx context.Context) (map[domain.Target]float64, error) {
	rows, err := r.q.ListPlanTargets(ctx)
	if err != nil {
		return nil, fmt.Errorf("select plan targets: %w", err)
	}
	out := make(map[domain.Target]float64, len(rows))
	for _, row := range rows {
		out[domain.Target(row.Target)] = row.Value
	}
	return out, nil
}

// SetTarget stores a target, in place of an earlier value.
func (r *Repository) SetTarget(ctx context.Context, t domain.Target, value float64, by string, at time.Time) error {
	return r.q.SetPlanTarget(ctx, queries.SetPlanTargetParams{Target: string(t), Value: value, SetBy: by, SetAt: at})
}
