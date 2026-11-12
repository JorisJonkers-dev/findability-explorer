package httpapi_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/JorisJonkers-dev/findability-explorer/internal/activity/domain"
	"github.com/JorisJonkers-dev/findability-explorer/internal/shared/work"
)

// activityCases stands in for the Activity use cases; err fails them.
type activityCases struct {
	err error
}

func (a *analyses) activities() *activityCases {
	if a.activity == nil {
		a.activity = &activityCases{}
	}
	return a.activity
}

var started = time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC)

func (c *activityCases) Activity(context.Context) (domain.Activity, error) {
	return domain.Activity{
		Running: []domain.Work{{ID: "run", Kind: work.KindPromptRun, StartedAt: started, Counts: work.Counts{Done: 3, Total: 360}, OpenRouter: 1_250_000}},
		Recent: []domain.Work{{
			ID: "review", Kind: work.KindTermReview, StartedAt: started.Add(-time.Hour), FinishedAt: started.Add(-time.Minute),
			Ending: work.EndingStoppedByHand, Counts: work.Counts{Done: 40, Total: 85, Failed: 1}, Anthropic: 400_000,
		}},
	}, c.err
}

func (c *activityCases) Spending(context.Context) ([]domain.SpendLine, time.Time, error) {
	return []domain.SpendLine{
		{Kind: domain.SpendPromptRuns, Key: domain.KeyOpenRouter, Cost: 2_500_000},
		{Kind: domain.SpendReadingResponses, Key: domain.KeyAnthropic, Cost: 100_000},
	}, time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC), c.err
}

func TestAnAdministratorSeesActivity(t *testing.T) {
	a := &analyses{}
	srv := serve(t, a)
	const path = "/api/v1/admin/activity"
	if r := call(t, srv, http.MethodGet, path, "", true); r.code != http.StatusForbidden {
		t.Fatalf("a viewer: status %d", r.code)
	}
	r := callAs(t, srv, http.MethodGet, path, "", "admin")
	if r.code != http.StatusOK {
		t.Fatalf("status %d %v", r.code, r.body)
	}
	running := r.body["running"].([]any)[0].(map[string]any)
	if running["id"] != "run" || running["kind"] != "promptRun" || running["finishedAt"] != nil || running["ending"] != nil ||
		running["done"] != float64(3) || running["total"] != float64(360) || running["openRouterCostUsd"] != 1.25 || running["anthropicCostUsd"] != float64(0) {
		t.Errorf("running = %v", running)
	}
	recent := r.body["recent"].([]any)[0].(map[string]any)
	if recent["kind"] != "termReview" || recent["ending"] != "stoppedByHand" || recent["finishedAt"] != "2026-11-01T08:59:00Z" ||
		recent["failed"] != float64(1) || recent["anthropicCostUsd"] != 0.4 {
		t.Errorf("recent = %v", recent)
	}
	spending := r.body["spending"].(map[string]any)
	lines := spending["lines"].([]any)
	if spending["since"] != "2026-11-01T00:00:00Z" || len(lines) != 2 ||
		lines[0].(map[string]any)["kind"] != "promptRuns" || lines[0].(map[string]any)["key"] != "openRouter" || lines[0].(map[string]any)["costUsd"] != 2.5 ||
		lines[1].(map[string]any)["key"] != "anthropic" {
		t.Errorf("spending = %v", spending)
	}
	a.activities().err = errors.New("database down")
	if r := callAs(t, srv, http.MethodGet, path, "", "admin"); r.code != http.StatusInternalServerError {
		t.Errorf("failing: status %d", r.code)
	}
}
