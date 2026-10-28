package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/JorisJonkers-dev/findability-explorer/internal/insight/app"
	"github.com/JorisJonkers-dev/findability-explorer/internal/insight/domain"
)

func TestThePlanFollowsTheTicksTheScanAndTheTargets(t *testing.T) {
	w := newWorld()
	w.readiness["tribelt"] = domain.Readiness{Crawlers: domain.AccessOpen, Markdown: true}
	w.set[domain.TargetShareOfVoice] = 0.5
	svc := service(w)
	if err := svc.Tick(context.Background(), "joris", "listings", true); err != nil {
		t.Fatal(err)
	}
	got, err := svc.Plan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	done := map[string]bool{}
	for _, it := range got.Items {
		done[it.ID] = it.Done
	}
	if !done["listings"] || !done["markdown-pages"] || done["llms-files"] || w.ticks[0].By != "joris" || !w.ticks[0].At.Equal(today) {
		t.Errorf("done = %v, ticks %+v", done, w.ticks)
	}
	if got.Watched[0].Goal != 0.5 || got.Watched[1].Goal != 0.75 || len(got.EyeOn) == 0 {
		t.Errorf("watched = %+v, eyes %+v", got.Watched, got.EyeOn)
	}
	if err := svc.Tick(context.Background(), "joris", "listings", false); err != nil || len(w.ticks) != 0 {
		t.Errorf("undo = %v, %+v", err, w.ticks)
	}
	st, err := svc.Standings(context.Background())
	if err != nil || len(st.Open) != app.OpenShown || st.Open[0].ID != "submit-search-engines" {
		t.Errorf("open = %+v, %v", st.Open, err)
	}
}

func TestTickingRefusals(t *testing.T) {
	w := newWorld()
	svc := service(w)
	if err := svc.Tick(context.Background(), "joris", "nothing", true); !errors.Is(err, domain.ErrUnknownImprovement) {
		t.Errorf("an unknown improvement = %v", err)
	}
	if err := svc.Tick(context.Background(), "joris", "markdown-pages", true); !errors.Is(err, domain.ErrCheckedByScan) || !app.IsInputError(err) {
		t.Errorf("an automatic improvement = %v", err)
	}
	w.writeErr = errDown
	for _, done := range []bool{true, false} {
		if err := svc.Tick(context.Background(), "joris", "listings", done); !errors.Is(err, errDown) {
			t.Errorf("a failing store, done %v = %v", done, err)
		}
	}
}

func TestSettingTargets(t *testing.T) {
	w := newWorld()
	svc := service(w)
	if err := svc.SetTargets(context.Background(), "joris", map[domain.Target]float64{domain.TargetTermsUsed: 40, domain.TargetCitationShare: 0.25}); err != nil {
		t.Fatal(err)
	}
	if w.set[domain.TargetTermsUsed] != 40 || w.set[domain.TargetCitationShare] != 0.25 || len(w.set) != 2 {
		t.Errorf("set = %v", w.set)
	}
	if err := svc.SetTargets(context.Background(), "joris", map[domain.Target]float64{domain.TargetTermsUsed: 50, domain.TargetShareOfVoice: 2}); !app.IsInputError(err) || w.set[domain.TargetTermsUsed] != 40 {
		t.Errorf("a bad target = %v, set %v", err, w.set)
	}
	w.writeErr = errDown
	if err := svc.SetTargets(context.Background(), "joris", map[domain.Target]float64{domain.TargetTermsUsed: 30}); !errors.Is(err, errDown) {
		t.Errorf("a failing store = %v", err)
	}
}

func TestThePlanSaysWhichSourceFailed(t *testing.T) {
	for _, breaks := range []func(*world){
		func(w *world) { w.companiesErr = errDown },
		func(w *world) { w.ticksErr = errDown },
		func(w *world) { w.targetsErr = errDown },
	} {
		w := newWorld()
		breaks(w)
		if _, err := service(w).Plan(context.Background()); !errors.Is(err, errDown) {
			t.Errorf("err = %v", err)
		}
	}
}
