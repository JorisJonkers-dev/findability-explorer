package domain_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain"
	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain/domaintest"
)

func cell(scope domain.Scope, mode domain.Mode, answers, named, cited int) domain.Cell {
	return domain.Cell{Provider: "openai", Scope: scope, Family: domain.FamilyEyelink, Mode: mode, Answers: answers, Named: named, Cited: cited}
}

func TestHistoryGivesTheSharesOverallPerLanguageForTwenteAndOfCitations(t *testing.T) {
	at := time.Date(2026, 10, 27, 8, 0, 0, 0, time.UTC)
	id := uuid.New()
	got := domain.History([]domain.RunFigureInput{{
		Run:      domain.Run{ID: id, PromptSetVersion: 2, StartedAt: at, Tasks: 40},
		Progress: domain.Progress{Answered: 40},
		Cells: []domain.Cell{
			cell(domain.ScopeTwente, domain.ModeKnowledge, 10, 5, 0),
			cell(domain.ScopeDutch, domain.ModeWeb, 10, 3, 4),
			cell(domain.ScopeGerman, domain.ModeWeb, 10, 1, 1),
			cell(domain.ScopeEnglish, domain.ModeKnowledge, 10, 0, 0),
		},
	}}, domaintest.Scopes())
	if len(got) != 1 {
		t.Fatalf("got %d runs", len(got))
	}
	r := got[0]
	want := map[string]float64{"all": 9.0 / 40, "nl": 8.0 / 20, "de": 0.1, "en": 0, "twente": 0.5, "citation": 5.0 / 20}
	shares := map[string]*float64{"all": r.All, "nl": r.Dutch, "de": r.German, "en": r.English, "twente": r.Twente, "citation": r.Citation}
	for k, w := range want {
		if shares[k] == nil || *shares[k] != w {
			t.Errorf("%s = %v, want %v", k, shares[k], w)
		}
	}
	if r.RunID != id || !r.StartedAt.Equal(at) || r.Version != 2 || r.Tasks != 40 || r.Answered != 40 || r.Read != 40 || r.Partial {
		t.Errorf("figures = %+v", r)
	}
}

func TestARunWithFewerThanHalfItsResponsesIsPartialAndHasNoShareWhereNothingWasRead(t *testing.T) {
	got := domain.History([]domain.RunFigureInput{{
		Run:      domain.Run{ID: uuid.New(), PromptSetVersion: 1, Tasks: 1260},
		Progress: domain.Progress{Answered: 62},
		Cells:    []domain.Cell{cell(domain.ScopeNetherlands, domain.ModeKnowledge, 47, 19, 0)},
	}}, domaintest.Scopes())
	r := got[0]
	if !r.Partial || r.Read != 47 || r.Dutch == nil || r.English != nil || r.German != nil || r.Twente != nil || r.Citation != nil {
		t.Fatalf("figures = %+v", r)
	}
	// Half its responses is not partial.
	half := domain.History([]domain.RunFigureInput{{
		Run: domain.Run{ID: uuid.New(), Tasks: 80}, Progress: domain.Progress{Answered: 40},
		Cells: []domain.Cell{cell(domain.ScopeGermany, domain.ModeWeb, 40, 0, 0)},
	}}, domaintest.Scopes())
	if half[0].Partial || half[0].German == nil || *half[0].German != 0 || half[0].Citation == nil || *half[0].Citation != 0 {
		t.Errorf("half = %+v", half[0])
	}
}

func TestHistoryLeavesOutARunWithNothingReadAndKeepsTheOrderGiven(t *testing.T) {
	first, last := uuid.New(), uuid.New()
	got := domain.History([]domain.RunFigureInput{
		{Run: domain.Run{ID: first, Tasks: 40}, Progress: domain.Progress{Answered: 40}, Cells: []domain.Cell{cell(domain.ScopeTwente, domain.ModeWeb, 40, 40, 0)}},
		{Run: domain.Run{ID: uuid.New(), Tasks: 200}},
		{Run: domain.Run{ID: last, Tasks: 40}, Progress: domain.Progress{Answered: 40}, Cells: []domain.Cell{cell(domain.ScopeTwente, domain.ModeWeb, 40, 20, 0)}},
	}, domaintest.Scopes())
	if len(got) != 2 || got[0].RunID != first || got[1].RunID != last {
		t.Fatalf("history = %+v, want the two runs with responses read, in order", got)
	}
	if empty := domain.History(nil, domaintest.Scopes()); empty == nil || len(empty) != 0 {
		t.Errorf("no runs = %#v, want an empty list", empty)
	}
}

func TestTheCommonestFailureIsTheReasonMostFailedResponsesGive(t *testing.T) {
	answers := []domain.Answer{
		{Failure: "The model is rate limited."},
		{Failure: "OpenRouter answered 401."},
		{},
		{Failure: "OpenRouter answered 401."},
		{Failure: "The model is rate limited."},
		{Failure: "A."},
	}
	if got := domain.CommonestFailure(answers); got != "OpenRouter answered 401." {
		t.Errorf("commonest = %q, a tie goes to the first alphabetically", got)
	}
	if got := domain.CommonestFailure([]domain.Answer{{}}); got != "" {
		t.Errorf("no failure = %q", got)
	}
}

func TestHistoryCountsAResponseInAnotherLanguageOnlyTowardsAll(t *testing.T) {
	lyon := domain.ScopeInfo{ID: "fr-lyon", Language: "fr", Region: "Lyon", Name: "French, Lyon"}
	got := domain.History([]domain.RunFigureInput{{
		Run:      domain.Run{ID: uuid.New(), Tasks: 20},
		Progress: domain.Progress{Answered: 20},
		Cells:    []domain.Cell{cell(domain.ScopeDutch, domain.ModeKnowledge, 10, 5, 0), cell("fr-lyon", domain.ModeKnowledge, 10, 10, 0), cell("mars", domain.ModeKnowledge, 0, 0, 0)},
	}}, append(domaintest.Scopes(), lyon))
	if len(got) != 1 || *got[0].All != 0.75 || *got[0].Dutch != 0.5 || got[0].German != nil {
		t.Fatalf("history = %+v", got)
	}
}
