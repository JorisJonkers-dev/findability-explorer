package httpapi_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/JorisJonkers-dev/findability-explorer/internal/insight/app"
	"github.com/JorisJonkers-dev/findability-explorer/internal/insight/domain"
)

// insightCases is a stand-in for the insight use cases, so these tests exercise the contract.
type insightCases struct {
	err error
	run *domain.Run
}

func (a *analyses) insights() *insightCases {
	if a.insight == nil {
		a.insight = &insightCases{}
	}
	return a.insight
}

func (c *insightCases) Standings(context.Context) (app.Standings, error) {
	share := 0.24
	return app.Standings{
		Standings: []domain.Standing{
			{Measure: domain.MeasureShareOfVoice, Tribelt: &share, Median: new(0.17), Leader: &domain.Leader{CompanyID: "ashworth", Name: "Ashworth", Value: 0.57}, Competitors: 38},
			{Measure: domain.MeasureSitemapPages},
		},
		Trailing: 1,
		Run:      c.run,
	}, c.err
}

func TestStandingsAreForEveryoneSignedIn(t *testing.T) {
	a := &analyses{}
	srv := serve(t, a)
	r := callAs(t, srv, http.MethodGet, "/api/v1/standings", "", "tester")
	measures, _ := r.body["measures"].([]any)
	if r.code != http.StatusOK || len(measures) != 2 || r.body["trailing"].(float64) != 1 || r.body["runStartedAt"] != nil {
		t.Fatalf("standings = %d %v", r.code, r.body)
	}
	voice := measures[0].(map[string]any)
	leader := voice["leader"].(map[string]any)
	if voice["measure"] != "aiShareOfVoice" || voice["tribelt"].(float64) != 0.24 || voice["median"].(float64) != 0.17 ||
		leader["name"] != "Ashworth" || leader["competitorId"] != "ashworth" || voice["trails"] != true || voice["leads"] != false {
		t.Errorf("share of voice = %v", voice)
	}
	empty := measures[1].(map[string]any)
	if _, ok := empty["tribelt"]; ok || empty["leader"] != nil || empty["trails"] != false {
		t.Errorf("a measure without data = %v", empty)
	}

	a.insights().run = &domain.Run{StartedAt: time.Date(2026, 11, 2, 6, 0, 0, 0, time.UTC)}
	if r := callAs(t, srv, http.MethodGet, "/api/v1/standings", "", "tester"); r.body["runStartedAt"] != "2026-11-02T06:00:00Z" {
		t.Errorf("runStartedAt = %v", r.body["runStartedAt"])
	}
	a.insights().err = errors.New("down")
	if r := callAs(t, srv, http.MethodGet, "/api/v1/standings", "", "tester"); r.code != http.StatusInternalServerError {
		t.Errorf("a failure = %d", r.code)
	}
	if r := call(t, srv, http.MethodGet, "/api/v1/standings", "", false); r.code != http.StatusUnauthorized {
		t.Errorf("signed out = %d", r.code)
	}
}
