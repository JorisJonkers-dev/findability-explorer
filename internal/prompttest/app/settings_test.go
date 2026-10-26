package app_test

import (
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/app"
	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain"
)

func ptr[T any](v T) *T { return &v }

func TestSettingsStartFromTheDefaults(t *testing.T) {
	r := newRepo()
	e := &env{repo: r, gw: newGateway(), dir: &directory{}, clock: &clock{now: start}, sleeps: &sleeps{}}
	e.svc = e.service("a")
	view, err := e.svc.Settings(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if view.Settings.CredentialID != "" || view.Settings.MonthlyBudget != domain.DefaultBudget || len(view.PromptSet.Prompts) != 90 ||
		view.ExtractorModel != domain.ExtractorModel || view.SpentThisMonth != 0 {
		t.Errorf("view = %+v", view)
	}
}

func TestUpdateChangesEachSetting(t *testing.T) {
	e := newEnv(t, newRepo(), "a")
	e.clock.set(time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC)) // a Tuesday
	view, err := e.svc.Update(t.Context(), "joris", app.Change{
		CredentialID: ptr("key"), Providers: []string{"OpenAI", "x-ai"}, Budget: ptr(domain.MicrosOf(25)), Cadence: ptr("weekly"),
		FactSheet: ptr("Tribelt makes metal conveyor belts in Haaksbergen."),
	})
	if err != nil {
		t.Fatal(err)
	}
	s := view.Settings
	if s.CredentialID != "key" || !slices.Equal(s.Providers, []string{"openai", "x-ai"}) || s.MonthlyBudget != 25_000_000 ||
		s.Cadence != domain.CadenceWeekly || !s.NextDue.Equal(time.Date(2026, 11, 2, 6, 0, 0, 0, time.UTC)) {
		t.Errorf("settings = %+v", s)
	}
	if s.FactSheet.Version != 2 || s.FactSheet.UpdatedBy != "joris" || !strings.HasPrefix(s.FactSheet.Text, "Tribelt makes") {
		t.Errorf("fact sheet = %+v", s.FactSheet)
	}
	// The same fact sheet again is no new version.
	view, _ = e.svc.Update(t.Context(), "joris", app.Change{FactSheet: ptr(s.FactSheet.Text)})
	if view.Settings.FactSheet.Version != 2 {
		t.Errorf("version = %d", view.Settings.FactSheet.Version)
	}
	// A run records the fact-sheet version it was judged against.
	run, err := e.svc.Start(t.Context(), "joris")
	if err != nil || run.FactSheetVersion != 2 || run.FactSheet != s.FactSheet.Text {
		t.Errorf("run = %+v, %v", run, err)
	}
}

func TestUpdateRefuses(t *testing.T) {
	cases := map[string]struct {
		change app.Change
		want   error
	}{
		"an unknown key":       {app.Change{CredentialID: ptr("other")}, domain.ErrNoKey},
		"no providers":         {app.Change{Providers: []string{}}, domain.ErrInvalid},
		"a budget of nothing":  {app.Change{Budget: ptr(domain.Micros(0))}, domain.ErrInvalid},
		"a daily schedule":     {app.Change{Cadence: ptr("daily")}, domain.ErrInvalid},
		"a fact sheet of none": {app.Change{FactSheet: ptr("")}, domain.ErrInvalid},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			e := newEnv(t, newRepo(), "a")
			if _, err := e.svc.Update(t.Context(), "joris", c.change); !errors.Is(err, c.want) || !app.IsInputError(err) {
				t.Fatalf("Update = %v, want %v", err, c.want)
			}
		})
	}
	for _, op := range []string{"Settings", "SaveSettings", "SpentSince"} {
		e := newEnv(t, newRepo(op), "a")
		if _, err := e.svc.Update(t.Context(), "joris", app.Change{}); !errors.Is(err, errBoom) {
			t.Errorf("%s failing: %v", op, err)
		}
	}
}

func TestResultsFailures(t *testing.T) {
	for _, op := range []string{"Runs", "Answers"} {
		e := newEnv(t, newRepo(), "a")
		e.repo.settings.Providers = []string{"openai"}
		if _, err := e.svc.Start(t.Context(), "joris"); err != nil {
			t.Fatal(err)
		}
		e.repo.failOn[op] = true
		if _, err := e.svc.Results(t.Context()); !errors.Is(err, errBoom) {
			t.Errorf("results with %s failing: %v", op, err)
		}
		if op == "Answers" {
			if _, _, err := e.svc.Answers(t.Context(), e.run(t).ID); !errors.Is(err, errBoom) {
				t.Errorf("answers failing: %v", err)
			}
		}
	}
	e := newEnv(t, newRepo(), "a")
	e.dir.err = errBoom
	if _, err := e.svc.Results(t.Context()); !errors.Is(err, errBoom) {
		t.Errorf("results without companies: %v", err)
	}
}

func TestAFailureIsClippedToASentence(t *testing.T) {
	e := newEnv(t, newRepo(), "a")
	e.repo.settings.Providers = []string{"openai"}
	e.gw.askErr = errors.New(strings.Repeat("very long ", 50))
	if _, err := e.svc.Start(t.Context(), "joris"); err != nil {
		t.Fatal(err)
	}
	if f := e.repo.answers[0].Failure; len([]rune(f)) != 300 || !strings.HasPrefix(f, "Very long") || !strings.HasSuffix(f, "…") {
		t.Errorf("failure = %q", f)
	}
}

