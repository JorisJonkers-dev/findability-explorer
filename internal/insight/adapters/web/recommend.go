package web

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"

	"github.com/JorisJonkers-dev/findability-explorer/internal/insight/app"
	llm "github.com/JorisJonkers-dev/findability-explorer/internal/llm/domain"
	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/httpx"
	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/oas"
)

// GetRecommendationRuns implements getRecommendationRuns.
func (h *Handler) GetRecommendationRuns(ctx context.Context) (oas.GetRecommendationRunsRes, error) {
	r, err := h.uc.RecommendationRuns(ctx)
	if err != nil {
		return nil, err
	}
	out := toRecRuns(r)
	return &out, nil
}

// StartRecommendationRun implements startRecommendationRun.
func (h *Handler) StartRecommendationRun(ctx context.Context) (oas.StartRecommendationRunRes, error) {
	err := h.uc.StartRecommendationRun(ctx, h.actor(ctx))
	switch {
	case errors.Is(err, app.ErrRunning):
		return &oas.StartRecommendationRunConflict{StatusCode: http.StatusConflict, Response: httpx.Problem(http.StatusConflict, "A Recommendation run is running already.")}, nil
	case errors.Is(err, app.ErrNoWriter):
		return &oas.StartRecommendationRunConflict{StatusCode: http.StatusConflict, Response: httpx.Problem(http.StatusConflict, "No Anthropic key is set up to write recommendations.")}, nil
	case err != nil:
		return nil, err
	}
	r, err := h.uc.RecommendationRuns(ctx)
	if err != nil {
		return nil, err
	}
	out := toRecRuns(r)
	return &out, nil
}

// StopRecommendationRun implements stopRecommendationRun.
func (h *Handler) StopRecommendationRun(ctx context.Context) (oas.StopRecommendationRunRes, error) {
	err := h.uc.StopRecommendationRun(ctx)
	switch {
	case errors.Is(err, app.ErrNothingToStop):
		return &oas.StopRecommendationRunNotFound{StatusCode: http.StatusNotFound, Response: httpx.Problem(http.StatusNotFound, "No Recommendation run runs.")}, nil
	case err != nil:
		return nil, err
	}
	return &oas.StopRecommendationRunNoContent{}, nil
}

func toRecRuns(r app.RecRuns) oas.RecommendationRuns {
	out := oas.RecommendationRuns{
		Running: r.Running, NoKey: r.NoKey, Replaces: int32(min(r.Replaces, 100_000)), Stale: r.Stale, //nolint:gosec // at most 100,000
		Terms: int32(min(r.Terms, 100_000)), Gaps: int32(min(r.Gaps, 1000)), EstimatedCostUsd: oas.Usd(r.Estimate.Dollars()), //nolint:gosec // bounded
		Runs: make([]oas.RecommendationRun, 0, len(r.Runs)),
	}
	if r.Model != "" {
		out.Model = oas.NewOptString(r.Model)
	}
	if !r.PromptRunAt.IsZero() {
		out.PromptRunAt = oas.NewOptDateTime(r.PromptRunAt)
	}
	if !r.ScanAt.IsZero() {
		out.ScanAt = oas.NewOptDateTime(r.ScanAt)
	}
	for _, run := range r.Runs {
		id, _ := uuid.Parse(run.ID)
		o := oas.RecommendationRun{
			ID: id, StartedAt: run.StartedAt, StartedBy: run.By, Model: run.Model,
			Gaps: int32(min(run.Gaps, 1000)), Added: int32(min(run.Added, 100_000)), Replaced: int32(min(run.Replaced, 100_000)), Kept: int32(min(run.Kept, 100_000)), //nolint:gosec // bounded
			CostUsd: oas.Usd(llm.TokenUse{Model: run.Model, Input: run.InputTokens, Output: run.OutputTokens}.Cost().Dollars()),
		}
		if !run.FinishedAt.IsZero() {
			o.FinishedAt = oas.NewOptDateTime(run.FinishedAt)
			o.Ending = oas.NewOptRecommendationRunEnding(oas.RecommendationRunEnding(run.Ending))
		}
		out.Runs = append(out.Runs, o)
	}
	return out
}
