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

// onTheList is a company list beside Tribelt and Twentebelt: a group company, a rubber belting
// maker, a maker of plastic belts Haiku is unsure about, a wire belt maker, and one a person kept.
func onTheList() *directory {
	return &directory{added: []domain.DirectoryCompany{
		{ID: "de-spiraal", Name: "De Spiraal", Role: domain.RoleGroup},
		{ID: "fenner", Name: "Fenner Dunlop", Host: "fennerdunlop.com", Website: "https://www.fennerdunlop.com/", Role: domain.RoleCompetitor},
		{ID: "intralox", Name: "Intralox", Role: domain.RoleCompetitor},
		{ID: "wire-belt", Name: "Wire Belt Company", OtherNames: []string{"WBC"}, Role: domain.RoleCompetitor},
		{ID: "kept", Name: "Kept Rubber", Role: domain.RoleCompetitor, Kept: true},
	}}
}

// rubber is sure Fenner makes rubber belting, unsure about Intralox, and sure the rest compete.
func rubber(cs []domain.Candidate) []domain.CompanyVerdict {
	var out []domain.CompanyVerdict
	for _, c := range cs {
		v := domain.CompanyVerdict{CandidateID: c.ID, Competitor: true, Sure: true, Reason: "Makes metal belts."}
		switch c.ID {
		case "fenner", "kept":
			v.Competitor, v.Reason = false, "Makes rubber belting for quarries."
		case "intralox":
			v.Sure, v.Reason = false, "Makes plastic modular belts."
		}
		out = append(out, v)
	}
	return out
}

func TestHaikuArchivesTheListsSureNonCompetitors(t *testing.T) {
	r, _ := naming([2]string{"Fenner Dunlop", "https://www.fennerdunlop.com/"}, [2]string{"WBC", ""})
	d := onTheList()
	m := &model{verdicts: rubber}
	p := &pages{}
	svc := app.New(r, gateways{gw: newGateway()}, d, func() time.Time { return start }, app.Config{Owner: "a"}, func(uuid.UUID, error) {},
		app.WithSpawn(func(f func()) { f() }), app.WithReviewers(models{m: m}, p, func(err error) { t.Errorf("reported %v", err) }))

	checks, unchecked, err := svc.ListChecks(t.Context())
	if err != nil || len(checks) != 0 || unchecked != 4 {
		t.Fatalf("before: %+v, %d, %v", checks, unchecked, err)
	}
	if !svc.StartListCheck(t.Context()) {
		t.Fatal("the check did not start")
	}
	if st := svc.ListCheckState(); st.Running || st.Problem != app.ReviewProblemNone {
		t.Errorf("state = %+v", st)
	}
	// Tribelt, the group company and the kept one are never checked.
	if len(m.checked) != 1 || !slices.Equal(m.checked[0], []string{"twentebelt", "fenner", "intralox", "wire-belt"}) {
		t.Fatalf("checked %v", m.checked)
	}
	// Only the sure non-competitor is archived, by the language model.
	if !slices.Equal(d.archived, []string{"fenner"}) {
		t.Errorf("archived %v", d.archived)
	}
	// The evidence is that of a candidate: names, website, home page, and the answers.
	fenner, wbc := m.dossiers[1], m.dossiers[3]
	if fenner.Home == nil || fenner.Home.URL != "https://www.fennerdunlop.com/" || len(fenner.Snippets) != 1 || fenner.Candidate.Answers != 1 {
		t.Errorf("Fenner's dossier = %+v", fenner)
	}
	if wbc.Home != nil || len(wbc.Snippets) != 1 || !strings.Contains(wbc.Snippets[0], "WBC") {
		t.Errorf("Wire Belt's dossier = %+v", wbc)
	}
	if len(r.usage) != 1 || r.usage[0].At != start {
		t.Errorf("usage = %+v", r.usage)
	}
	checks, unchecked, err = svc.ListChecks(t.Context())
	if err != nil || len(checks) != 4 || unchecked != 0 || checks[0].Version != domain.CheckVersion || checks[0].Model != "anthropic: claude-haiku-5-5" {
		t.Fatalf("after: %+v, %d, %v", checks, unchecked, err)
	}
	// A second check asks nothing: every company is checked with this version.
	if err := svc.CheckList(t.Context()); err != nil || len(m.checked) != 1 {
		t.Errorf("again = %v, %d requests", err, len(m.checked))
	}
	// A check of an older version is made again, unless the company left the list.
	for i := range r.checks {
		r.checks[i].Version = domain.CheckVersion - 1
	}
	if err := svc.CheckList(t.Context()); err != nil || len(m.checked) != 2 || !slices.Equal(m.checked[1], []string{"twentebelt", "intralox", "wire-belt"}) {
		t.Errorf("after a new version = %v, %v", err, m.checked)
	}
}

