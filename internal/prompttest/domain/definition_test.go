package domain_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	llm "github.com/JorisJonkers-dev/findability-explorer/internal/llm/domain"
	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain"
)

func TestADefinitionIsTrimmedAndHasText(t *testing.T) {
	if got, err := domain.CheckDefinition("  A competitor makes metal belts.\n"); err != nil || got != "A competitor makes metal belts." {
		t.Errorf("CheckDefinition = %q, %v", got, err)
	}
	for name, text := range map[string]string{"empty": " \n ", "too long": strings.Repeat("x", domain.MaxDefinition+1)} {
		if _, err := domain.CheckDefinition(text); !errors.Is(err, domain.ErrInvalid) {
			t.Errorf("%s: %v", name, err)
		}
	}
	if _, err := domain.CheckDefinition(strings.Repeat("é", domain.MaxDefinition)); err != nil {
		t.Errorf("the longest one, counted in letters: %v", err)
	}
}

func TestTheNewestDefinitionIsTheOneInUse(t *testing.T) {
	at := time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC)
	versions := []domain.Definition{{Version: 2, Text: "b", SavedAt: at}, {Version: 4, Text: "d"}, {Version: 3, Text: "c"}}
	if d, ok := domain.NewestDefinition(versions); !ok || d.Version != 4 {
		t.Errorf("NewestDefinition = %+v, %v", d, ok)
	}
	if _, ok := domain.NewestDefinition(nil); ok {
		t.Error("a newest of none")
	}
	if d, ok := domain.DefinitionVersion(versions, 3); !ok || d.Text != "c" {
		t.Errorf("DefinitionVersion 3 = %+v, %v", d, ok)
	}
	if _, ok := domain.DefinitionVersion(versions, 9); ok {
		t.Error("an unknown version was found")
	}
}

func TestCompaniesCheckedAgainstAnOlderDefinitionAreCounted(t *testing.T) {
	directory := []domain.DirectoryCompany{
		{ID: "old", Role: domain.RoleCompetitor}, {ID: "new", Role: domain.RoleCompetitor}, {ID: "never", Role: domain.RoleCompetitor},
		{ID: "kept", Role: domain.RoleCompetitor, Kept: true}, {ID: "tribelt", Role: domain.RoleTribelt},
	}
	checks := []domain.ListCheck{{CompanyID: "old", Version: 2}, {CompanyID: "new", Version: 3}, {CompanyID: "kept", Version: 2}, {CompanyID: "tribelt", Version: 2}}
	waiting := domain.Candidate{ID: "w", Members: []domain.Member{{ID: "w"}}}
	decided := domain.Candidate{ID: "d", Members: []domain.Member{{ID: "d"}}, Decision: &domain.CandidateDecision{Status: domain.CandidateAdded}}
	current := domain.Candidate{ID: "c", Members: []domain.Member{{ID: "c"}}}
	silent := domain.Candidate{ID: "s", Members: []domain.Member{{ID: "s"}}}
	verdicts := []domain.CompanyVerdict{
		{CandidateID: "w", Reason: "Makes belts.", Version: 2}, {CandidateID: "d", Reason: "Makes belts.", Version: 2},
		{CandidateID: "c", Reason: "Makes belts.", Version: 3}, {CandidateID: "s", Version: 2},
	}
	competitors, candidates := domain.Outdated(directory, checks, []domain.Candidate{waiting, decided, current, silent}, verdicts, 3)
	if competitors != 1 || candidates != 1 {
		t.Errorf("outdated = %d competitors, %d candidates", competitors, candidates)
	}
}

func TestCheckingAgainIsEstimatedFromWhatChecksUsedSoFar(t *testing.T) {
	haiku := func(in, out int64) llm.Micros { return llm.ListPrice(llm.DefaultModel).Of(in, out) }
	// 10 companies checked with 100,000 tokens in and 10,000 out: 10,000 and 1,000 each.
	if got := domain.EstimateChecks(4, domain.ReviewUsage{InputTokens: 100_000, OutputTokens: 10_000}, 10); got != haiku(40_000, 4_000) {
		t.Errorf("from history = %s", got)
	}
	// Before any check, a typical check.
	if got := domain.EstimateChecks(4, domain.ReviewUsage{}, 0); got != haiku(4*domain.TypicalCheckIn, 4*domain.TypicalCheckOut) || got == 0 {
		t.Errorf("before any = %s", got)
	}
	if got := domain.EstimateChecks(0, domain.ReviewUsage{InputTokens: 9}, 1); got != 0 {
		t.Errorf("nothing to check = %s", got)
	}
}
