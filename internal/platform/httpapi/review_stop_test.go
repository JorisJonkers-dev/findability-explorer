package httpapi_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/app"
)

func (p *promptTests) StopReview(context.Context) error {
	p.workStops = append(p.workStops, "candidates")
	return p.workStopErr
}

func (p *promptTests) StopListCheck(context.Context) error {
	p.workStops = append(p.workStops, "list")
	return p.workStopErr
}

func TestAnAdministratorSwitchesTheTermReviewOffAndOn(t *testing.T) {
	srv := serve(t, &analyses{})
	const path = "/api/v1/admin/term-candidates/review/switch"
	if r := call(t, srv, http.MethodPut, path, `{"on":false}`, true); r.code != http.StatusForbidden {
		t.Fatalf("a viewer switching: status %d", r.code)
	}
	r := callAs(t, srv, http.MethodPut, path, `{"on":false}`, "admin")
	if r.code != http.StatusOK || r.body["on"] != false || r.body["waiting"] != float64(1) ||
		r.body["monthTokens"] != float64(1_200_000) || r.body["monthCostUsd"] != 0.2 {
		t.Fatalf("switched off = %d %v", r.code, r.body)
	}
	list := callAs(t, srv, http.MethodGet, "/api/v1/admin/term-candidates", "", "admin")
	if review := list.body["review"].(map[string]any); review["on"] != false {
		t.Errorf("the list's review = %v", review)
	}
	if r := callAs(t, srv, http.MethodPut, path, `{"on":true}`, "admin"); r.code != http.StatusOK || r.body["on"] != true {
		t.Errorf("switched on = %d %v", r.code, r.body)
	}
	for _, body := range []string{`{}`, `{"on":true,"why":"x"}`, `{"on":"yes"}`} {
		if r := callAs(t, srv, http.MethodPut, path, body, "admin"); r.code != http.StatusBadRequest {
			t.Errorf("%s: status %d", body, r.code)
		}
	}
}

func TestAnAdministratorStopsTheTermReview(t *testing.T) {
	srv := serve(t, &analyses{})
	const path = "/api/v1/admin/term-candidates/review/stop"
	if r := callAs(t, srv, http.MethodPost, path, "", "admin"); r.code != http.StatusNotFound || r.body["detail"] == nil {
		t.Fatalf("nothing to stop: status %d %v", r.code, r.body)
	}
	if r := callAs(t, srv, http.MethodPost, "/api/v1/admin/term-candidates/review", "", "admin"); r.code != http.StatusAccepted || r.body["running"] != true {
		t.Fatalf("Review now = %d %v", r.code, r.body)
	}
	if r := call(t, srv, http.MethodPost, path, "", true); r.code != http.StatusForbidden {
		t.Fatalf("a viewer stopping: status %d", r.code)
	}
	if r := callAs(t, srv, http.MethodPost, path, "", "admin"); r.code != http.StatusNoContent {
		t.Fatalf("an administrator stopping: status %d", r.code)
	}
	list := callAs(t, srv, http.MethodGet, "/api/v1/admin/term-candidates", "", "admin")
	if review := list.body["review"].(map[string]any); review["running"] != false || review["problem"] != "stopped" {
		t.Errorf("the stopped review = %v", review)
	}
}

func TestAnAdministratorStopsTheCandidateReviewAndTheCompetitorCheck(t *testing.T) {
	stops := map[string]string{
		"candidates": "/api/v1/admin/competitor-candidates/review/stop",
		"list":       "/api/v1/admin/competitor-checks/stop",
	}
	for name, path := range stops {
		t.Run(name, func(t *testing.T) {
			a := &analyses{}
			p := a.prompts()
			srv := serve(t, a)
			if r := call(t, srv, http.MethodPost, path, "", true); r.code != http.StatusForbidden || len(p.workStops) != 0 {
				t.Fatalf("a viewer stopping: status %d, stops %v", r.code, p.workStops)
			}
			if r := callAs(t, srv, http.MethodPost, path, "", "admin"); r.code != http.StatusNoContent || len(p.workStops) != 1 || p.workStops[0] != name {
				t.Fatalf("an administrator stopping: status %d, stops %v", r.code, p.workStops)
			}
			p.workStopErr = app.ErrNothingToStop
			if r := callAs(t, srv, http.MethodPost, path, "", "admin"); r.code != http.StatusNotFound || r.body["detail"] == nil {
				t.Errorf("nothing to stop: status %d %v", r.code, r.body)
			}
			p.workStopErr = errors.New("database down")
			if r := callAs(t, srv, http.MethodPost, path, "", "admin"); r.code != http.StatusInternalServerError {
				t.Errorf("a failing stop: status %d", r.code)
			}
		})
	}
	// A review stopped by hand says so.
	a := &analyses{}
	a.prompts().problem = app.ReviewProblemStopped
	r := callAs(t, serve(t, a), http.MethodGet, "/api/v1/admin/competitor-candidates", "", "admin")
	if review := r.body["review"].(map[string]any); review["problem"] != "stopped" {
		t.Errorf("the stopped review = %d %v", r.code, review)
	}
}
