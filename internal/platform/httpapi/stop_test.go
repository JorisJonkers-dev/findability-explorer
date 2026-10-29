package httpapi_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/google/uuid"

	analysisapp "github.com/JorisJonkers-dev/findability-explorer/internal/analysis/app"
	"github.com/JorisJonkers-dev/findability-explorer/internal/analysis/domain"
)

func (a *analyses) stop(name string) error {
	a.stops = append(a.stops, name)
	return a.stopErr
}

func (a *analyses) StopScan(_ context.Context, id uuid.UUID) error {
	return a.stop("scan " + id.String())
}

func (a *analyses) StopRescan(context.Context) error { return a.stop("rescan") }

func (a *analyses) RemoveWaitingScans(context.Context) error { return a.stop("waiting") }

func (a *analyses) StopReadinessCheck(context.Context) error { return a.stop("readiness") }

func TestAnAdministratorStopsScansRescansAndTheReadinessCheck(t *testing.T) {
	id := uuid.New()
	stops := map[string]struct{ method, path string }{
		"scan " + id.String(): {http.MethodPost, "/api/v1/admin/analyses/" + id.String() + "/stop"},
		"waiting":             {http.MethodDelete, "/api/v1/admin/analyses/queue"},
		"rescan":              {http.MethodPost, "/api/v1/admin/rescans/stop"},
		"readiness":           {http.MethodPost, "/api/v1/admin/readiness-checks/stop"},
	}
	for name, op := range stops {
		t.Run(name, func(t *testing.T) {
			a := &analyses{}
			srv := serve(t, a)
			if r := call(t, srv, op.method, op.path, "", true); r.code != http.StatusForbidden || len(a.stops) != 0 {
				t.Fatalf("a viewer stopping: status %d, stops %v", r.code, a.stops)
			}
			if r := callAs(t, srv, op.method, op.path, "", "admin"); r.code != http.StatusNoContent || len(a.stops) != 1 || a.stops[0] != name {
				t.Fatalf("an administrator stopping: status %d, stops %v", r.code, a.stops)
			}
			a.stopErr = analysisapp.ErrNothingToStop
			if r := callAs(t, srv, op.method, op.path, "", "admin"); r.code != http.StatusNotFound || r.body["detail"] == nil {
				t.Errorf("nothing to stop: status %d %v", r.code, r.body)
			}
			a.stopErr = errors.New("database down")
			if r := callAs(t, srv, op.method, op.path, "", "admin"); r.code != http.StatusInternalServerError {
				t.Errorf("a failing stop: status %d", r.code)
			}
		})
	}
	if r := callAs(t, serve(t, &analyses{}), http.MethodPost, "/api/v1/admin/analyses/not-an-id/stop", "", "admin"); r.code != http.StatusBadRequest {
		t.Errorf("a malformed id: status %d", r.code)
	}
}

// A scan stopped by hand and a stopped readiness check say so.
func TestStoppedWorkSaysItWasStoppedByHand(t *testing.T) {
	scan := done()
	scan.Status, scan.Failure = domain.StatusFailed, domain.StoppedByHandReason
	a := &analyses{stored: scan, check: analysisapp.ReadinessCheck{Checked: 3, Total: 40, Stopped: true}}
	srv := serve(t, a)
	r := call(t, srv, http.MethodGet, "/api/v1/analyses/"+scan.ID.String(), "", true)
	if summary, _ := r.body["summary"].(map[string]any); summary["stoppedByHand"] != true || summary["status"] != "failed" {
		t.Errorf("stopped scan = %d %v", r.code, summary)
	}
	if r := call(t, srv, http.MethodGet, "/api/v1/ai-readiness", "", true); r.body["check"].(map[string]any)["stopped"] != true {
		t.Errorf("stopped check = %d %v", r.code, r.body["check"])
	}
	a.stored, a.check = done(), analysisapp.ReadinessCheck{}
	r = call(t, srv, http.MethodGet, "/api/v1/analyses/"+a.stored.ID.String(), "", true)
	if summary, _ := r.body["summary"].(map[string]any); summary == nil || summary["stoppedByHand"] != nil {
		t.Errorf("a finished scan = %v", summary)
	}
	if r := call(t, srv, http.MethodGet, "/api/v1/ai-readiness", "", true); r.body["check"].(map[string]any)["stopped"] != nil {
		t.Errorf("a check nobody stopped = %v", r.body["check"])
	}
}
