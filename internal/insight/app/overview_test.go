package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/JorisJonkers-dev/findability-explorer/internal/insight/domain"
)

func TestTheOverviewCountsTheMapTheCompetitorsAndTheNewestRun(t *testing.T) {
	w := newWorld()
	w.terms = domain.TermMap{Terms: []domain.MapTerm{{Text: "spiral belt", Language: "en", UsedByTribelt: true}, {Text: "Spiralband", Language: "de"}}}
	got, err := service(w).Overview(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if w.asked != 1 || got.Terms.Total != 2 || got.Terms.Tribelt != 1 || got.Competitors.Makers != 2 || got.AiSearch.Answers != 2 {
		t.Errorf("overview = %+v, asked for %d", got, w.asked)
	}
	if c := got.AiSearch.Companies; len(c) != 2 || c[0].CompanyID != "ashworth" || c[0].Answers != 2 || c[1].CompanyID != "tribelt" || c[1].Answers != 1 {
		t.Errorf("named = %+v", c)
	}
	w.runs = nil
	if got, err := service(w).Overview(context.Background()); err != nil || got.AiSearch.Answers != 0 || len(got.AiSearch.Companies) != 0 {
		t.Errorf("before any run = %+v, %v", got, err)
	}
}

func TestTheOverviewSaysWhichSourceFailed(t *testing.T) {
	for _, breaks := range []func(*world){
		func(w *world) { w.companiesErr = errDown },
		func(w *world) { w.termsErr = errDown },
		func(w *world) { w.runsErr = errDown },
	} {
		w := newWorld()
		breaks(w)
		if _, err := service(w).Overview(context.Background()); !errors.Is(err, errDown) {
			t.Errorf("err = %v", err)
		}
	}
}
