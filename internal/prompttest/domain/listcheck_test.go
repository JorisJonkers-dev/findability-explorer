package domain_test

import (
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain"
)

func listed() []domain.DirectoryCompany {
	return []domain.DirectoryCompany{
		{ID: "tribelt", Name: "Tribelt B.V.", Role: domain.RoleTribelt},
		{ID: "tribelt-mirror", Name: "Mirror of tribelt.nl", Role: domain.RoleMirror},
		{ID: "de-spiraal", Name: "De Spiraal", Role: domain.RoleGroup},
		{ID: "fenner", Name: "Fenner Dunlop", Host: "fennerdunlop.com", Role: domain.RoleCompetitor, Website: "https://www.fennerdunlop.com/"},
		{ID: "wire-belt", Name: "Wire Belt Company", OtherNames: []string{"WBC"}, Role: domain.RoleCompetitor},
		{ID: "gone", Name: "Gone", Role: domain.RoleCompetitor, Archived: true},
		{ID: "kept", Name: "Kept", Role: domain.RoleCompetitor, Kept: true},
		{ID: "old", Name: "Old", Role: domain.RoleCompetitor},
	}
}

func ids(cs []domain.DirectoryCompany) []string {
	var out []string
	for _, c := range cs {
		out = append(out, c.ID)
	}
	return out
}

// Only the competitors on the list that nobody kept are checked, each once per check version.
func TestToCheck(t *testing.T) {
	checks := []domain.ListCheck{
		{CompanyID: "wire-belt", Version: domain.CheckVersion},
		{CompanyID: "old", Version: domain.CheckVersion - 1},
		{CompanyID: "kept", Version: domain.CheckVersion - 1},
	}
	if got := ids(domain.ToCheck(listed(), checks)); !slices.Equal(got, []string{"fenner", "old"}) {
		t.Fatalf("ToCheck = %v", got)
	}
}

// A person sees the model's check of each competitor, but not an outdated one on a company it
// checks again.
func TestShownChecks(t *testing.T) {
	checks := []domain.ListCheck{
		{CompanyID: "tribelt", Version: domain.CheckVersion},
		{CompanyID: "wire-belt", Version: domain.CheckVersion},
		{CompanyID: "old", Version: domain.CheckVersion - 1},
		{CompanyID: "gone", Version: domain.CheckVersion - 1, Sure: true},
		{CompanyID: "kept", Version: domain.CheckVersion - 1},
		{CompanyID: "unknown", Version: domain.CheckVersion},
	}
	var got []string
	for _, c := range domain.ShownChecks(listed(), checks) {
		got = append(got, c.CompanyID)
	}
	if !slices.Equal(got, []string{"wire-belt", "gone", "kept"}) {
		t.Fatalf("ShownChecks = %v", got)
	}
}

func TestOnlyASureNonCompetitorIsArchived(t *testing.T) {
	if !(domain.ListCheck{Sure: true}).Archives() || (domain.ListCheck{}).Archives() || (domain.ListCheck{Sure: true, Competitor: true}).Archives() {
		t.Fatal("Archives is wrong")
	}
}

// The answers that name a company on the list count once each, matched when read or now.
func TestNamedIn(t *testing.T) {
	a, b, c, d, e := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	namings := []domain.Naming{
		{AnswerID: a, Name: "Fenner", CompetitorID: "fenner"},
		{AnswerID: a, Name: "Fenner Dunlop", CompetitorID: "fenner"},
		{AnswerID: b, Name: "Fenner Dunlop B.V.", URL: "https://fennerdunlop.com/nl"},
		{AnswerID: c, Name: "Fenner Dunlop"},
		{AnswerID: d, Name: "Fenner Dunlop"},
		{AnswerID: e, Name: "Nobody Belts"},
	}
	got := domain.NewMatcher(listed(), nil, nil).NamedIn(namings)
	if q := got["fenner"]; q.Answers != 4 || !slices.Equal(q.Quoted, []uuid.UUID{a, b, c}) || len(got) != 1 {
		t.Fatalf("NamedIn = %+v", got)
	}
}

// A company on the list is checked as a candidate: by its names, website and answers.
func TestAsCandidateAndBack(t *testing.T) {
	q := domain.Quotes{Answers: 2, Quoted: []uuid.UUID{uuid.New()}}
	c := domain.AsCandidate(listed()[4], q)
	if c.ID != "wire-belt" || c.Name != "Wire Belt Company" || c.Answers != 2 || len(c.Quoted) != 1 || c.Website != "" ||
		len(c.Members) != 2 || c.Members[1].Name != "WBC" {
		t.Fatalf("AsCandidate = %+v", c)
	}
	at := time.Date(2026, 10, 27, 9, 0, 0, 0, time.UTC)
	verdicts := domain.CompleteVerdicts([]domain.Candidate{c}, []domain.CompanyVerdict{{CandidateID: "wire-belt", Sure: true, Reason: "Makes wire belts."}}, "haiku", at)
	got := domain.ListChecksOf(verdicts)
	want := domain.ListCheck{CompanyID: "wire-belt", Sure: true, Reason: "Makes wire belts.", Model: "haiku", Version: domain.CheckVersion, At: at}
	if len(got) != 1 || got[0] != want {
		t.Fatalf("ListChecksOf = %+v", got)
	}
}
