package app_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/app"
	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain"
)

func TestASplitCanBeUndone(t *testing.T) {
	r, _ := naming([2]string{"Kienzle GmbH", "https://www.kienzle.com/"}, [2]string{"Kienzle GmbH", "https://www.kienzle.com/"}, [2]string{"Kienzle Drahtförderbänder", ""})
	r.merges = []domain.Merge{{CandidateID: idOf("Kienzle Drahtförderbänder"), IntoID: idOf("Kienzle GmbH"), By: domain.ModelMerger}}
	svc := reviewing(r, &directory{}, nil, nil)
	if err := svc.UndoSplit(t.Context(), "joris", idOf("Kienzle Drahtförderbänder")); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("undo before a split = %v", err)
	}
	if err := svc.Split(t.Context(), "joris", idOf("Kienzle Drahtförderbänder")); err != nil {
		t.Fatal(err)
	}
	pending, _, _ := svc.Candidates(t.Context(), domain.CandidatePending)
	i := slices.IndexFunc(pending, func(c domain.Candidate) bool { return c.ID == idOf("Kienzle Drahtförderbänder") })
	if len(pending) != 2 || i < 0 || pending[i].SplitFrom != idOf("Kienzle GmbH") {
		t.Fatalf("after the split = %+v", pending)
	}
	if err := svc.UndoSplit(t.Context(), "joris", idOf("Kienzle Drahtförderbänder")); err != nil {
		t.Fatal(err)
	}
	if again, _, _ := svc.Candidates(t.Context(), domain.CandidatePending); len(again) != 1 || len(again[0].Members) != 2 || again[0].SplitFrom != "" {
		t.Errorf("after undoing the split = %+v", again)
	}
	r.failOn["Merges"] = true
	if err := svc.UndoSplit(t.Context(), "joris", idOf("Kienzle Drahtförderbänder")); !errors.Is(err, errBoom) {
		t.Errorf("merges failing = %v", err)
	}
}

func TestTheCountsSayHowManyAnAdministratorDecided(t *testing.T) {
	r, _ := naming([2]string{"Kienzle GmbH", ""}, [2]string{"Twente Belt Works", ""}, [2]string{"Company 01", ""})
	r.decisions = []domain.CandidateDecision{
		{ID: idOf("Kienzle GmbH"), Status: domain.CandidateDismissed, By: "joris"},
		{ID: idOf("Twente Belt Works"), Status: domain.CandidateDismissed, By: domain.ModelDecider},
	}
	svc := reviewing(r, &directory{}, nil, nil)
	if _, counts, err := svc.Candidates(t.Context(), ""); err != nil || counts.ByPerson != 1 || counts.Dismissed != 2 || counts.Pending != 1 {
		t.Errorf("counts = %+v, %v", counts, err)
	}
}

// sureOf says every candidate whose name is in sure competes, sure, and every other one is unsure.
func sureOf(sure ...string) func([]domain.Candidate) []domain.CompanyVerdict {
	return func(cs []domain.Candidate) []domain.CompanyVerdict {
		var out []domain.CompanyVerdict
		for _, c := range cs {
			out = append(out, domain.CompanyVerdict{CandidateID: c.ID, Competitor: true, Sure: slices.Contains(sure, c.Name), Reason: "Reads its site."})
		}
		return out
	}
}

func TestARecheckAlsoAsksAboutWhatHaikuDecided(t *testing.T) {
	r, _ := naming(
		[2]string{"Example Belts", "https://example-belts.test/"},
		[2]string{"Sample Mesh", "https://sample-mesh.test/"},
		[2]string{"Person Kept", "https://person-kept.test/"},
		[2]string{"Waiting One", ""},
	)
	r.decisions = []domain.CandidateDecision{
		{ID: idOf("Example Belts"), Name: "Example Belts", Status: domain.CandidateDismissed, By: domain.ModelDecider},
		{ID: idOf("Sample Mesh"), Name: "Sample Mesh", Status: domain.CandidateDismissed, By: domain.ModelDecider},
		{ID: idOf("Person Kept"), Name: "Person Kept", Status: domain.CandidateDismissed, By: "joris"},
	}
	m := &model{verdicts: sureOf("Example Belts", "Person Kept")}
	d := &directory{}
	svc := reviewing(r, d, models{m: m}, func(err error) { t.Errorf("reported %v", err) })
	if !svc.StartRecheck(t.Context()) {
		t.Fatal("the recheck did not start")
	}
	if len(m.checked) != 1 || len(m.checked[0]) != 3 || slices.Contains(m.checked[0], idOf("Person Kept")) {
		t.Fatalf("checked = %v", m.checked)
	}
	all, counts, _ := svc.Candidates(t.Context(), "")
	status := map[string]domain.CandidateStatus{}
	for _, c := range all {
		status[c.Name] = c.Status()
	}
	// Haiku now adds the one it is sure of, sends the other it dismissed back to waiting, and
	// leaves what a person decided.
	if status["Example Belts"] != domain.CandidateAdded || status["Sample Mesh"] != domain.CandidatePending ||
		status["Person Kept"] != domain.CandidateDismissed || status["Waiting One"] != domain.CandidatePending || counts.ByPerson != 1 {
		t.Errorf("statuses = %v, counts %+v", status, counts)
	}
}

