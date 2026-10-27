package app_test

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/app"
	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain"
)

// byName is Haiku judging by the start of each name: "Sure" a sure competitor, "Not" a sure
// non-competitor, anything else unsure.
func byName(cs []domain.Candidate) []domain.CompanyVerdict {
	var out []domain.CompanyVerdict
	for _, c := range cs {
		v := domain.CompanyVerdict{CandidateID: c.ID, Reason: "Makes eyelink belts for bakery ovens."}
		switch {
		case strings.HasPrefix(c.Name, "Sure"):
			v.Sure, v.Competitor = true, true
		case strings.HasPrefix(c.Name, "Not"):
			v.Sure = true
		}
		out = append(out, v)
	}
	return out
}

func TestHaikuDecidesWhatItIsSureAbout(t *testing.T) {
	r, _ := naming(
		[2]string{"Sure Belts", "https://www.surebelts.example/"},
		[2]string{"Sure Without Site", ""},
		[2]string{"Not Plastics", "https://notplastics.example/"},
		[2]string{"Maybe Mesh", ""},
		[2]string{"Old Check", ""},
		[2]string{"Person Dismissed", ""},
	)
	// Checked before the market was defined: checked again. Decided by a person: left alone.
	r.verdicts = []domain.CompanyVerdict{
		{CandidateID: idOf("Old Check"), Competitor: true, Sure: true, Version: 1, At: start},
		{CandidateID: idOf("Person Dismissed"), Sure: true, Version: 1, At: start},
	}
	r.decisions = []domain.CandidateDecision{{ID: idOf("Person Dismissed"), Name: "Person Dismissed", Status: domain.CandidateDismissed, By: "joris", At: start}}
	d := &directory{}
	m := &model{verdicts: byName}
	pg := &pages{}
	svc := app.New(r, gateways{gw: newGateway()}, d, func() time.Time { return start }, app.Config{Owner: "a"}, func(uuid.UUID, error) {},
		app.WithReviewers(models{m: m}, pg, nil))
	if _, counts, _ := svc.Candidates(t.Context(), ""); counts.Unchecked != 5 {
		t.Fatalf("unchecked before = %d", counts.Unchecked)
	}
	if err := svc.Review(t.Context()); err != nil {
		t.Fatal(err)
	}
	if len(m.checked) != 1 || len(m.checked[0]) != 5 || slices.Contains(m.checked[0], idOf("Person Dismissed")) {
		t.Fatalf("checked %v", m.checked)
	}
	// What Haiku read: the answers around the name, and the home page when there is a website.
	for _, ds := range m.dossiers {
		switch ds.Candidate.Name {
		case "Sure Belts":
			if ds.Home == nil || ds.Home.Title != "https://www.surebelts.example/" || !slices.Equal(ds.Snippets, []string{"Try Sure Belts for oven belts."}) {
				t.Errorf("dossier = %+v", ds)
			}
		case "Maybe Mesh":
			if ds.Home != nil || len(ds.Snippets) != 1 {
				t.Errorf("dossier = %+v", ds)
			}
		}
	}
	if len(pg.fetched) != 2 {
		t.Errorf("fetched %v", pg.fetched)
	}
	all, counts, _ := svc.Candidates(t.Context(), "")
	if counts.Added != 1 || counts.Dismissed != 2 || counts.Pending != 3 || counts.Unchecked != 0 {
		t.Errorf("counts = %+v", counts)
	}
	status := map[string]domain.CandidateStatus{}
	for _, c := range all {
		status[c.Name] = c.Status()
		if c.Decision != nil && c.Name != "Person Dismissed" && c.Decision.By != domain.ModelDecider {
			t.Errorf("%s decided by %q", c.Name, c.Decision.By)
		}
	}
	want := map[string]domain.CandidateStatus{
		"Sure Belts": domain.CandidateAdded, "Sure Without Site": domain.CandidatePending, "Not Plastics": domain.CandidateDismissed,
		"Maybe Mesh": domain.CandidatePending, "Old Check": domain.CandidatePending, "Person Dismissed": domain.CandidateDismissed,
	}
	for name, st := range want {
		if status[name] != st {
			t.Errorf("%s is %s, want %s", name, status[name], st)
		}
	}
	if len(d.added) != 1 || d.added[0].Name != "Sure Belts" || d.added[0].Host != "www.surebelts.example/" {
		t.Errorf("added = %+v", d.added)
	}
	if len(r.usage) != 2 || r.usage[0].InputTokens != 90 || !r.usage[1].At.Equal(start) {
		t.Errorf("usage = %+v", r.usage)
	}
	// A person undoes Haiku's decision: it waits for a person and is not decided again.
	if err := svc.UndoCandidate(t.Context(), "joris", idOf("Not Plastics")); err != nil {
		t.Fatal(err)
	}
	if err := svc.Review(t.Context()); err != nil || len(m.checked) != 1 {
		t.Errorf("a second review = %v, checked %d times", err, len(m.checked))
	}
	if pending, _, _ := svc.Candidates(t.Context(), domain.CandidatePending); len(pending) != 4 {
		t.Errorf("pending = %d", len(pending))
	}
}

