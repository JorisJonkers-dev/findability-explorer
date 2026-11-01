package httpapi_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/JorisJonkers-dev/findability-explorer/internal/insight/app"
	"github.com/JorisJonkers-dev/findability-explorer/internal/insight/domain"
)

// recRunCases stands in for the Recommendation run's use cases.
type recRunCases struct {
	err      error
	startErr error
	stopErr  error
	started  string
	stopped  int
	edited   string
	edit     domain.Edit
	editErr  error
}

func (c *insightCases) RecommendationRuns(context.Context) (app.RecRuns, error) {
	at := time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC)
	return app.RecRuns{
		Model: "claude-haiku-5-5", Replaces: 12, Stale: true, PromptRunAt: at, ScanAt: at.Add(-time.Hour), Terms: 420, Gaps: 40, Estimate: 70_000,
		Runs: []domain.RecRun{
			{ID: uuid.NewString(), StartedAt: at, FinishedAt: at.Add(time.Minute), Ending: "done", By: "joris", Model: "claude-haiku-5-5", Gaps: 40, Added: 38, Replaced: 10, Kept: 2, InputTokens: 1_000_000, OutputTokens: 100_000},
			{ID: uuid.NewString(), StartedAt: at.Add(time.Hour), By: "joris", Model: "claude-haiku-5-5"},
		},
	}, c.runs.err
}

func (c *insightCases) StartRecommendationRun(_ context.Context, user string) error {
	c.runs.started = user
	return c.runs.startErr
}

func (c *insightCases) EditRecommendation(_ context.Context, id string, e domain.Edit) error {
	c.runs.edited, c.runs.edit = id, e
	return c.runs.editErr
}

func (c *insightCases) StopRecommendationRun(context.Context) error {
	c.runs.stopped++
	return c.runs.stopErr
}

func TestAnAdministratorSeesTheRecommendationRuns(t *testing.T) {
	a := &analyses{}
	srv := serve(t, a)
	const path = "/api/v1/admin/recommendation-runs"
	if r := call(t, srv, http.MethodGet, path, "", true); r.code != http.StatusForbidden {
		t.Fatalf("a viewer: status %d", r.code)
	}
	r := callAs(t, srv, http.MethodGet, path, "", "admin")
	if r.code != http.StatusOK || r.body["replaces"] != float64(12) || r.body["stale"] != true || r.body["estimatedCostUsd"] != 0.07 || r.body["model"] != "claude-haiku-5-5" ||
		r.body["gaps"] != float64(40) || r.body["terms"] != float64(420) || r.body["promptRunAt"] != "2026-11-01T09:00:00Z" || r.body["noKey"] != false {
		t.Fatalf("runs = %d %v", r.code, r.body)
	}
	runs := r.body["runs"].([]any)
	if done := runs[0].(map[string]any); done["ending"] != "done" || done["added"] != float64(38) || done["kept"] != float64(2) || done["costUsd"] != 0.15 {
		t.Errorf("the finished run = %v", done)
	}
	if running := runs[1].(map[string]any); running["ending"] != nil || running["finishedAt"] != nil {
		t.Errorf("the running run = %v", running)
	}
	a.insights().runs.err = errors.New("database down")
	if r := callAs(t, srv, http.MethodGet, path, "", "admin"); r.code != http.StatusInternalServerError {
		t.Errorf("failing: status %d", r.code)
	}
}

func TestAnAdministratorStartsAndStopsARecommendationRun(t *testing.T) {
	a := &analyses{}
	srv := serve(t, a)
	const path = "/api/v1/admin/recommendation-runs"
	if r := call(t, srv, http.MethodPost, path, "", true); r.code != http.StatusForbidden || a.insights().runs.started != "" {
		t.Fatalf("a viewer starting: status %d", r.code)
	}
	if r := callAs(t, srv, http.MethodPost, path, "", "admin"); r.code != http.StatusAccepted || a.insights().runs.started != "Admin" || r.body["replaces"] != float64(12) {
		t.Fatalf("start = %d %v", r.code, r.body)
	}
	for err, detail := range map[error]string{app.ErrRunning: "A Recommendation run is running already.", app.ErrNoWriter: "No Anthropic key is set up to write recommendations."} {
		a.insights().runs.startErr = err
		if r := callAs(t, srv, http.MethodPost, path, "", "admin"); r.code != http.StatusConflict || r.body["detail"] != detail {
			t.Errorf("%v: %d %v", err, r.code, r.body)
		}
	}
	a.insights().runs.startErr = errors.New("database down")
	if r := callAs(t, srv, http.MethodPost, path, "", "admin"); r.code != http.StatusInternalServerError {
		t.Errorf("failing: %d", r.code)
	}
	a.insights().runs.startErr, a.insights().runs.err = nil, errors.New("database down")
	if r := callAs(t, srv, http.MethodPost, path, "", "admin"); r.code != http.StatusInternalServerError {
		t.Errorf("failing to read after starting: %d", r.code)
	}
	const stop = "/api/v1/admin/recommendation-runs/stop"
	if r := call(t, srv, http.MethodPost, stop, "", true); r.code != http.StatusForbidden {
		t.Fatalf("a viewer stopping: status %d", r.code)
	}
	if r := callAs(t, srv, http.MethodPost, stop, "", "admin"); r.code != http.StatusNoContent || a.insights().runs.stopped != 1 {
		t.Fatalf("stop = %d", r.code)
	}
	a.insights().runs.stopErr = app.ErrNothingToStop
	if r := callAs(t, srv, http.MethodPost, stop, "", "admin"); r.code != http.StatusNotFound || r.body["detail"] == nil {
		t.Errorf("nothing to stop = %d %v", r.code, r.body)
	}
	a.insights().runs.stopErr = errors.New("database down")
	if r := callAs(t, srv, http.MethodPost, stop, "", "admin"); r.code != http.StatusInternalServerError {
		t.Errorf("failing stop = %d", r.code)
	}
}

