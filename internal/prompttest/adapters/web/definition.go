package web

import (
	"context"
	"errors"
	"net/http"

	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/httpx"
	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/oas"
	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/app"
	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain"
)

// GetCompetitorDefinition implements getCompetitorDefinition.
func (h *Handler) GetCompetitorDefinition(ctx context.Context) (oas.GetCompetitorDefinitionRes, error) {
	v, err := h.uc.CompetitorDefinition(ctx)
	if err != nil {
		return nil, err
	}
	out := toDefinition(v)
	return &out, nil
}

// SaveCompetitorDefinition implements saveCompetitorDefinition.
func (h *Handler) SaveCompetitorDefinition(ctx context.Context, req *oas.CompetitorDefinitionText) (oas.SaveCompetitorDefinitionRes, error) {
	v, err := h.uc.SaveDefinition(ctx, h.actor(ctx), req.Text)
	switch {
	case errors.Is(err, app.ErrUnchanged):
		return &oas.SaveCompetitorDefinitionConflict{StatusCode: http.StatusConflict, Response: httpx.Problem(http.StatusConflict, sentence(err))}, nil
	case errors.Is(err, domain.ErrInvalid):
		return &oas.SaveCompetitorDefinitionUnprocessableEntity{
			StatusCode: http.StatusUnprocessableEntity, Response: httpx.Problem(http.StatusUnprocessableEntity, sentence(err)),
		}, nil
	case err != nil:
		return nil, err
	}
	out := toDefinition(v)
	return &out, nil
}

// RestoreCompetitorDefinition implements restoreCompetitorDefinition.
func (h *Handler) RestoreCompetitorDefinition(ctx context.Context, params oas.RestoreCompetitorDefinitionParams) (oas.RestoreCompetitorDefinitionRes, error) {
	v, err := h.uc.RestoreDefinition(ctx, h.actor(ctx), int(params.Version))
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return &oas.RestoreCompetitorDefinitionNotFound{
			StatusCode: http.StatusNotFound, Response: httpx.Problem(http.StatusNotFound, "No version of the competitor definition has this number."),
		}, nil
	case errors.Is(err, app.ErrUnchanged):
		return &oas.RestoreCompetitorDefinitionConflict{StatusCode: http.StatusConflict, Response: httpx.Problem(http.StatusConflict, sentence(err))}, nil
	case err != nil:
		return nil, err
	}
	out := toDefinition(v)
	return &out, nil
}

func toDefinition(v app.DefinitionView) oas.CompetitorDefinition {
	versions := make([]oas.CompetitorDefinitionVersion, 0, len(v.Versions))
	for _, d := range v.Versions {
		versions = append(versions, toDefinitionVersion(d))
	}
	return oas.CompetitorDefinition{
		Current: toDefinitionVersion(v.Current), Versions: versions,
		OutdatedCompetitors: int32(min(v.OutdatedCompetitors, 100_000)), //nolint:gosec // at most 100,000
		OutdatedCandidates:  int32(min(v.OutdatedCandidates, 100_000)),  //nolint:gosec // at most 100,000
		RecheckCostUsd:      oas.Usd(v.RecheckCost.Dollars()),
	}
}

func toDefinitionVersion(d domain.Definition) oas.CompetitorDefinitionVersion {
	out := oas.CompetitorDefinitionVersion{
		Version: int32(min(d.Version, 100_000)), Text: d.Text, SavedAt: d.SavedAt, SavedBy: d.SavedBy, //nolint:gosec // at most 100,000
	}
	if d.RestoredFrom > 0 {
		out.RestoredFrom = oas.NewOptInt32(int32(min(d.RestoredFrom, 100_000))) //nolint:gosec // at most 100,000
	}
	return out
}
