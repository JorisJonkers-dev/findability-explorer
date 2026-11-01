// Package web is the Claude steps' inbound HTTP adapter: it implements listClaudeSteps,
// getClaudeStep and saveClaudeStepVersion of the generated ogen Handler.
package web

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/JorisJonkers-dev/findability-explorer/internal/claudestep/app"
	"github.com/JorisJonkers-dev/findability-explorer/internal/claudestep/domain"
	llm "github.com/JorisJonkers-dev/findability-explorer/internal/llm/domain"
	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/httpx"
	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/oas"
)

// UseCases is what the adapter needs from the application layer.
type UseCases interface {
	Steps(ctx context.Context) ([]app.Step, error)
	Step(ctx context.Context, id domain.ID) (app.Step, error)
	Save(ctx context.Context, id domain.ID, actor string, ch domain.Change) (app.Step, error)
}

// Handler implements the Claude step operations.
type Handler struct {
	uc    UseCases
	actor func(context.Context) string
}

// New returns a Handler calling uc; actor names the caller on a saved version.
func New(uc UseCases, actor func(context.Context) string) *Handler {
	return &Handler{uc: uc, actor: actor}
}

// ListClaudeSteps implements listClaudeSteps.
func (h *Handler) ListClaudeSteps(ctx context.Context) (oas.ListClaudeStepsRes, error) {
	steps, err := h.uc.Steps(ctx)
	if err != nil {
		return nil, err
	}
	out := &oas.ClaudeStepList{Items: make([]oas.ClaudeStepSummary, 0, len(steps))}
	for _, s := range steps {
		out.Items = append(out.Items, oas.ClaudeStepSummary{
			ID: oas.ClaudeStepId(s.Definition.ID), Title: s.Definition.Title, FixedInstructions: s.Definition.Fixed,
			Current: toVersion(s.Current), JobCostUsd: oas.Usd(s.JobCost.Dollars()),
		})
	}
	return out, nil
}

// GetClaudeStep implements getClaudeStep.
func (h *Handler) GetClaudeStep(ctx context.Context, params oas.GetClaudeStepParams) (oas.GetClaudeStepRes, error) {
	s, err := h.uc.Step(ctx, domain.ID(params.StepId))
	if errors.Is(err, domain.ErrNotFound) {
		return &oas.GetClaudeStepNotFound{StatusCode: http.StatusNotFound, Response: httpx.Problem(http.StatusNotFound, "No Claude step has this id.")}, nil
	}
	if err != nil {
		return nil, err
	}
	return toStep(s), nil
}

// SaveClaudeStepVersion implements saveClaudeStepVersion.
func (h *Handler) SaveClaudeStepVersion(ctx context.Context, req *oas.ClaudeStepChange, params oas.SaveClaudeStepVersionParams) (oas.SaveClaudeStepVersionRes, error) {
	ch := domain.Change{Model: req.Model, Instructions: req.Instructions}
	if v, ok := req.RestoredFrom.Get(); ok {
		ch.RestoredFrom = int(v)
	}
	s, err := h.uc.Save(ctx, domain.ID(params.StepId), h.actor(ctx), ch)
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return &oas.SaveClaudeStepVersionNotFound{StatusCode: http.StatusNotFound, Response: httpx.Problem(http.StatusNotFound, "No Claude step has this id.")}, nil
	case errors.Is(err, domain.ErrInvalid):
		return &oas.SaveClaudeStepVersionUnprocessableEntity{
			StatusCode: http.StatusUnprocessableEntity, Response: httpx.Problem(http.StatusUnprocessableEntity, sentence(err)),
		}, nil
	case err != nil:
		return nil, err
	}
	return toStep(s), nil
}

// sentence is why a version was refused, as an administrator reads it: "Nothing changed, so there is no new version."
func sentence(err error) string {
	s := strings.TrimPrefix(err.Error(), domain.ErrInvalid.Error()+": ")
	return strings.ToUpper(s[:1]) + s[1:] + "."
}

func toStep(s app.Step) *oas.ClaudeStep {
	out := &oas.ClaudeStep{
		ID: oas.ClaudeStepId(s.Definition.ID), Title: s.Definition.Title, FixedInstructions: s.Definition.Fixed,
		Original: s.Definition.Original, Current: toVersion(s.Current), JobCostUsd: oas.Usd(s.JobCost.Dollars()),
		Versions: make([]oas.ClaudeStepVersion, 0, len(s.Versions)), Models: make([]oas.ClaudeStepModel, 0, len(s.Models)),
	}
	for _, v := range s.Versions {
		out.Versions = append(out.Versions, toVersion(v))
	}
	for _, m := range s.Models {
		out.Models = append(out.Models, oas.ClaudeStepModel{ID: m.Info.ID, Name: m.Info.Name, JobCostUsd: oas.Usd(m.JobCost.Dollars())})
	}
	return out
}

func toVersion(v domain.Version) oas.ClaudeStepVersion {
	return oas.ClaudeStepVersion{
		Number: int32(min(v.Number, 1_000_000)), Model: v.Model, ModelName: llm.ClaudeName(v.Model), //nolint:gosec // at most a million
		Instructions: v.Instructions, Note: v.Note, CreatedBy: v.By, CreatedAt: v.At,
	}
}
