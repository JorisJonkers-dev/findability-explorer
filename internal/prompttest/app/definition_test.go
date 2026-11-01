package app_test

import (
	"errors"
	"testing"

	llm "github.com/JorisJonkers-dev/findability-explorer/internal/llm/domain"
	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/app"
	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain"
	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain/domaintest"
)

// checkVersion is the competitor definition in use in these tests: the first stored version.
const checkVersion = 2

func TestTheDefinitionInUseIsTheNewestWithItsVersions(t *testing.T) {
	svc := reviewing(newRepo(), onTheList(), nil, nil)
	v, err := svc.CompetitorDefinition(t.Context())
	if err != nil || v.Current != domaintest.Definition() || len(v.Versions) != 1 {
		t.Fatalf("definition = %+v, %v", v, err)
	}
	if _, err := reviewing(newRepo("Definitions"), onTheList(), nil, nil).CompetitorDefinition(t.Context()); !errors.Is(err, errBoom) {
		t.Errorf("unreadable = %v", err)
	}
	empty := newRepo()
	empty.definitions = nil
	if _, err := reviewing(empty, onTheList(), nil, nil).CompetitorDefinition(t.Context()); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("none stored = %v", err)
	}
}

func TestSavingADefinitionMakesANewVersionAndSaysWhatToCheckAgain(t *testing.T) {
	r, _ := naming([2]string{"Kienzle GmbH", ""}, [2]string{"Ashworth", ""})
	// One competitor on the list and one waiting candidate were checked against version 2; one
	// candidate was never checked.
	r.checks = []domain.ListCheck{{CompanyID: "twentebelt", Reason: "Makes belts.", Version: checkVersion}}
	r.verdicts = []domain.CompanyVerdict{{CandidateID: idOf("Kienzle GmbH"), Reason: "Makes belts.", Version: checkVersion}}
	r.usage = []domain.ReviewUsage{{InputTokens: 20_000, OutputTokens: 2_000}}
	svc := reviewing(r, &directory{}, models{m: &model{verdicts: competitors}}, nil)

	if _, err := svc.SaveDefinition(t.Context(), "joris", " "); !errors.Is(err, domain.ErrInvalid) {
		t.Errorf("an empty definition = %v", err)
	}
	if _, err := svc.SaveDefinition(t.Context(), "joris", domaintest.Definition().Text+"\n"); !errors.Is(err, app.ErrUnchanged) {
		t.Errorf("the same text = %v", err)
	}
	v, err := svc.SaveDefinition(t.Context(), "joris", "  Only makers of eyelink belts compete.  ")
	if err != nil {
		t.Fatal(err)
	}
	if v.Current.Version != 3 || v.Current.Text != "Only makers of eyelink belts compete." || v.Current.SavedBy != "joris" || !v.Current.SavedAt.Equal(start) ||
		v.Current.RestoredFrom != 0 || len(v.Versions) != 2 || v.Versions[0].Version != 3 {
		t.Fatalf("saved = %+v", v)
	}
	// Two companies were checked before, with 22,000 tokens: 10,000 in and 1,000 out each.
	want := llm.TokenUse{Model: llm.DefaultModel, Input: 20_000, Output: 2_000}.Cost()
	if v.OutdatedCompetitors != 1 || v.OutdatedCandidates != 1 || v.RecheckCost != want {
		t.Errorf("to check again = %d competitors, %d candidates, %s", v.OutdatedCompetitors, v.OutdatedCandidates, v.RecheckCost)
	}
	// Saving starts no check: the candidate checked before waits to be checked again.
	if len(r.verdicts) != 1 {
		t.Errorf("a check ran: %+v", r.verdicts)
	}
	pending, counts, err := svc.Candidates(t.Context(), domain.CandidatePending)
	if err != nil || counts.Unchecked != 2 || pending[0].Verdict != nil {
		t.Errorf("after saving: %+v, %+v, %v", pending, counts, err)
	}
	if _, err := reviewing(newRepo("AddDefinition"), onTheList(), nil, nil).SaveDefinition(t.Context(), "joris", "x"); !errors.Is(err, errBoom) {
		t.Errorf("unstorable = %v", err)
	}
}

func TestAnOlderDefinitionIsRestoredAsANewVersion(t *testing.T) {
	r := newRepo()
	svc := reviewing(r, onTheList(), nil, nil)
	if _, err := svc.SaveDefinition(t.Context(), "joris", "Only makers of eyelink belts compete."); err != nil {
		t.Fatal(err)
	}
	v, err := svc.RestoreDefinition(t.Context(), "anna", 2)
	if err != nil {
		t.Fatal(err)
	}
	if v.Current.Version != 4 || v.Current.Text != domaintest.Definition().Text || v.Current.RestoredFrom != 2 || v.Current.SavedBy != "anna" || len(v.Versions) != 3 {
		t.Fatalf("restored = %+v", v)
	}
	if _, err := svc.RestoreDefinition(t.Context(), "anna", 9); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("an unknown version = %v", err)
	}
	if _, err := svc.RestoreDefinition(t.Context(), "anna", 2); !errors.Is(err, app.ErrUnchanged) {
		t.Errorf("the text in use = %v", err)
	}
}

