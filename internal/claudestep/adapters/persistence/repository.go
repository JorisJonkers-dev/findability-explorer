// Package persistence implements the Claude steps' Repository on the sqlc-generated queries.
package persistence

import (
	"context"
	"fmt"

	"github.com/JorisJonkers-dev/findability-explorer/internal/claudestep/app"
	"github.com/JorisJonkers-dev/findability-explorer/internal/claudestep/domain"
	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/pg/queries"
)

// Store is what the repository needs from the Postgres adapter.
type Store interface {
	Queries() *queries.Queries
	InTx(ctx context.Context, fn func(*queries.Queries) error) error
}

// Repository stores the steps' versions.
type Repository struct {
	store Store
}

var _ app.Repository = (*Repository)(nil)

// New returns a Repository over store.
func New(store Store) *Repository { return &Repository{store: store} }

// Versions returns a step's saved versions, oldest first.
func (r *Repository) Versions(ctx context.Context, step domain.ID) ([]domain.Version, error) {
	rows, err := r.store.Queries().ClaudeStepVersions(ctx, string(step))
	if err != nil {
		return nil, fmt.Errorf("select versions: %w", err)
	}
	out := make([]domain.Version, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.Version{
			Step: domain.ID(row.Step), Number: int(row.Number), Model: row.Model, Instructions: row.Instructions,
			Note: row.Note, By: row.CreatedBy, At: row.CreatedAt.UTC(),
		})
	}
	return out, nil
}

// Save stores v, and the step's original before it when given, in one transaction.
func (r *Repository) Save(ctx context.Context, v domain.Version, original *domain.Version) error {
	return r.store.InTx(ctx, func(q *queries.Queries) error {
		for _, x := range []*domain.Version{original, &v} {
			if x == nil {
				continue
			}
			if err := q.AddClaudeStepVersion(ctx, queries.AddClaudeStepVersionParams{
				Step: string(x.Step), Number: int64(x.Number), Model: x.Model, Instructions: x.Instructions,
				Note: x.Note, CreatedBy: x.By, CreatedAt: x.At,
			}); err != nil {
				return fmt.Errorf("insert version %d: %w", x.Number, err)
			}
		}
		return nil
	})
}
