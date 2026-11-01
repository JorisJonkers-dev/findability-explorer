// Package persistence implements the Activity Source and the work journal on the sqlc-generated
// queries: it reads the Prompt runs, the scans and the journal, and what each kind of work used.
package persistence

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/JorisJonkers-dev/findability-explorer/internal/activity/app"
	"github.com/JorisJonkers-dev/findability-explorer/internal/activity/domain"
	llm "github.com/JorisJonkers-dev/findability-explorer/internal/llm/domain"
	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/pg/queries"
	"github.com/JorisJonkers-dev/findability-explorer/internal/shared/work"
)

// Store is what the repository needs from the Postgres adapter.
type Store interface {
	Queries() *queries.Queries
}

// Repository reads and records the work.
type Repository struct {
	store Store
}

var (
	_ app.Source   = (*Repository)(nil)
	_ work.Journal = (*Repository)(nil)
)

// errEnded is returned by End for work that ended already, or was never begun.
var errEnded = errors.New("no running work has this id")

// New returns a Repository over store.
func New(store Store) *Repository { return &Repository{store: store} }

// Begin records a piece of work begun, and returns its id.
func (r *Repository) Begin(ctx context.Context, kind work.Kind, by string, at time.Time) (uuid.UUID, error) {
	id := uuid.New()
	if err := r.store.Queries().BeginWork(ctx, queries.BeginWorkParams{ID: id, Kind: string(kind), StartedBy: by, StartedAt: at}); err != nil {
		return uuid.Nil, fmt.Errorf("insert work: %w", err)
	}
	return id, nil
}

// End records how the work with id ended.
func (r *Repository) End(ctx context.Context, id uuid.UUID, o work.Outcome, at time.Time) error {
	n, err := r.store.Queries().EndWork(ctx, queries.EndWorkParams{
		ID: id, FinishedAt: pgtype.Timestamptz{Time: at, Valid: true}, Ending: pgtype.Text{String: string(o.Ending), Valid: true},
		Done: int64(o.Counts.Done), Total: int64(o.Counts.Total), Failed: int64(o.Counts.Failed),
		InputTokens: o.InputTokens, OutputTokens: o.OutputTokens,
	})
	if err != nil {
		return fmt.Errorf("end work: %w", err)
	}
	if n == 0 {
		return errEnded
	}
	return nil
}

// Beat records that the work with id still runs, at; an error when it ended.
func (r *Repository) Beat(ctx context.Context, id uuid.UUID, at time.Time) error {
	n, err := r.store.Queries().BeatWork(ctx, queries.BeatWorkParams{ID: id, At: pgtype.Timestamptz{Time: at, Valid: true}})
	if err != nil {
		return fmt.Errorf("beat work: %w", err)
	}
	if n == 0 {
		return errEnded
	}
	return nil
}

// EndAbandoned ends, as failed at at, the running work last heard of before staleBefore, or, begun
// before work beat, begun before unbeatenBefore; and counts it.
func (r *Repository) EndAbandoned(ctx context.Context, at, staleBefore, unbeatenBefore time.Time) (int, error) {
	n, err := r.store.Queries().EndAbandonedWork(ctx, queries.EndAbandonedWorkParams{
		FinishedAt: pgtype.Timestamptz{Time: at, Valid: true}, StaleBefore: pgtype.Timestamptz{Time: staleBefore, Valid: true}, UnbeatenBefore: unbeatenBefore,
	})
	if err != nil {
		return 0, fmt.Errorf("end the work left running: %w", err)
	}
	return int(n), nil
}