func TestCandidatesAreReviewedAddedDismissedAndUndone(t *testing.T) {
	e := newEnv(t, newRepo(), "a")
	e.repo.settings.Providers = []string{"openai"}
	if _, err := e.svc.Start(t.Context(), "joris"); err != nil {
		t.Fatal(err)
	}
	pending, counts, err := e.svc.Candidates(t.Context(), domain.CandidatePending)
	if err != nil || len(pending) != 1 || pending[0].Name != "Ashworth" || pending[0].Answers != 180 ||
		!slices.Equal(pending[0].Addresses, []string{"https://ashworth.com"}) || counts[domain.CandidatePending] != 1 {
		t.Fatalf("pending = %+v, %v, %v", pending, counts, err)
	}
	id := pending[0].ID
	added, err := e.svc.AddCandidate(t.Context(), "joris", id, "Ashworth Bros", "https://ashworth.com")
	if err != nil || added.Status() != domain.CandidateAdded || added.Decision.CompetitorID != "ashworth" {
		t.Fatalf("added = %+v, %v", added, err)
	}
	if _, err := e.svc.DismissCandidate(t.Context(), "joris", id); !errors.Is(err, app.ErrDecided) {
		t.Errorf("a decided candidate dismissed: %v", err)
	}
	all, counts, _ := e.svc.Candidates(t.Context(), "")
	if len(all) != 1 || counts[domain.CandidateAdded] != 1 {
		t.Errorf("all = %+v, %v", all, counts)
	}
	if err := e.svc.UndoCandidate(t.Context(), "joris", id); err != nil || len(e.dir.added) != 0 {
		t.Fatalf("undo add = %v, %d added", err, len(e.dir.added))
	}
	dismissed, err := e.svc.DismissCandidate(t.Context(), "joris", id)
	if err != nil || dismissed.Status() != domain.CandidateDismissed {
		t.Fatalf("dismissed = %+v, %v", dismissed, err)
	}
	// A later run names it again; it stays dismissed.
	if _, err := e.svc.Start(t.Context(), "joris"); err != nil {
		t.Fatal(err)
	}
	if pending, _, _ := e.svc.Candidates(t.Context(), domain.CandidatePending); len(pending) != 0 {
		t.Errorf("a dismissed candidate came back: %+v", pending)
	}
	if err := e.svc.UndoCandidate(t.Context(), "joris", id); err != nil {
		t.Fatal(err)
	}
	if err := e.svc.UndoCandidate(t.Context(), "joris", id); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("undo of nothing = %v", err)
	}
	if _, err := e.svc.AddCandidate(t.Context(), "joris", "0000000000000000", "X", "https://x.test"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("an unknown candidate = %v", err)
	}
}

func TestCandidateFailures(t *testing.T) {
	setup := func(t *testing.T, failOn ...string) (*env, string) {
		t.Helper()
		e := newEnv(t, newRepo(), "a")
		e.repo.settings.Providers = []string{"openai"}
		if _, err := e.svc.Start(t.Context(), "joris"); err != nil {
			t.Fatal(err)
		}
		pending, _, _ := e.svc.Candidates(t.Context(), domain.CandidatePending)
		for _, f := range failOn {
			e.repo.failOn[f] = true
		}
		return e, pending[0].ID
	}
	for _, op := range []string{"Sightings", "CandidateDecisions"} {
		e, id := setup(t, op)
		if _, _, err := e.svc.Candidates(t.Context(), ""); !errors.Is(err, errBoom) {
			t.Errorf("%s failing: %v", op, err)
		}
		if _, err := e.svc.DismissCandidate(t.Context(), "joris", id); !errors.Is(err, errBoom) {
			t.Errorf("dismiss with %s failing: %v", op, err)
		}
	}
	e, _ := setup(t)
	e.dir.err = errBoom
	if _, _, err := e.svc.Candidates(t.Context(), ""); !errors.Is(err, errBoom) {
		t.Errorf("no companies: %v", err)
	}
	e, id := setup(t)
	e.dir.addErr = errBoom
	if _, err := e.svc.AddCandidate(t.Context(), "joris", id, "A", "https://a.test"); !errors.Is(err, errBoom) {
		t.Errorf("add failing: %v", err)
	}
	e, id = setup(t, "Decide")
	if _, err := e.svc.AddCandidate(t.Context(), "joris", id, "A", "https://a.test"); !errors.Is(err, errBoom) {
		t.Errorf("decide failing on add: %v", err)
	}
	if _, err := e.svc.DismissCandidate(t.Context(), "joris", id); !errors.Is(err, errBoom) {
		t.Errorf("decide failing on dismiss: %v", err)
	}
	e, id = setup(t)
	if _, err := e.svc.AddCandidate(t.Context(), "joris", id, "Ashworth", "https://ashworth.com"); err != nil {
		t.Fatal(err)
	}
	e.dir.removeErr = errBoom
	if err := e.svc.UndoCandidate(t.Context(), "joris", id); !errors.Is(err, errBoom) {
		t.Errorf("remove failing: %v", err)
	}
	e.repo.failOn["CandidateDecisions"] = true
	if err := e.svc.UndoCandidate(t.Context(), "joris", id); !errors.Is(err, errBoom) {
		t.Errorf("decisions failing on undo: %v", err)
	}
	_ = uuid.Nil
}
