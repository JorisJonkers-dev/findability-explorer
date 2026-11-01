package domain_test

import (
	"testing"
	"time"

	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain"
)

func TestANameSplitOffSaysWhichCandidateItCameFrom(t *testing.T) {
	day := time.Date(2026, 10, 27, 8, 0, 0, 0, time.UTC)
	sightings := []domain.Sighting{
		{Name: "Jonge Poerink Conveyors", URL: "jpconveyors.com/", Provider: "openai", At: day},
		{Name: "Jonge Poerink", Provider: "openai", At: day},
	}
	lead, other := idOf("Jonge Poerink Conveyors"), idOf("Jonge Poerink")
	merged := []domain.Merge{{CandidateID: other, IntoID: lead, By: domain.ModelMerger, At: day}}
	if got := domain.Candidates(sightings, nil, nil, merged); len(got) != 1 || got[0].SplitFrom != "" {
		t.Fatalf("merged = %+v", got)
	}
	split := []domain.Merge{{CandidateID: other, Separate: true, SplitFrom: lead, By: "joris", At: day}}
	got := domain.Candidates(sightings, nil, nil, split)
	if len(got) != 2 {
		t.Fatalf("split = %+v", got)
	}
	for _, c := range got {
		want := ""
		if c.ID == other {
			want = lead
		}
		if c.SplitFrom != want {
			t.Errorf("%s split from %q, want %q", c.Name, c.SplitFrom, want)
		}
	}
}

func TestARecheckChangesOnlyWhatTheModelDismissed(t *testing.T) {
	sure := domain.CompanyVerdict{Sure: true, Competitor: true}
	unsure := domain.CompanyVerdict{Competitor: true}
	notOne := domain.CompanyVerdict{Sure: true}
	withSite := domain.Candidate{Website: "https://example-belts.test/"}
	dismissed := func(by string) domain.Candidate {
		c := withSite
		c.Decision = &domain.CandidateDecision{Status: domain.CandidateDismissed, By: by}
		return c
	}
	added := withSite
	added.Decision = &domain.CandidateDecision{Status: domain.CandidateAdded, By: domain.ModelDecider}
	cases := []struct {
		name string
		c    domain.Candidate
		v    domain.CompanyVerdict
		want domain.CandidateStatus
		ok   bool
	}{
		{"a dismissed one now sure to compete is added", dismissed(domain.ModelDecider), sure, domain.CandidateAdded, true},
		{"a dismissed one the model is unsure of waits again", dismissed(domain.ModelDecider), unsure, domain.CandidatePending, true},
		{"a dismissed one still not a competitor stays", dismissed(domain.ModelDecider), notOne, "", false},
		{"one a person dismissed stays", dismissed("joris"), sure, "", false},
		{"one the model added stays on the list", added, notOne, "", false},
		{"a waiting one is decided as before", withSite, sure, domain.CandidateAdded, true},
	}
	for _, c := range cases {
		got, ok := c.v.Redecides(c.c)
		if got != c.want || ok != c.ok {
			t.Errorf("%s: %q, %v; want %q, %v", c.name, got, ok, c.want, c.ok)
		}
	}
}

func TestOnlyTheModelsDecisionsAreRechecked(t *testing.T) {
	byModel := domain.Candidate{ID: "a", Decision: &domain.CandidateDecision{Status: domain.CandidateDismissed, By: domain.ModelDecider}}
	byPerson := domain.Candidate{ID: "b", Decision: &domain.CandidateDecision{Status: domain.CandidateAdded, By: "joris"}}
	waiting := domain.Candidate{ID: "c"}
	checked := domain.Candidate{ID: "d", Verdict: &domain.CompanyVerdict{}}
	all := []domain.Candidate{byModel, byPerson, waiting, checked}
	got := domain.ToRecheck(all)
	if len(got) != 2 || got[0].ID != "a" || got[1].ID != "c" {
		t.Errorf("to recheck = %+v", got)
	}
	if domain.DecidedByPerson(all) != 1 {
		t.Errorf("decided by a person = %d", domain.DecidedByPerson(all))
	}
	if w := domain.Waiting(all); len(w) != 2 || w[0].ID != "c" || w[1].ID != "d" {
		t.Errorf("waiting = %+v", w)
	}
}
