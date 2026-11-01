package domain_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/JorisJonkers-dev/findability-explorer/internal/claudestep/domain"
	llm "github.com/JorisJonkers-dev/findability-explorer/internal/llm/domain"
)

var at = time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC)

func matching(t *testing.T) domain.Definition {
	t.Helper()
	d, ok := domain.Lookup(domain.Matching)
	if !ok {
		t.Fatal("no matching step")
	}
	return d
}

func TestMatchingProductsIsAStepWithTheInstructionsTheToolCameWith(t *testing.T) {
	d := matching(t)
	if d.Title != "Matching products" || d.Fixed || !strings.Contains(d.Original, "Tribelt") {
		t.Errorf("matching = %+v", d)
	}
	if len(domain.Definitions()) == 0 || domain.Definitions()[0].ID != domain.Matching {
		t.Error("the steps do not list matching")
	}
	if _, ok := domain.Lookup("nothing"); ok {
		t.Error("an unknown step was found")
	}
	v := domain.Original(d)
	if v.Number != 1 || v.Model != llm.DefaultModel || v.Instructions != d.Original || v.By != domain.ByTheTool || v.Note == "" {
		t.Errorf("original = %+v", v)
	}
}

func TestTheNewestVersionIsCurrentAndWithoutAnyTheOriginal(t *testing.T) {
	d := matching(t)
	if got := domain.CurrentOf(d, nil); got.Number != 1 || got.Instructions != d.Original {
		t.Errorf("without versions = %+v", got)
	}
	vs := []domain.Version{domain.Original(d), {Number: 2, Model: "claude-sonnet-5-5", Instructions: "new"}}
	if got := domain.CurrentOf(d, vs); got.Number != 2 {
		t.Errorf("current = %+v", got)
	}
}

func TestANewVersionIsCheckedAndNumberedNext(t *testing.T) {
	d := matching(t)
	cur := domain.Original(d)
	v, err := domain.Next(d, cur, domain.Change{Model: "claude-sonnet-5-5", Instructions: "  Match the belts.  "}, "Joris", at)
	if err != nil {
		t.Fatal(err)
	}
	if v.Number != 2 || v.Model != "claude-sonnet-5-5" || v.Instructions != "Match the belts." || v.By != "Joris" || !v.At.Equal(at) || v.Note != "" {
		t.Errorf("next = %+v", v)
	}
	restored, err := domain.Next(d, v, domain.Change{Model: cur.Model, Instructions: cur.Instructions, RestoredFrom: 1}, "Joris", at)
	if err != nil || restored.Note != "Restored from version 1." || restored.Number != 3 {
		t.Errorf("restored = %+v %v", restored, err)
	}
	cases := map[string]domain.Change{
		"unknown model": {Model: "gpt-5", Instructions: "x"},
		"empty":         {Model: llm.DefaultModel, Instructions: "  "},
		"too long":      {Model: llm.DefaultModel, Instructions: strings.Repeat("a", domain.MaxInstructions+1)},
		"unchanged":     {Model: cur.Model, Instructions: cur.Instructions},
	}
	for name, ch := range cases {
		if _, err := domain.Next(d, cur, ch, "Joris", at); !errors.Is(err, domain.ErrInvalid) {
			t.Errorf("%s: err = %v", name, err)
		}
	}
}

func TestAStepWithInstructionsSetInTheCodeTakesOnlyANewModel(t *testing.T) {
	fixed := domain.Definition{ID: "fixed", Title: "Fixed", Fixed: true, Original: "Read it.", Typical: domain.Tokens{Input: 1500, Output: 250}}
	cur := domain.Original(fixed)
	if _, err := domain.Next(fixed, cur, domain.Change{Model: llm.DefaultModel, Instructions: "Other."}, "Joris", at); !errors.Is(err, domain.ErrInvalid) {
		t.Errorf("a fixed step took new instructions: %v", err)
	}
	v, err := domain.Next(fixed, cur, domain.Change{Model: "claude-sonnet-5-5", Instructions: "Read it."}, "Joris", at)
	if err != nil || v.Instructions != "Read it." || v.Model != "claude-sonnet-5-5" {
		t.Errorf("model only = %+v %v", v, err)
	}
	// Its instructions are always the code's, whatever a version stored.
	if got := domain.CurrentOf(fixed, []domain.Version{{Number: 2, Model: "claude-sonnet-5-5", Instructions: "stale"}}); got.Instructions != "Read it." {
		t.Errorf("fixed current = %+v", got)
	}
}

func TestAJobsCostScalesWithTheModelsPriceForTheStepsMixOfTokens(t *testing.T) {
	d := matching(t)
	if got := domain.JobCost(d, 0, llm.DefaultModel); got != llm.ListPrice(llm.DefaultModel).Of(d.Typical.Input, d.Typical.Output) {
		t.Errorf("without a base, the typical tokens at Haiku: %d", got)
	}
	if got := domain.JobCost(d, 1_000, llm.DefaultModel); got != 1_000 {
		t.Errorf("Haiku = %d, want the base", got)
	}
	haiku := llm.ListPrice(llm.DefaultModel).Of(d.Typical.Input, d.Typical.Output)
	sonnet := llm.ListPrice("claude-sonnet-5-5").Of(d.Typical.Input, d.Typical.Output)
	if got, want := domain.JobCost(d, 1_000, "claude-sonnet-5-5"), llm.Micros(1_000*int64(sonnet)/int64(haiku)); got != want {
		t.Errorf("Sonnet = %d, want %d", got, want)
	}
	if domain.Scale(d, "claude-sonnet-5-5") <= 1 || domain.Scale(d, llm.DefaultModel) != 1 {
		t.Error("scale")
	}
}

func TestAStepWithoutTypicalTokensCostsTheSameWithEveryModel(t *testing.T) {
	if got := domain.Scale(domain.Definition{}, "claude-opus-5-5"); got != 1 {
		t.Errorf("scale = %v", got)
	}
}
