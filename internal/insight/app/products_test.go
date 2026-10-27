package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/JorisJonkers-dev/findability-explorer/internal/insight/domain"
)

func TestProductsComeFromTheNamesTheMapAndTheNewestRun(t *testing.T) {
	w := newWorld()
	w.companies[1].BeltNames = []domain.BeltName{{Name: "Omni-Grid", Family: "tri_flex_turn_curve"}}
	w.mapTerms = []domain.MapTerm{{Text: "spiral belt", Family: "tri_flex_turn_curve", UsedBy: 3}}
	w.runs[0].Answers[0].Family = "tri_flex_turn_curve"
	got, err := service(w).Products(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	flex := got.Products[3]
	if len(got.Products) != 5 || flex.Family != "tri_flex_turn_curve" || len(flex.Sellers) != 1 || flex.Answers != 1 || len(got.Companies) != 3 {
		t.Errorf("products = %+v", got.Products)
	}
	if len(got.Standouts) != 2 || got.Standouts[0].Finding != domain.FindingCrowded || got.Standouts[1].Finding != domain.FindingFewestTerms {
		t.Errorf("standouts = %+v", got.Standouts)
	}
	w.runs = nil
	if got, err := service(w).Products(context.Background()); err != nil || got.Products[3].Answers != 0 {
		t.Errorf("before any run = %+v, %v", got, err)
	}
}

func TestProductsSayWhichSourceFailed(t *testing.T) {
	for _, breaks := range []func(*world){
		func(w *world) { w.companiesErr = errDown },
		func(w *world) { w.mapErr = errDown },
		func(w *world) { w.runsErr = errDown },
	} {
		w := newWorld()
		breaks(w)
		if _, err := service(w).Products(context.Background()); !errors.Is(err, errDown) {
			t.Errorf("err = %v", err)
		}
	}
}
