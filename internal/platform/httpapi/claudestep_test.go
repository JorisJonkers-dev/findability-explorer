package httpapi_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	claudestepapp "github.com/JorisJonkers-dev/findability-explorer/internal/claudestep/app"
	"github.com/JorisJonkers-dev/findability-explorer/internal/claudestep/domain"
	llm "github.com/JorisJonkers-dev/findability-explorer/internal/llm/domain"
)

type stepCases struct {
	err     error
	saved   domain.Change
	actor   string
	saveErr error
}

func (a *analyses) claudeSteps() *stepCases {
	if a.steps == nil {
		a.steps = &stepCases{}
	}
	return a.steps
}

var stepAt = time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC)

func matchingStep() claudestepapp.Step {
	d, _ := domain.Lookup(domain.Matching)
	original := domain.Original(d)
	original.At = stepAt
	current := domain.Version{Step: domain.Matching, Number: 2, Model: "claude-sonnet-5-5", Instructions: "Match.", Note: "", By: "Admin", At: stepAt}
	return claudestepapp.Step{
		Definition: d, Current: current, Versions: []domain.Version{original, current}, JobCost: 88_000,
		Models: []claudestepapp.Model{{Info: llm.ClaudeModels()[0], JobCost: 4_400}, {Info: llm.ClaudeModels()[2], JobCost: 88_000}},
	}
}

func (c *stepCases) Steps(context.Context) ([]claudestepapp.Step, error) {
	return []claudestepapp.Step{matchingStep()}, c.err
}

func (c *stepCases) Step(_ context.Context, id domain.ID) (claudestepapp.Step, error) {
	if id != domain.Matching {
		return claudestepapp.Step{}, domain.ErrNotFound
	}
	return matchingStep(), c.err
}

func (c *stepCases) Save(_ context.Context, _ domain.ID, actor string, ch domain.Change) (claudestepapp.Step, error) {
	c.actor, c.saved = actor, ch
	return matchingStep(), c.saveErr
}

func TestAnAdministratorListsTheClaudeSteps(t *testing.T) {
	a := &analyses{}
	srv := serve(t, a)
	if r := call(t, srv, http.MethodGet, "/api/v1/admin/claude-steps", "", true); r.code != http.StatusForbidden {
		t.Fatalf("a viewer: status %d", r.code)
	}
	r := callAs(t, srv, http.MethodGet, "/api/v1/admin/claude-steps", "", "admin")
	if r.code != http.StatusOK {
		t.Fatalf("status %d %v", r.code, r.body)
	}
	item := r.body["items"].([]any)[0].(map[string]any)
	current := item["current"].(map[string]any)
	if item["id"] != "matching" || item["title"] != "Matching products" || item["fixedInstructions"] != false || item["jobCostUsd"] != 0.088 ||
		current["number"] != float64(2) || current["modelName"] != "Claude Sonnet 5.5" || current["createdBy"] != "Admin" {
		t.Errorf("item = %v", item)
	}
	a.claudeSteps().err = errors.New("database down")
	if r := callAs(t, srv, http.MethodGet, "/api/v1/admin/claude-steps", "", "admin"); r.code != http.StatusInternalServerError {
		t.Errorf("failing: status %d", r.code)
	}
}

func TestAnAdministratorReadsOneClaudeStep(t *testing.T) {
	a := &analyses{}
	srv := serve(t, a)
	r := callAs(t, srv, http.MethodGet, "/api/v1/admin/claude-steps/matching", "", "admin")
	if r.code != http.StatusOK {
		t.Fatalf("status %d %v", r.code, r.body)
	}
	versions := r.body["versions"].([]any)
	models := r.body["models"].([]any)
	if r.body["original"] == "" || len(versions) != 2 || versions[0].(map[string]any)["createdBy"] != domain.ByTheTool ||
		models[0].(map[string]any)["id"] != llm.DefaultModel || models[0].(map[string]any)["jobCostUsd"] != 0.0044 {
		t.Errorf("step = %v", r.body)
	}
	if r := callAs(t, srv, http.MethodGet, "/api/v1/admin/claude-steps/nothing", "", "admin"); r.code != http.StatusBadRequest {
		t.Errorf("a step the contract does not name: status %d", r.code)
	}
	a.claudeSteps().err = errors.New("database down")
	if r := callAs(t, srv, http.MethodGet, "/api/v1/admin/claude-steps/matching", "", "admin"); r.code != http.StatusInternalServerError {
		t.Errorf("failing: status %d", r.code)
	}
}

func TestAnAdministratorSavesAVersionOfAClaudeStep(t *testing.T) {
	a := &analyses{}
	srv := serve(t, a)
	const path = "/api/v1/admin/claude-steps/matching/versions"
	body := `{"model":"claude-sonnet-5-5","instructions":"Match.","restoredFrom":1}`
	if r := call(t, srv, http.MethodPost, path, body, true); r.code != http.StatusForbidden {
		t.Fatalf("a viewer: status %d", r.code)
	}
	r := callAs(t, srv, http.MethodPost, path, body, "admin")
	if r.code != http.StatusCreated || r.body["id"] != "matching" {
		t.Fatalf("status %d %v", r.code, r.body)
	}
	if c := a.claudeSteps(); c.actor != "Admin" || c.saved != (domain.Change{Model: "claude-sonnet-5-5", Instructions: "Match.", RestoredFrom: 1}) {
		t.Errorf("saved %+v by %q", c.saved, c.actor)
	}
	a.claudeSteps().saveErr = fmt.Errorf("%w: nothing changed, so there is no new version", domain.ErrInvalid)
	r = callAs(t, srv, http.MethodPost, path, `{"model":"claude-haiku-5-5","instructions":"Match."}`, "admin")
	if r.code != http.StatusUnprocessableEntity || r.body["detail"] != "Nothing changed, so there is no new version." {
		t.Errorf("unchanged: %d %v", r.code, r.body)
	}
	a.claudeSteps().saveErr = domain.ErrNotFound
	if r := callAs(t, srv, http.MethodPost, path, body, "admin"); r.code != http.StatusNotFound {
		t.Errorf("unknown: status %d", r.code)
	}
	a.claudeSteps().saveErr = errors.New("database down")
	if r := callAs(t, srv, http.MethodPost, path, body, "admin"); r.code != http.StatusInternalServerError {
		t.Errorf("failing: status %d", r.code)
	}
	if r := callAs(t, srv, http.MethodPost, path, `{"model":"","instructions":""}`, "admin"); r.code != http.StatusBadRequest {
		t.Errorf("empty: status %d", r.code)
	}
}

func TestAStepTheServiceDoesNotKnowIsNotFound(t *testing.T) {
	srv := serve(t, &analyses{})
	if r := callAs(t, srv, http.MethodGet, "/api/v1/admin/claude-steps/reading", "", "admin"); r.code != http.StatusNotFound {
		t.Errorf("status %d", r.code)
	}
}
