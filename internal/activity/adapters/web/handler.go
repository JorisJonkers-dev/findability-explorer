// Package web is the Activity context's inbound HTTP adapter: it implements getActivity of the
// generated ogen Handler and maps the read model to the contract.
package web

import (
	"context"
	"time"

	"github.com/JorisJonkers-dev/findability-explorer/internal/activity/domain"
	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/oas"
)

// UseCases is what the adapter needs from the application layer.
type UseCases interface {
	Activity(ctx context.Context) (domain.Activity, error)
	Spending(ctx context.Context) ([]domain.SpendLine, time.Time, error)
}

// Handler implements the Activity operations.
type Handler struct {
	uc UseCases
}

// New returns a Handler calling uc.
func New(uc UseCases) *Handler { return &Handler{uc: uc} }

// GetActivity implements getActivity.
func (h *Handler) GetActivity(ctx context.Context) (oas.GetActivityRes, error) {
	a, err := h.uc.Activity(ctx)
	if err != nil {
		return nil, err
	}
	lines, since, err := h.uc.Spending(ctx)
	if err != nil {
		return nil, err
	}
	out := &oas.Activity{Running: toWork(a.Running), Recent: toWork(a.Recent), Spending: oas.Spending{Since: since, Lines: make([]oas.SpendLine, 0, len(lines))}}
	for _, l := range lines {
		out.Spending.Lines = append(out.Spending.Lines, oas.SpendLine{
			Kind: oas.SpendLineKind(l.Kind), Key: oas.SpendLineKey(l.Key), CostUsd: oas.Usd(l.Cost.Dollars()),
		})
	}
	return out, nil
}

func toWork(ws []domain.Work) []oas.Work {
	out := make([]oas.Work, 0, len(ws))
	for _, w := range ws {
		o := oas.Work{
			ID: w.ID, Kind: oas.WorkKind(w.Kind), StartedAt: w.StartedAt,
			Done: int32(min(w.Counts.Done, 1_000_000)), Total: int32(min(w.Counts.Total, 1_000_000)), //nolint:gosec // at most a million
			Failed:            int32(min(w.Counts.Failed, 1_000_000)), //nolint:gosec // at most a million
			OpenRouterCostUsd: oas.Usd(w.OpenRouter.Dollars()), AnthropicCostUsd: oas.Usd(w.Anthropic.Dollars()),
		}
		if !w.Running() {
			o.FinishedAt = oas.NewOptDateTime(w.FinishedAt)
			o.Ending = oas.NewOptWorkEnding(oas.WorkEnding(w.Ending))
		}
		out = append(out, o)
	}
	return out
}