func TestThePlanSaysEachRecommendationsKindAndWhere(t *testing.T) {
	a := &analyses{}
	a.insights().written = true
	r := callAs(t, serve(t, a), http.MethodGet, "/api/v1/plan", "", "tester")
	items := r.body["improvements"].([]any)
	if r.code != http.StatusOK || len(items) != 4 {
		t.Fatalf("plan = %d %v", r.code, r.body)
	}
	hand, scanned := items[0].(map[string]any), items[1].(map[string]any)
	if hand["kind"] != "general" || hand["generated"] != false || hand["new"] != false || hand["where"] != nil || scanned["kind"] != "readiness" {
		t.Errorf("by hand = %v; scanned = %v", hand, scanned)
	}
	w := items[2].(map[string]any)
	if w["kind"] != "listing" || w["generated"] != true || w["new"] != false || w["editedAt"] != "2026-11-02T09:00:00Z" || w["auto"] != true || w["where"] != nil || w["whereNote"] != "Where to put this is for the web agency to decide." ||
		w["scoreReason"] != "Moved." || w["facts"].([]any)[0] != "Cited twice." {
		t.Errorf("written = %v", w)
	}
	if placed := items[3].(map[string]any); placed["where"] != "https://www.tribelt.nl/" || placed["whereNote"] != nil || placed["new"] != false || placed["scoreReason"] != nil ||
		placed["editedAt"] != nil {
		t.Errorf("placed = %v", placed)
	}
}

func TestAnAdministratorEditsARecommendationARunWrote(t *testing.T) {
	a := &analyses{}
	a.insights().written = true
	srv := serve(t, a)
	const path = "/api/v1/admin/recommendations/3fa85f64-5717-4562-b3fc-2c963f66afa6"
	body := `{"title":"Get listed on DirectIndustry","where":"https://www.tribelt.nl/","why":"It is cited.","steps":["Ask."],"doneWhen":"Listed.","benefit":4,"effort":2}`
	if r := call(t, srv, http.MethodPut, path, body, true); r.code != http.StatusForbidden {
		t.Fatalf("a viewer editing: status %d", r.code)
	}
	r := callAs(t, srv, http.MethodPut, path, body, "admin")
	c := a.insights().runs
	if r.code != http.StatusOK || c.edited != "3fa85f64-5717-4562-b3fc-2c963f66afa6" || c.edit.Title != "Get listed on DirectIndustry" || c.edit.Where != "https://www.tribelt.nl/" ||
		c.edit.Benefit != 4 || c.edit.Effort != 2 || c.edit.Steps[0] != "Ask." || r.body["improvements"] == nil {
		t.Fatalf("edit = %d %v; %+v", r.code, r.body, c)
	}
	for _, bad := range []string{`{}`, `{"title":"x","why":"","steps":[],"doneWhen":"","benefit":6,"effort":1}`} {
		if r := callAs(t, srv, http.MethodPut, path, bad, "admin"); r.code != http.StatusBadRequest {
			t.Errorf("%s: status %d", bad, r.code)
		}
	}
	cases := map[error]int{
		domain.ErrUnknownImprovement: http.StatusNotFound,
		domain.ErrNotEditable:        http.StatusUnprocessableEntity,
		domain.ErrInvalid:            http.StatusUnprocessableEntity,
		errors.New("database down"):  http.StatusInternalServerError,
	}
	for err, want := range cases {
		a.insights().runs.editErr = err
		if r := callAs(t, srv, http.MethodPut, path, body, "admin"); r.code != want || (want < 500 && r.body["detail"] == nil) {
			t.Errorf("%v: %d %v", err, r.code, r.body)
		}
	}
}
