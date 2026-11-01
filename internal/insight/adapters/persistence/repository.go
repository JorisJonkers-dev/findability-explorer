// Package persistence implements the insight Store port on the sqlc-generated queries: the plan's
// ticks and targets.
package persistence

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/JorisJonkers-dev/findability-explorer/internal/insight/app"
	"github.com/JorisJonkers-dev/findability-explorer/internal/insight/domain"
	llm "github.com/JorisJonkers-dev/findability-explorer/internal/llm/domain"
	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/pg/queries"
)

// Store is what the repository needs from the Postgres adapter.
type Store interface {
	Queries() *queries.Queries
	InTx(ctx context.Context, fn func(*queries.Queries) error) error
}

// Repository maps rows to ticks, targets, Recommendation runs and the recommendations they wrote.
type Repository struct {
	q     queries.Querier
	store Store
}

var _ app.Store = (*Repository)(nil)

// New returns a Repository over store.
func New(store Store) *Repository {
	return &Repository{q: store.Queries(), store: store}
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

// Recommendations returns every recommendation the runs wrote that is kept, the oldest first.
func (r *Repository) Recommendations(ctx context.Context) ([]domain.Recommendation, error) {
	rows, err := r.q.ListRecommendations(ctx)
	if err != nil {
		return nil, fmt.Errorf("select recommendations: %w", err)
	}
	out := make([]domain.Recommendation, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.Recommendation{
			ID: row.ID, RunID: row.RunID.String(),
			Gap: domain.Gap{
				Key: row.GapKey, Kind: domain.GapKind(row.Kind), Language: row.Language, Family: row.Family, Specifics: row.Specifics,
				Benefit: int(row.RuleBenefit), Effort: int(row.RuleEffort),
			},
			Title: row.Title, Where: row.WhereUrl, WhereNote: row.WhereNote, Why: row.Why, Steps: row.Steps, DoneWhen: row.DoneWhen,
			Benefit: int(row.Benefit), Effort: int(row.Effort), ScoreReason: row.ScoreReason, CreatedAt: row.CreatedAt.UTC(),
			DoneBy: row.DoneBy, DoneAt: timeOf(row.DoneAt), EditedAt: timeOf(row.EditedAt),
		})
	}
	return out, nil
}

// RecRuns returns the newest limit Recommendation runs, the newest first.
func (r *Repository) RecRuns(ctx context.Context, limit int) ([]domain.RecRun, error) {
	rows, err := r.q.ListRecommendationRuns(ctx, int32(min(limit, 1000))) //nolint:gosec // at most 1,000
	if err != nil {
		return nil, fmt.Errorf("select recommendation runs: %w", err)
	}
	out := make([]domain.RecRun, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.RecRun{
			ID: row.ID.String(), StartedAt: row.StartedAt.UTC(), FinishedAt: timeOf(row.FinishedAt), Ending: row.Ending, By: row.StartedBy,
			Model: row.Model, Gaps: int(row.Gaps), Added: int(row.Added), Replaced: int(row.Replaced), Kept: int(row.Kept),
			InputTokens: row.InputTokens, OutputTokens: row.OutputTokens,
		})
	}
	return out, nil
}

// BeginRecRun records a run begun.
func (r *Repository) BeginRecRun(ctx context.Context, run domain.RecRun) error {
	id, err := uuid.Parse(run.ID)
	if err != nil {
		return fmt.Errorf("a run's id: %w", err)
	}
	if err := r.q.BeginRecommendationRun(ctx, queries.BeginRecommendationRunParams{
		ID: id, StartedAt: run.StartedAt, StartedBy: run.By, Model: run.Model,
	}); err != nil {
		return fmt.Errorf("insert recommendation run: %w", err)
	}
	return nil
}

