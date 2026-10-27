package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/JorisJonkers-dev/findability-explorer/internal/insight/app"
	"github.com/JorisJonkers-dev/findability-explorer/internal/insight/domain"
)

func TestAiSearchFollowsTribeltAndTheMostNamed(t *testing.T) {
	w := newWorld()
	w.runs[0].Answers[0].PromptID, w.runs[0].Answers[0].Provider = "q1", "openai"
	w.runs[0].Answers[1].PromptID, w.runs[0].Answers[1].Provider = "q1", "google"
	w.readiness["tribelt"] = domain.Readiness{Bots: map[string]domain.BotAccess{"GPTBot": domain.BotBlocked, "Googlebot": domain.BotAllowed}}
	w.readiness["ashworth"] = domain.Readiness{Blocked: []string{"CCBot"}}
	got, err := service(w).AiSearch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if w.asked != app.TrendRuns || got.Goal != 0.40 || got.Questions != 1 || got.Providers != 2 || got.Sites != 3 {
		t.Errorf("summary = %+v, asked for %d runs", got, w.asked)
	}
	if len(got.Followed) != 2 || got.Followed[0].ID != "tribelt" || got.Followed[1].ID != "ashworth" {
		t.Errorf("followed = %+v", got.Followed)
	}
	if len(got.Trend) != 2 || got.Trend[0].RunID != "older" || got.Trend[1].Shares[0].Share != 0.5 {
		t.Errorf("trend = %+v", got.Trend)
	}
	access := map[string]domain.BotAccess{}
	for _, c := range got.Crawlers {
		access[c.Bot] = c.Access
	}
	if len(got.Crawlers) != len(domain.Crawlers()) || access["GPTBot"] != domain.BotBlocked || access["Googlebot"] != domain.BotAllowed {
		t.Errorf("crawlers = %+v", got.Crawlers)
	}
	if len(got.KeepingOut) != 1 || got.KeepingOut[0].Name != "Ashworth" || len(got.Ranking) != 3 {
		t.Errorf("keeping out %+v, ranking %+v", got.KeepingOut, got.Ranking)
	}
}

func TestAiSearchBeforeAnyRunOrScan(t *testing.T) {
	w := newWorld()
	w.runs, w.readiness = nil, map[string]domain.Readiness{}
	w.companies = w.companies[1:] // no Tribelt on the list either
	got, err := service(w).AiSearch(context.Background())
	if err != nil || len(got.Trend) != 0 || len(got.Followed) != 0 || got.Questions != 0 || got.Crawlers[0].Access != "" {
		t.Fatalf("summary = %+v, %v", got, err)
	}
}

func TestAiSearchSaysWhichSourceFailed(t *testing.T) {
	for _, breaks := range []func(*world){
		func(w *world) { w.companiesErr = errDown },
		func(w *world) { w.runsErr = errDown },
		func(w *world) { w.readyErr = errDown },
		func(w *world) { w.targetsErr = errDown },
	} {
		w := newWorld()
		breaks(w)
		if _, err := service(w).AiSearch(context.Background()); !errors.Is(err, errDown) {
			t.Errorf("err = %v", err)
		}
	}
}