func TestWhatHaikuCannotDecideWaits(t *testing.T) {
	sure := [2]string{"Sure Belts", "https://www.surebelts.example/"}
	// The list refuses the company: it waits for a person.
	r, _ := naming(sure)
	d := &directory{addErr: fmt.Errorf("%w: Sure Belts", domain.ErrCompanyKnown)}
	if err := reviewing(r, d, models{m: &model{verdicts: byName}}, nil).Review(t.Context()); err != nil || len(r.decisions) != 0 || len(r.verdicts) != 1 {
		t.Errorf("refused = %v; %d decisions, %d verdicts", err, len(r.decisions), len(r.verdicts))
	}
	// A person decides it while Haiku checks: Haiku leaves it.
	r, _ = naming(sure)
	person := reviewing(r, &directory{}, nil, nil)
	decided := &model{verdicts: func(cs []domain.Candidate) []domain.CompanyVerdict {
		if _, err := person.DismissCandidate(t.Context(), "joris", cs[0].ID); err != nil {
			t.Fatal(err)
		}
		return byName(cs)
	}}
	if err := reviewing(r, &directory{}, models{m: decided}, nil).Review(t.Context()); err != nil || len(r.decisions) != 1 || r.decisions[0].By != "joris" {
		t.Errorf("decided meanwhile = %v; %+v", err, r.decisions)
	}

	cases := map[string]struct {
		failOn string
		dir    *directory
		m      *model
	}{
		"reading the answers' text":   {"AnswerTexts", &directory{}, &model{verdicts: byName}},
		"storing what the check used": {"SaveReviewUsage", &directory{}, &model{verdicts: byName}},
		"the check failing":           {"", &directory{}, &model{checkErr: errBoom}},
		"the list failing to add":     {"", &directory{addErr: errBoom}, &model{verdicts: byName}},
		"recording the decision":      {"Decide", &directory{}, &model{verdicts: byName}},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			r, _ := naming(sure)
			if tc.failOn != "" {
				r.failOn[tc.failOn] = true
			}
			if err := reviewing(r, tc.dir, models{m: tc.m}, nil).Review(t.Context()); !errors.Is(err, errBoom) {
				t.Errorf("Review = %v", err)
			}
		})
	}
	// The candidates cannot be read again before deciding.
	r, _ = naming(sure)
	failing := &model{verdicts: func(cs []domain.Candidate) []domain.CompanyVerdict {
		r.failOn["Sightings"] = true
		return byName(cs)
	}}
	if err := reviewing(r, &directory{}, models{m: failing}, nil).Review(t.Context()); !errors.Is(err, errBoom) {
		t.Errorf("unreadable before deciding = %v", err)
	}
	// A provider that reports no tokens: nothing to store.
	r, _ = naming(sure)
	if err := reviewing(r, &directory{}, models{m: &model{verdicts: byName, free: true}}, nil).Review(t.Context()); err != nil || len(r.usage) != 0 {
		t.Errorf("no tokens = %v, %+v", err, r.usage)
	}
	// Merging that fails still counts what it used, and storing that can fail too.
	r, _ = naming(sure)
	r.failOn["SaveReviewUsage"] = true
	if err := reviewing(r, &directory{}, models{m: &model{mergeErr: errBoom}}, nil).Review(t.Context()); !errors.Is(err, errBoom) {
		t.Errorf("merging = %v", err)
	}
}