// Recorded returns the running work and the newest limit ended pieces.
func (r *Repository) Recorded(ctx context.Context, limit int) ([]domain.Recorded, error) {
	rows, err := r.store.Queries().ListWork(ctx, int32(min(limit, 1000))) //nolint:gosec // at most 1,000
	if err != nil {
		return nil, fmt.Errorf("select work: %w", err)
	}
	out := make([]domain.Recorded, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.Recorded{
			ID: row.ID, Kind: work.Kind(row.Kind), StartedAt: row.StartedAt.UTC(), FinishedAt: timeOf(row.FinishedAt),
			Outcome: work.Outcome{
				Ending: work.Ending(row.Ending.String), Counts: work.Counts{Done: int(row.Done), Total: int(row.Total), Failed: int(row.Failed)},
				InputTokens: row.InputTokens, OutputTokens: row.OutputTokens,
			},
		})
	}
	return out, nil
}

// PromptRuns returns the running Prompt run and the newest limit ended ones, with their counts and cost.
func (r *Repository) PromptRuns(ctx context.Context, limit int) ([]domain.PromptRunRow, error) {
	rows, err := r.store.Queries().ActivityPromptRuns(ctx, int32(min(limit, 1000))) //nolint:gosec // at most 1,000
	if err != nil {
		return nil, fmt.Errorf("select prompt runs: %w", err)
	}
	out := make([]domain.PromptRunRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.PromptRunRow{
			ID: row.ID, StartedAt: row.StartedAt.UTC(), FinishedAt: timeOf(row.FinishedAt), Status: row.Status, StoppedByHand: row.StoppedByHand,
			Tasks: int(row.Tasks), Answered: int(row.Answered), Failed: int(row.Failed),
			AnswerCost: llm.Micros(row.AnswerCost), OpenRouterReadingCost: llm.Micros(row.OpenrouterReadingCost),
			HaikuInput: row.HaikuInput, HaikuOutput: row.HaikuOutput,
		})
	}
	return out, nil
}

// Scans returns the waiting and running scans and the newest limit others.
func (r *Repository) Scans(ctx context.Context, limit int) ([]domain.ScanRow, error) {
	rows, err := r.store.Queries().ActivityScans(ctx, int32(min(limit, 10000))) //nolint:gosec // at most 10,000
	if err != nil {
		return nil, fmt.Errorf("select scans: %w", err)
	}
	out := make([]domain.ScanRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.ScanRow{
			ID: row.ID, Rescan: row.Rescan, CreatedBy: row.CreatedBy, CreatedAt: row.CreatedAt.UTC(), FinishedAt: timeOf(row.FinishedAt),
			Status: row.Status, StopRequested: row.StopRequested, Pages: int(row.PagesCrawled), MaxPages: int(row.MaxPages),
			InputTokens: row.LlmInputTokens, OutputTokens: row.LlmOutputTokens,
		})
	}
	return out, nil
}

// MonthUsage is what each kind of work used since since.
func (r *Repository) MonthUsage(ctx context.Context, since time.Time) (domain.MonthUsage, error) {
	u, err := r.store.Queries().ActivityMonthUsage(ctx, since)
	if err != nil {
		return domain.MonthUsage{}, fmt.Errorf("sum this month's usage: %w", err)
	}
	return domain.MonthUsage{
		PromptAnswers: llm.Micros(u.PromptAnswers), OpenRouterReadings: llm.Micros(u.OpenrouterReadings),
		HaikuReadings:      llm.TokenUse{Model: llm.DefaultModel, Input: u.HaikuReadingInput, Output: u.HaikuReadingOutput},
		Scans:              llm.TokenUse{Model: llm.DefaultModel, Input: u.ScanInput, Output: u.ScanOutput},
		TermReview:         llm.TokenUse{Model: llm.DefaultModel, Input: u.TermReviewInput, Output: u.TermReviewOutput},
		CandidateChecks:    llm.TokenUse{Model: llm.DefaultModel, Input: u.CandidateCheckInput, Output: u.CandidateCheckOutput},
		RecommendationRuns: llm.TokenUse{Model: llm.DefaultModel, Input: u.RecommendationInput, Output: u.RecommendationOutput},
	}, nil
}

// timeOf is t in UTC, or zero when it is null.
func timeOf(t pgtype.Timestamptz) time.Time {
	if !t.Valid {
		return time.Time{}
	}
	return t.Time.UTC()
}
