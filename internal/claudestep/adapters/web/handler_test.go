package web_test

import (
	"context"
	"testing"

	"github.com/JorisJonkers-dev/findability-explorer/internal/claudestep/adapters/web"
	"github.com/JorisJonkers-dev/findability-explorer/internal/claudestep/app"
	"github.com/JorisJonkers-dev/findability-explorer/internal/claudestep/domain"
	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/oas"
)

type missing struct{}

func (missing) Steps(context.Context) ([]app.Step, error) { return nil, nil }
func (missing) Step(context.Context, domain.ID) (app.Step, error) {
	return app.Step{}, domain.ErrNotFound
}

func (missing) Save(context.Context, domain.ID, string, domain.Change) (app.Step, error) {
	return app.Step{}, domain.ErrInvalid
}

func TestAStepNobodyKnowsIsNotFound(t *testing.T) {
	h := web.New(missing{}, func(context.Context) string { return "Joris" })
	res, err := h.GetClaudeStep(t.Context(), oas.GetClaudeStepParams{StepId: oas.ClaudeStepIdMatching})
	if _, ok := res.(*oas.GetClaudeStepNotFound); !ok || err != nil {
		t.Errorf("get = %T %v", res, err)
	}
	saved, err := h.SaveClaudeStepVersion(t.Context(), &oas.ClaudeStepChange{Model: "m", Instructions: "x"}, oas.SaveClaudeStepVersionParams{StepId: oas.ClaudeStepIdMatching})
	refused, ok := saved.(*oas.SaveClaudeStepVersionUnprocessableEntity)
	if !ok || err != nil || refused.Response.Detail.Value != "Invalid version." {
		t.Errorf("save = %#v %v", saved, err)
	}
}
