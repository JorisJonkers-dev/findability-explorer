package httpapi_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/app"
	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain"
)

// definitions stands in for the competitor definition's use cases; err fails each, saved and
// restored record what was asked.
type definitions struct {
	err      error
	saved    []string
	restored []int
}

func (p *promptTests) defs() *definitions {
	if p.definitions == nil {
		p.definitions = &definitions{}
	}
	return p.definitions
}

func definitionView() app.DefinitionView {
	at := time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC)
	current := domain.Definition{Version: 3, Text: "Only makers of eyelink belts compete.", SavedAt: at, SavedBy: "joris"}
	old := domain.Definition{Version: 2, Text: "A competitor makes metal belts.", SavedAt: at.Add(-24 * time.Hour), SavedBy: "import"}
	return app.DefinitionView{Current: current, Versions: []domain.Definition{current, old}, OutdatedCompetitors: 12, OutdatedCandidates: 40, RecheckCost: 260_000}
}

func (p *promptTests) CompetitorDefinition(context.Context) (app.DefinitionView, error) {
	return definitionView(), p.defs().err
}

func (p *promptTests) SaveDefinition(_ context.Context, user, text string) (app.DefinitionView, error) {
	d := p.defs()
	d.saved = append(d.saved, user+": "+text)
	return definitionView(), d.err
}

func (p *promptTests) RestoreDefinition(_ context.Context, _ string, version int) (app.DefinitionView, error) {
	d := p.defs()
	d.restored = append(d.restored, version)
	v := definitionView()
	v.Current.RestoredFrom = version
	return v, d.err
}

func TestAnAdministratorReadsTheCompetitorDefinition(t *testing.T) {
	a := &analyses{}
	srv := serve(t, a)
	const path = "/api/v1/admin/competitor-definition"
	if r := call(t, srv, http.MethodGet, path, "", true); r.code != http.StatusForbidden {
		t.Fatalf("a viewer: status %d", r.code)
	}
	r := callAs(t, srv, http.MethodGet, path, "", "admin")
	current := r.body["current"].(map[string]any)
	if r.code != http.StatusOK || current["version"] != float64(3) || current["text"] != "Only makers of eyelink belts compete." || current["savedBy"] != "joris" ||
		current["savedAt"] != "2026-11-01T09:00:00Z" || current["restoredFrom"] != nil || len(r.body["versions"].([]any)) != 2 ||
		r.body["outdatedCompetitors"] != float64(12) || r.body["outdatedCandidates"] != float64(40) || r.body["recheckCostUsd"] != 0.26 {
		t.Errorf("definition = %d %v", r.code, r.body)
	}
	a.prompts().defs().err = errors.New("database down")
	if r := callAs(t, srv, http.MethodGet, path, "", "admin"); r.code != http.StatusInternalServerError {
		t.Errorf("failing: status %d", r.code)
	}
}

func TestAnAdministratorSavesTheCompetitorDefinition(t *testing.T) {
	a := &analyses{}
	srv := serve(t, a)
	const path = "/api/v1/admin/competitor-definition"
	body := `{"text":"Only makers of eyelink belts compete."}`
	if r := call(t, srv, http.MethodPut, path, body, true); r.code != http.StatusForbidden || len(a.prompts().defs().saved) != 0 {
		t.Fatalf("a viewer: status %d", r.code)
	}
	if r := callAs(t, srv, http.MethodPut, path, body, "admin"); r.code != http.StatusOK || r.body["current"].(map[string]any)["version"] != float64(3) ||
		len(a.prompts().defs().saved) != 1 || a.prompts().defs().saved[0] != "Admin: Only makers of eyelink belts compete." {
		t.Fatalf("saved = %d %v; %v", r.code, r.body, a.prompts().defs().saved)
	}
	for _, bad := range []string{`{}`, `{"text":""}`, `{"text":"x","note":"y"}`} {
		if r := callAs(t, srv, http.MethodPut, path, bad, "admin"); r.code != http.StatusBadRequest {
			t.Errorf("%s: status %d", bad, r.code)
		}
	}
	cases := map[error]int{
		app.ErrUnchanged:                       http.StatusConflict,
		fmt.Errorf("%w: x", domain.ErrInvalid): http.StatusUnprocessableEntity,
		errors.New("database down"):            http.StatusInternalServerError,
	}
	for err, want := range cases {
		a.prompts().defs().err = err
		if r := callAs(t, srv, http.MethodPut, path, body, "admin"); r.code != want || (want < 500 && r.body["detail"] == nil) {
			t.Errorf("%v: status %d %v", err, r.code, r.body)
		}
	}
}

func TestAnAdministratorRestoresAVersionOfTheCompetitorDefinition(t *testing.T) {
	a := &analyses{}
	srv := serve(t, a)
	const path = "/api/v1/admin/competitor-definition/versions/2/restore"
	if r := call(t, srv, http.MethodPost, path, "", true); r.code != http.StatusForbidden {
		t.Fatalf("a viewer: status %d", r.code)
	}
	if r := callAs(t, srv, http.MethodPost, path, "", "admin"); r.code != http.StatusOK || r.body["current"].(map[string]any)["restoredFrom"] != float64(2) ||
		len(a.prompts().defs().restored) != 1 || a.prompts().defs().restored[0] != 2 {
		t.Fatalf("restored = %d %v", r.code, r.body)
	}
	if r := callAs(t, srv, http.MethodPost, "/api/v1/admin/competitor-definition/versions/0/restore", "", "admin"); r.code != http.StatusBadRequest {
		t.Errorf("version 0: status %d", r.code)
	}
	cases := map[error]int{
		fmt.Errorf("version 9: %w", domain.ErrNotFound): http.StatusNotFound,
		app.ErrUnchanged:            http.StatusConflict,
		errors.New("database down"): http.StatusInternalServerError,
	}
	for err, want := range cases {
		a.prompts().defs().err = err
		if r := callAs(t, srv, http.MethodPost, path, "", "admin"); r.code != want || (want < 500 && r.body["detail"] == nil) {
			t.Errorf("%v: status %d %v", err, r.code, r.body)
		}
	}
}