// What the model does not answer on is an unsure check, so it is not asked again.
func TestACompanyTheModelSaysNothingAboutIsAskedAgain(t *testing.T) {
	r := newRepo()
	m := &model{verdicts: func([]domain.Candidate) []domain.CompanyVerdict { return nil }}
	svc := reviewing(r, onTheList(), models{m: m}, nil)
	if err := svc.CheckList(t.Context()); err != nil || len(r.checks) != 0 {
		t.Fatalf("CheckList = %v, %+v", err, r.checks)
	}
	if err := svc.CheckList(t.Context()); err != nil || len(m.checked) != 2 {
		t.Fatalf("a second check = %v, asked %d times", err, len(m.checked))
	}
}

func TestTheListCheckFails(t *testing.T) {
	if err := reviewing(newRepo(), onTheList(), nil, nil).CheckList(t.Context()); !errors.Is(err, app.ErrNoReviewer) {
		t.Errorf("without a reviewer = %v", err)
	}
	var reported []error
	svc := reviewing(newRepo(), onTheList(), models{}, func(err error) { reported = append(reported, err) })
	svc.StartListCheck(t.Context())
	if st := svc.ListCheckState(); st.Problem != app.ReviewProblemNoKey || len(reported) != 0 {
		t.Errorf("without a key: %+v, %v", st, reported)
	}
	svc = reviewing(newRepo(), onTheList(), models{err: errBoom}, func(err error) { reported = append(reported, err) })
	svc.StartListCheck(t.Context())
	if st := svc.ListCheckState(); st.Problem != app.ReviewProblemFails || len(reported) != 1 {
		t.Errorf("a failing model: %+v, %v", st, reported)
	}
	// Nothing to check asks nothing.
	quiet := &model{verdicts: rubber}
	r := newRepo()
	r.checks = []domain.ListCheck{{CompanyID: "twentebelt", Version: domain.CheckVersion}}
	if err := reviewing(r, &directory{}, models{m: quiet}, nil).CheckList(t.Context()); err != nil || len(quiet.checked) != 0 {
		t.Errorf("nothing to check = %v, %v", err, quiet.checked)
	}

	cases := map[string]struct {
		repo []string
		dir  func(*directory)
		m    *model
	}{
		"the list unreadable":    {dir: func(d *directory) { d.err = errBoom }},
		"the checks unreadable":  {repo: []string{"ListChecks"}},
		"the namings unreadable": {repo: []string{"Namings"}},
		"the decisions":          {repo: []string{"CandidateDecisions"}},
		"the merges":             {repo: []string{"Merges"}},
		"the answers' text":      {repo: []string{"AnswerTexts"}},
		"the model failing":      {m: &model{verdicts: rubber, checkErr: errBoom}},
		"storing the usage":      {repo: []string{"SaveReviewUsage"}},
		"storing the checks":     {repo: []string{"SaveListChecks"}},
		"archiving failing":      {dir: func(d *directory) { d.archiveErr = errBoom }},
	}
	for name, c := range cases {
		d := onTheList()
		if c.dir != nil {
			c.dir(d)
		}
		m := c.m
		if m == nil {
			m = &model{verdicts: rubber}
		}
		if err := reviewing(newRepo(c.repo...), d, models{m: m}, nil).CheckList(t.Context()); !errors.Is(err, errBoom) {
			t.Errorf("%s: CheckList = %v", name, err)
		}
	}
	for name, failOn := range map[string]string{"the checks": "ListChecks"} {
		if _, _, err := reviewing(newRepo(failOn), onTheList(), nil, nil).ListChecks(t.Context()); !errors.Is(err, errBoom) {
			t.Errorf("ListChecks with %s failing = %v", name, err)
		}
	}
	if _, _, err := reviewing(newRepo(), &directory{err: errBoom}, nil, nil).ListChecks(t.Context()); !errors.Is(err, errBoom) {
		t.Errorf("ListChecks with the list unreadable = %v", err)
	}
}

// One check runs at a time.
func TestOneListCheckAtATime(t *testing.T) {
	var started []func()
	svc := app.New(newRepo(), gateways{gw: newGateway()}, onTheList(), func() time.Time { return start }, app.Config{Owner: "a"}, func(uuid.UUID, error) {},
		app.WithSpawn(func(f func()) { started = append(started, f) }), app.WithReviewers(models{m: &model{verdicts: rubber}}, &pages{}, nil))
	if !svc.StartListCheck(t.Context()) || svc.StartListCheck(t.Context()) || !svc.ListCheckState().Running {
		t.Fatal("a second check started")
	}
	// The candidates' review is apart.
	if svc.ReviewState().Running {
		t.Error("the candidates' review runs too")
	}
	started[0]()
	if svc.ListCheckState().Running {
		t.Error("still running")
	}
}