// FinishRecRun records how a run ended and, with a replacement, deletes the open recommendations
// it replaces and adds its own, in one transaction.
func (r *Repository) FinishRecRun(ctx context.Context, run domain.RecRun, rep *domain.Replacement) error {
	id, err := uuid.Parse(run.ID)
	if err != nil {
		return fmt.Errorf("a run's id: %w", err)
	}
	return r.store.InTx(ctx, func(q *queries.Queries) error {
		if err := q.FinishRecommendationRun(ctx, queries.FinishRecommendationRunParams{
			ID: id, FinishedAt: pgtype.Timestamptz{Time: run.FinishedAt, Valid: true}, Ending: pgtype.Text{String: run.Ending, Valid: true},
			Gaps: int64(run.Gaps), Added: int64(run.Added), Replaced: int64(run.Replaced), Kept: int64(run.Kept),
			InputTokens: run.InputTokens, OutputTokens: run.OutputTokens,
		}); err != nil {
			return fmt.Errorf("finish recommendation run: %w", err)
		}
		if rep == nil {
			return nil
		}
		for _, old := range rep.Replaced {
			if err := q.DeleteRecommendation(ctx, old.ID); err != nil {
				return fmt.Errorf("delete recommendation: %w", err)
			}
		}
		for _, rec := range rep.Added {
			if err := addRecommendation(ctx, q, id, rec); err != nil {
				return err
			}
		}
		return nil
	})
}

func addRecommendation(ctx context.Context, q *queries.Queries, runID uuid.UUID, rec domain.Recommendation) error {
	g := rec.Gap
	if err := q.AddRecommendation(ctx, queries.AddRecommendationParams{
		ID: rec.ID, RunID: runID, GapKey: g.Key, Kind: string(g.Kind), Language: g.Language, Family: g.Family, Specifics: nonNil(g.Specifics),
		RuleBenefit: int64(g.Benefit), RuleEffort: int64(g.Effort), Title: rec.Title, WhereUrl: rec.Where, WhereNote: rec.WhereNote, //nolint:gosec // scores of 1 to 5
		Why: rec.Why, Steps: nonNil(rec.Steps), DoneWhen: rec.DoneWhen, Benefit: int64(rec.Benefit), Effort: int64(rec.Effort), //nolint:gosec // scores of 1 to 5
		ScoreReason: rec.ScoreReason, CreatedAt: rec.CreatedAt,
	}); err != nil {
		return fmt.Errorf("insert recommendation: %w", err)
	}
	return nil
}

// SetRecDone ticks a written recommendation off, or with by empty undoes the tick; false when
// there is no such recommendation.
func (r *Repository) SetRecDone(ctx context.Context, id, by string, at time.Time) (bool, error) {
	n, err := r.q.SetRecommendationDone(ctx, queries.SetRecommendationDoneParams{
		ID: id, DoneBy: pgtype.Text{String: by, Valid: by != ""}, DoneAt: pgtype.Timestamptz{Time: at, Valid: by != ""},
	})
	if err != nil {
		return false, fmt.Errorf("tick recommendation: %w", err)
	}
	return n > 0, nil
}

// RecommendationTokensSince is what the Recommendation runs used since t, per model that wrote.
func (r *Repository) RecommendationTokensSince(ctx context.Context, t time.Time) ([]llm.TokenUse, error) {
	rows, err := r.q.RecommendationTokensSince(ctx, t)
	if err != nil {
		return nil, fmt.Errorf("sum the recommendation runs' tokens: %w", err)
	}
	out := make([]llm.TokenUse, 0, len(rows))
	for _, row := range rows {
		out = append(out, llm.TokenUse{Model: row.Model, Input: row.Input, Output: row.Output})
	}
	return out, nil
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func timeOf(t pgtype.Timestamptz) time.Time {
	if !t.Valid {
		return time.Time{}
	}
	return t.Time.UTC()
}

// errNoRecommendation is returned by UpdateRecommendation for a recommendation there is not.
var errNoRecommendation = errors.New("no recommendation has this id")

// UpdateRecommendation stores an edited recommendation's wording, scores and when it was edited.
func (r *Repository) UpdateRecommendation(ctx context.Context, rec domain.Recommendation) error {
	n, err := r.q.UpdateRecommendation(ctx, queries.UpdateRecommendationParams{
		ID: rec.ID, Title: rec.Title, WhereUrl: rec.Where, WhereNote: rec.WhereNote, Why: rec.Why, Steps: nonNil(rec.Steps), DoneWhen: rec.DoneWhen,
		Benefit: int64(rec.Benefit), Effort: int64(rec.Effort), EditedAt: pgtype.Timestamptz{Time: rec.EditedAt, Valid: !rec.EditedAt.IsZero()},
	})
	if err != nil {
		return fmt.Errorf("update recommendation: %w", err)
	}
	if n == 0 {
		return errNoRecommendation
	}
	return nil
}