func TestADefinitionWhoseFiguresCannotBeReadFails(t *testing.T) {
	for _, op := range []string{"ListChecks", "CompanyVerdicts", "Sightings", "CheckUsage"} {
		if _, err := reviewing(newRepo(op), onTheList(), nil, nil).CompetitorDefinition(t.Context()); !errors.Is(err, errBoom) {
			t.Errorf("%s failing: %v", op, err)
		}
	}
	if _, err := reviewing(newRepo(), &directory{err: errBoom}, nil, nil).CompetitorDefinition(t.Context()); !errors.Is(err, errBoom) {
		t.Errorf("the list failing: %v", err)
	}
	if _, err := reviewing(newRepo("Definitions"), onTheList(), nil, nil).SaveDefinition(t.Context(), "joris", "x"); !errors.Is(err, errBoom) {
		t.Errorf("save with the versions unreadable: %v", err)
	}
	if _, err := reviewing(newRepo("Definitions"), onTheList(), nil, nil).RestoreDefinition(t.Context(), "joris", 2); !errors.Is(err, errBoom) {
		t.Errorf("restore with the versions unreadable: %v", err)
	}
}

func TestChecksAreMadeAgainstTheDefinitionInUseAndRecordItsVersion(t *testing.T) {
	r, _ := naming([2]string{"Kienzle GmbH", ""}, [2]string{"Ashworth", ""})
	m := &model{verdicts: competitors}
	svc := reviewing(r, onTheList(), models{m: m}, nil)
	if _, err := svc.SaveDefinition(t.Context(), "joris", "Only makers of eyelink belts compete."); err != nil {
		t.Fatal(err)
	}
	if err := svc.Review(t.Context()); err != nil {
		t.Fatal(err)
	}
	if len(m.definitions) != 1 || m.definitions[0] != "Only makers of eyelink belts compete." || len(r.verdicts) == 0 || r.verdicts[0].Version != 3 {
		t.Errorf("checked against %q; verdicts %+v", m.definitions, r.verdicts)
	}
	if err := svc.CheckList(t.Context()); err != nil {
		t.Fatal(err)
	}
	if len(m.definitions) != 2 || m.definitions[1] != "Only makers of eyelink belts compete." || r.checks[0].Version != 3 {
		t.Errorf("list checked against %q; checks %+v", m.definitions, r.checks)
	}
	for _, op := range []string{"Definitions"} {
		failing := reviewing(newRepo(op), onTheList(), models{m: m}, nil)
		if err := failing.Review(t.Context()); !errors.Is(err, errBoom) {
			t.Errorf("Review with %s failing = %v", op, err)
		}
		if err := failing.CheckList(t.Context()); !errors.Is(err, errBoom) {
			t.Errorf("CheckList with %s failing = %v", op, err)
		}
		if _, _, err := failing.ListChecks(t.Context()); !errors.Is(err, errBoom) {
			t.Errorf("ListChecks with %s failing = %v", op, err)
		}
		if _, _, err := failing.Candidates(t.Context(), domain.CandidatePending); !errors.Is(err, errBoom) {
			t.Errorf("Candidates with %s failing = %v", op, err)
		}
	}
}

func TestTheEstimateOfACheckPricesEachCompanyFromTheChecksSoFar(t *testing.T) {
	r, _ := naming([2]string{"Kienzle GmbH", ""})
	r.checks = []domain.ListCheck{{CompanyID: "twentebelt", Reason: "Makes belts.", Version: checkVersion}}
	r.usage = []domain.ReviewUsage{{InputTokens: 10_000, OutputTokens: 1_000}}
	svc := reviewing(r, &directory{}, models{m: &model{verdicts: competitors}}, nil)
	got, err := svc.CheckEstimate(t.Context(), 3)
	if want := (llm.TokenUse{Model: llm.DefaultModel, Input: 30_000, Output: 3_000}).Cost(); err != nil || got != want {
		t.Errorf("three companies = %s, %v; want %s", got, err, want)
	}
	if got, err := svc.CheckEstimate(t.Context(), 0); err != nil || got != 0 {
		t.Errorf("no company = %s, %v", got, err)
	}
	broken, _ := naming([2]string{"Kienzle GmbH", ""})
	broken.failOn["CheckUsage"] = true
	if _, err := reviewing(broken, &directory{}, models{m: &model{verdicts: competitors}}, nil).CheckEstimate(t.Context(), 1); !errors.Is(err, errBoom) {
		t.Errorf("usage that cannot be read = %v", err)
	}
}