func TestCheckTheCandidatesOnlyAdvises(t *testing.T) {
	r, _ := naming([2]string{"Example Belts", "https://example-belts.test/"}, [2]string{"Waiting One", ""})
	r.verdicts = []domain.CompanyVerdict{{CandidateID: idOf("Waiting One"), Competitor: true, Reason: "Unsure.", Version: 1}}
	m := &model{verdicts: sureOf("Example Belts")}
	svc := reviewing(r, &directory{}, models{m: m}, func(err error) { t.Errorf("reported %v", err) })
	if !svc.StartCandidateCheck(t.Context()) {
		t.Fatal("the check did not start")
	}
	// Every waiting company is read, the one checked before too, and nothing is decided.
	if len(m.checked) != 1 || len(m.checked[0]) != 2 || m.merged != 0 {
		t.Fatalf("checked = %v, merged %d", m.checked, m.merged)
	}
	pending, counts, _ := svc.Candidates(t.Context(), domain.CandidatePending)
	if len(pending) != 2 || counts.Unchecked != 0 || pending[0].Verdict == nil || !pending[0].Verdict.Sure {
		t.Errorf("pending = %+v, counts %+v", pending, counts)
	}
	if st := svc.ReviewState(); st.Running || st.Problem != app.ReviewProblemNone {
		t.Errorf("state = %+v", st)
	}
}

func TestCheckTheCandidatesNeedsAKeyAndStops(t *testing.T) {
	r, _ := naming([2]string{"Example Belts", ""})
	if err := reviewing(r, &directory{}, nil, nil).Advise(t.Context()); !errors.Is(err, app.ErrNoReviewer) {
		t.Errorf("without reviewers = %v", err)
	}
	m := &model{verdicts: sureOf(), checkErr: errBoom}
	if err := reviewing(r, &directory{}, models{m: m}, nil).Advise(t.Context()); !errors.Is(err, errBoom) {
		t.Errorf("a failing check = %v", err)
	}
	for _, op := range []string{"CompanyVerdicts", "Sightings"} {
		r.failOn[op] = true
		if err := reviewing(r, &directory{}, models{m: &model{verdicts: sureOf()}}, nil).Advise(t.Context()); !errors.Is(err, errBoom) {
			t.Errorf("%s failing = %v", op, err)
		}
		r.failOn[op] = false
	}
	d := &directory{err: errBoom}
	if err := reviewing(r, d, models{m: &model{verdicts: sureOf()}}, nil).Advise(t.Context()); !errors.Is(err, errBoom) {
		t.Errorf("the list unreadable = %v", err)
	}
}

func TestARecheckSkipsACompanyListedMeanwhileAndReportsAFailingUndo(t *testing.T) {
	r, _ := naming([2]string{"Example Belts", "https://example-belts.test/"}, [2]string{"Sample Mesh", "https://sample-mesh.test/"})
	r.decisions = []domain.CandidateDecision{{ID: idOf("Sample Mesh"), Name: "Sample Mesh", Status: domain.CandidateDismissed, By: domain.ModelDecider}}
	d := &directory{}
	sure := sureOf("Example Belts", "Sample Mesh")
	m := &model{verdicts: func(cs []domain.Candidate) []domain.CompanyVerdict {
		// Someone adds Example Belts to the list while the model reads it.
		d.added = append(d.added, domain.DirectoryCompany{ID: "example-belts", Name: "Example Belts", Website: "https://example-belts.test/"})
		return sure(cs)
	}}
	r.failOn["Undecide"] = true
	if err := reviewing(r, d, models{m: m}, nil).Recheck(t.Context()); !errors.Is(err, errBoom) {
		t.Errorf("a failing undo = %v", err)
	}
	if slices.ContainsFunc(r.decisions, func(c domain.CandidateDecision) bool { return c.ID == idOf("Example Belts") }) {
		t.Errorf("the company listed meanwhile was decided: %+v", r.decisions)
	}
}
