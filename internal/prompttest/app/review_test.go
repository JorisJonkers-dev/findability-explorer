package app_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/app"
	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain"
)

func idOf(name string) string { return domain.CandidateID(domain.NameKey(name)) }

// naming is a repo with one finished run whose answers name these companies, none on the list.
func naming(names ...[2]string) (*repo, uuid.UUID) {
	r := newRepo()
	run := domain.Run{ID: uuid.New(), Status: domain.RunDone, StartedAt: start}
	r.runs = append(r.runs, &storedRun{run: run, owner: "a"})
	for i, n := range names {
		r.answers = append(r.answers, domain.Answer{
			ID: uuid.New(), RunID: run.ID, Provider: "openai", Model: "openai/gpt-oss-20b", At: start.Add(time.Duration(i) * time.Minute),
			Prompt: domain.Prompt{ID: "p"}, Mode: domain.ModeWeb,
			Reply:      domain.Reply{Text: "Try " + n[0] + " for oven belts."},
			Extraction: &domain.Extraction{Companies: []domain.Company{{Name: n[0], URL: n[1], Position: 1}}},
		})
	}
	return r, run.ID
}

// reviewing is a Service on r whose language model is rv; started reviews run in place.
func reviewing(r *repo, d *directory, rv app.Reviewers, onError func(error)) *app.Service {
	return app.New(r, gateways{gw: newGateway()}, d, func() time.Time { return start }, app.Config{Owner: "a"}, func(uuid.UUID, error) {},
		app.WithSpawn(func(f func()) { f() }), app.WithReviewers(rv, &pages{}, onError))
}

func TestACompanyIsDecidedAsOne(t *testing.T) {
	r, _ := naming(
		[2]string{"Taiwan Metiz Alliance", "https://metiz.tw/product_conveyor_belt.htm"},
		[2]string{"Taiwan Metiz Alliance", "metiz.tw"},
		[2]string{"Metiz", "http://www.metiz.tw/"},
		[2]string{"Kienzle GmbH", ""},
		[2]string{"Volta", "https://www.voltabelting.com/de/"},
	)
	d := &directory{}
	svc := reviewing(r, d, nil, nil)
	pending, counts, err := svc.Candidates(t.Context(), domain.CandidatePending)
	if err != nil || len(pending) != 3 || counts.Pending != 3 || counts.Unchecked != 3 || len(pending[0].Members) != 2 {
		t.Fatalf("pending = %+v, %+v, %v", pending, counts, err)
	}
	metiz := pending[0].ID
	// Dismissing one name dismisses the company: each of its names.
	if c, err := svc.DismissCandidate(t.Context(), "joris", idOf("Metiz")); err != nil || c.Status() != domain.CandidateDismissed || len(r.decisions) != 2 {
		t.Fatalf("dismiss = %+v, %v; %d decisions", c, err, len(r.decisions))
	}
	if err := svc.UndoCandidate(t.Context(), "joris", metiz); err != nil || len(r.decisions) != 0 {
		t.Fatalf("undo = %v; %d decisions", err, len(r.decisions))
	}

	got, err := svc.DecideMany(t.Context(), "joris", domain.CandidateAdded, []string{metiz, idOf("Metiz"), idOf("Kienzle GmbH"), "ffffffffffffffff"})
	if err != nil || len(got) != 3 {
		t.Fatalf("add several = %+v, %v", got, err)
	}
	if got[0].Outcome != app.OutcomeDone || got[0].Name != "Taiwan Metiz Alliance" || got[1].Outcome != app.OutcomeSkipped ||
		got[1].Reason == "" || got[2].Outcome != app.OutcomeSkipped || got[2].ID != "ffffffffffffffff" {
		t.Errorf("outcomes = %+v", got)
	}
	if len(d.added) != 1 || d.added[0].Host != "metiz.tw/" {
		t.Errorf("added = %+v", d.added)
	}
	// Undoing an added company removes it once, and every name's decision.
	if err := svc.UndoCandidate(t.Context(), "joris", idOf("Metiz")); err != nil || len(d.added) != 0 || len(r.decisions) != 0 {
		t.Fatalf("undo add = %v; %d added, %d decisions", err, len(d.added), len(r.decisions))
	}
	if got, err := svc.DecideMany(t.Context(), "joris", domain.CandidateDismissed, []string{idOf("Volta")}); err != nil || got[0].Outcome != app.OutcomeDone {
		t.Errorf("dismiss several = %+v, %v", got, err)
	}
	if _, err := svc.DecideMany(t.Context(), "joris", domain.CandidatePending, []string{metiz}); !errors.Is(err, app.ErrUnknownAction) {
		t.Errorf("an unknown action = %v", err)
	}
	// The list refuses one: it is skipped. Another failure stops it.
	d.addErr = fmt.Errorf("%w: Metiz", domain.ErrCompanyKnown)
	if got, err := svc.DecideMany(t.Context(), "joris", domain.CandidateAdded, []string{metiz}); err != nil || got[0].Outcome != app.OutcomeSkipped {
		t.Errorf("a company on the list = %+v, %v", got, err)
	}
	d.addErr = errBoom
	if _, err := svc.DecideMany(t.Context(), "joris", domain.CandidateAdded, []string{metiz}); !errors.Is(err, errBoom) {
		t.Errorf("the list failing = %v", err)
	}
	d.addErr = nil
	r.failOn["Decide"] = true
	if _, err := svc.DecideMany(t.Context(), "joris", domain.CandidateDismissed, []string{metiz}); !errors.Is(err, errBoom) {
		t.Errorf("deciding failing = %v", err)
	}
	r.failOn["Decide"] = false
	d.err = errBoom
	if _, err := svc.DecideMany(t.Context(), "joris", domain.CandidateDismissed, []string{metiz}); !errors.Is(err, errBoom) {
		t.Errorf("the list unreadable = %v", err)
	}
	if err := svc.UndoCandidate(t.Context(), "joris", metiz); !errors.Is(err, errBoom) {
		t.Errorf("undo with the list unreadable = %v", err)
	}
	d.err = nil
	r.failOn["Undecide"] = true
	if _, err := svc.DismissCandidate(t.Context(), "joris", metiz); err != nil {
		t.Fatal(err)
	}
	if err := svc.UndoCandidate(t.Context(), "joris", metiz); !errors.Is(err, errBoom) {
		t.Errorf("undeciding failing = %v", err)
	}
}

func TestANameMergedIntoACompanyCanBeSplitOff(t *testing.T) {
	r, _ := naming([2]string{"Kienzle GmbH", "https://www.kienzle.com/"}, [2]string{"Kienzle Drahtförderbänder", ""}, [2]string{"Kienzle Belts", "kienzle.com"})
	r.merges = []domain.Merge{{CandidateID: idOf("Kienzle Drahtförderbänder"), IntoID: idOf("Kienzle GmbH"), By: domain.ModelMerger}}
	d := &directory{}
	svc := reviewing(r, d, nil, nil)
	if err := svc.Split(t.Context(), "joris", idOf("Kienzle Drahtförderbänder")); err != nil {
		t.Fatal(err)
	}
	if pending, _, _ := svc.Candidates(t.Context(), domain.CandidatePending); len(pending) != 2 || !r.merges[0].Separate || r.merges[0].By != "joris" {
		t.Errorf("after the split: %d candidates, merges %+v", len(pending), r.merges)
	}
	for id, want := range map[string]error{
		idOf("Kienzle GmbH"):  app.ErrNotMerged, // the lead
		idOf("Kienzle Belts"): app.ErrNotMerged, // on the same website
		"ffffffffffffffff":    domain.ErrNotFound,
	} {
		if err := svc.Split(t.Context(), "joris", id); !errors.Is(err, want) {
			t.Errorf("Split(%s) = %v, want %v", id, err, want)
		}
	}
	d.err = errBoom
	if err := svc.Split(t.Context(), "joris", idOf("Kienzle GmbH")); !errors.Is(err, errBoom) {
		t.Errorf("the list unreadable = %v", err)
	}
	d.err = nil
	for _, op := range []string{"Merges", "CompanyVerdicts"} {
		r.failOn[op] = true
		if _, _, err := svc.Candidates(t.Context(), ""); !errors.Is(err, errBoom) {
			t.Errorf("%s failing = %v", op, err)
		}
		r.failOn[op] = false
	}
}

func TestAnswersCountACompanyAddedSince(t *testing.T) {
	r, runID := naming([2]string{"Volta Belting", "https://www.voltabelting.com/"}, [2]string{"Volta", ""})
	d := &directory{}
	svc := reviewing(r, d, nil, nil)
	if _, err := svc.AddCandidate(t.Context(), "joris", idOf("Volta Belting"), "Volta Belting", "https://www.voltabelting.com/"); err != nil {
		t.Fatal(err)
	}
	matched := func(answers []domain.Answer) []string {
		var out []string
		for _, a := range answers {
			out = append(out, a.Extraction.Companies[0].CompetitorID)
		}
		return out
	}
	_, answers, err := svc.Answers(t.Context(), runID, app.AnswerFilter{})
	if err != nil || !slices.Equal(matched(answers), []string{"volta", "volta"}) {
		t.Fatalf("answers = %v, %v", matched(answers), err)
	}
	if r.answers[0].Extraction.Companies[0].CompetitorID != "" {
		t.Error("the stored answer changed")
	}
	counted, err := svc.Counted(t.Context(), 1)
	if err != nil || len(counted) != 1 || !slices.Equal(matched(counted[0].Answers), []string{"volta", "volta"}) {
		t.Errorf("counted = %+v, %v", counted, err)
	}
	if _, err := svc.Results(t.Context()); err != nil {
		t.Errorf("results = %v", err)
	}
	for _, op := range []string{"Merges", "CandidateDecisions"} {
		r.failOn[op] = true
		if _, err := svc.Results(t.Context()); !errors.Is(err, errBoom) {
			t.Errorf("results with %s failing = %v", op, err)
		}
		if _, _, err := svc.Answers(t.Context(), runID, app.AnswerFilter{}); !errors.Is(err, errBoom) {
			t.Errorf("answers with %s failing = %v", op, err)
		}
		if _, err := svc.Counted(t.Context(), 1); !errors.Is(err, errBoom) {
			t.Errorf("counted with %s failing = %v", op, err)
		}
		r.failOn[op] = false
	}
	d.err = errBoom
	if _, err := svc.Results(t.Context()); !errors.Is(err, errBoom) {
		t.Errorf("results with the list unreadable = %v", err)
	}
}

// model is a language model that merges and checks by the rules given; each request uses 100
// tokens.
type model struct {
	mu       sync.Mutex
	clusters []domain.Cluster
	verdicts func([]domain.Candidate) []domain.CompanyVerdict
	mergeErr error
	checkErr error
	merged   int
	checked  [][]string
	dossiers []domain.Dossier
	// free uses no tokens, as a provider that reports none.
	free bool
}

func (m *model) usage() domain.ReviewUsage {
	if m.free {
		return domain.ReviewUsage{}
	}
	return used
}

var used = domain.ReviewUsage{InputTokens: 90, OutputTokens: 10}

func (m *model) Merge(_ context.Context, cs []domain.Candidate, _ []domain.DirectoryCompany) ([]domain.Cluster, domain.ReviewUsage, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.merged = len(cs)
	return m.clusters, m.usage(), m.mergeErr
}

func (m *model) Check(_ context.Context, ds []domain.Dossier) ([]domain.CompanyVerdict, domain.ReviewUsage, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var ids []string
	var cs []domain.Candidate
	for _, d := range ds {
		ids = append(ids, d.Candidate.ID)
		cs = append(cs, d.Candidate)
	}
	m.checked = append(m.checked, ids)
	m.dossiers = append(m.dossiers, ds...)
	if m.checkErr != nil {
		return nil, m.usage(), m.checkErr
	}
	return m.verdicts(cs), m.usage(), nil
}

// pages serves every home page with the website as its title.
type pages struct {
	mu      sync.Mutex
	fetched []string
}

func (p *pages) Fetch(_ context.Context, website string) domain.HomePage {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.fetched = append(p.fetched, website)
	return domain.HomePage{URL: website, Title: website}
}

type models struct {
	m   app.Reviewer
	err error
}

func (m models) Current(context.Context) (app.Reviewer, string, error) {
	return m.m, "anthropic: claude-haiku-5-5", m.err
}

// competitors takes every candidate with "Belt" in its name for a competitor, sure.
func competitors(cs []domain.Candidate) []domain.CompanyVerdict {
	var out []domain.CompanyVerdict
	for i, c := range cs {
		if i == len(cs)-1 {
			break // leaves the last one out
		}
		out = append(out, domain.CompanyVerdict{CandidateID: c.ID, Competitor: true, Sure: true, Reason: "Makes metal belts."})
	}
	return out
}

func TestHaikuMergesAndChecksTheCandidates(t *testing.T) {
	names := [][2]string{{"Kienzle GmbH", ""}, {"Kienzle Drahtförderbänder", ""}, {"Twente Belt Works", ""}}
	for i := range app.CheckBatch + 3 {
		names = append(names, [2]string{fmt.Sprintf("Company %02d", i), ""})
	}
	r, _ := naming(names...)
	m := &model{
		clusters: []domain.Cluster{{IDs: []string{idOf("Kienzle GmbH"), idOf("Kienzle Drahtförderbänder")}}, {IDs: []string{idOf("Twente Belt Works")}, CompetitorID: "twentebelt"}},
		verdicts: competitors,
	}
	svc := reviewing(r, &directory{}, models{m: m}, func(err error) { t.Errorf("reported %v", err) })
	if !svc.StartReview(t.Context()) {
		t.Fatal("the review did not start")
	}
	if st := svc.ReviewState(); st.Running || st.Problem != app.ReviewProblemNone {
		t.Errorf("state = %+v", st)
	}
	pending, counts, _ := svc.Candidates(t.Context(), domain.CandidatePending)
	// Kienzle is one company now; Twente Belt Works is Twentebelt, on the list. The model said
	// nothing about the last company of each batch, so those two wait to be checked again.
	if counts.Pending != app.CheckBatch+4 || counts.Unchecked != 2 || m.merged != app.CheckBatch+6 || len(m.checked) != 2 {
		t.Fatalf("pending %d, unchecked %d; merged %d, checked %d times", counts.Pending, counts.Unchecked, m.merged, len(m.checked))
	}
	silent := 0
	for _, c := range pending {
		switch {
		case c.Verdict == nil:
			silent++
		case c.Verdict.Model != "anthropic: claude-haiku-5-5":
			t.Fatalf("%s has a check by %q", c.Name, c.Verdict.Model)
		}
	}
	if silent != 2 {
		t.Errorf("%d companies without a check, want 2", silent)
	}
	// A second review asks only about the two it said nothing about.
	if err := svc.Review(t.Context()); err != nil || len(m.checked) != 3 || len(m.checked[2]) != 2 {
		t.Errorf("a second review = %v, checked %v", err, m.checked)
	}
}

func TestAReviewWithoutAKeyOrFailing(t *testing.T) {
	r, _ := naming([2]string{"Kienzle GmbH", ""})
	plain := reviewing(r, &directory{}, nil, nil)
	if err := plain.Review(t.Context()); !errors.Is(err, app.ErrNoReviewer) {
		t.Errorf("without reviewers = %v", err)
	}
	plain.StartReview(t.Context())
	if st := plain.ReviewState(); st.Problem != app.ReviewProblemNoKey {
		t.Errorf("state = %+v", st)
	}
	if err := reviewing(r, &directory{}, models{}, nil).Review(t.Context()); !errors.Is(err, app.ErrNoReviewer) {
		t.Errorf("without a key = %v", err)
	}
	ok := func() *model { return &model{verdicts: competitors} }
	cases := map[string]struct {
		rv     app.Reviewers
		dir    *directory
		failOn string
	}{
		"the settings":        {models{err: errBoom}, &directory{}, ""},
		"the list":            {models{m: ok()}, &directory{err: errBoom}, ""},
		"merging":             {models{m: &model{mergeErr: errBoom}}, &directory{}, ""},
		"checking":            {models{m: &model{checkErr: errBoom}}, &directory{}, ""},
		"reading the merges":  {models{m: ok()}, &directory{}, "Merges"},
		"saving the merges":   {models{m: &model{clusters: []domain.Cluster{{IDs: []string{idOf("Kienzle GmbH")}, CompetitorID: "twentebelt"}}}}, &directory{}, "SaveMerges"},
		"saving the checks":   {models{m: ok()}, &directory{}, "SaveCompanyVerdicts"},
		"reading the answers": {models{m: ok()}, &directory{}, "Sightings"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			r, _ := naming([2]string{"Kienzle GmbH", ""})
			if tc.failOn != "" {
				r.failOn[tc.failOn] = true
			}
			var reported []error
			svc := reviewing(r, tc.dir, tc.rv, func(err error) { reported = append(reported, err) })
			svc.StartReview(t.Context())
			if st := svc.ReviewState(); st.Problem != app.ReviewProblemFails || len(reported) != 1 || !errors.Is(reported[0], errBoom) {
				t.Errorf("state %+v, reported %v", st, reported)
			}
		})
	}
	// Checking stops when the answers can no longer be read after merging.
	r2, _ := naming([2]string{"Kienzle GmbH", ""})
	var once sync.Once
	merging := &model{verdicts: competitors}
	svc := reviewing(r2, &directory{}, models{m: hook{merging, func() { once.Do(func() { r2.failOn["CompanyVerdicts"] = true }) }}}, nil)
	if err := svc.Review(t.Context()); !errors.Is(err, errBoom) {
		t.Errorf("unreadable after merging = %v", err)
	}
	// Nothing waiting: nothing to merge.
	empty := newRepo()
	if err := reviewing(empty, &directory{}, models{m: ok()}, nil).Review(t.Context()); err != nil {
		t.Errorf("an empty list = %v", err)
	}
}

// hook runs after a merge, before the check.
type hook struct {
	*model
	after func()
}

func (h hook) Merge(ctx context.Context, cs []domain.Candidate, d []domain.DirectoryCompany) ([]domain.Cluster, domain.ReviewUsage, error) {
	out, u, err := h.model.Merge(ctx, cs, d)
	h.after()
	return out, u, err
}

func TestOneReviewRunsAtATime(t *testing.T) {
	r, _ := naming([2]string{"Kienzle GmbH", ""})
	var later func()
	svc := app.New(r, gateways{gw: newGateway()}, &directory{}, func() time.Time { return start }, app.Config{Owner: "a"}, func(uuid.UUID, error) {},
		app.WithSpawn(func(f func()) { later = f }), app.WithReviewers(models{m: &model{verdicts: competitors}}, &pages{}, func(error) {}))
	if !svc.StartReview(t.Context()) || svc.StartReview(t.Context()) || !svc.ReviewState().Running {
		t.Fatal("a second review started")
	}
	later()
	if svc.ReviewState().Running {
		t.Error("still running")
	}
}

func TestTheReviewStopsWhenTheMonthsTokensAreSpent(t *testing.T) {
	names := make([][2]string, 0, app.CheckBatch+3)
	for i := range app.CheckBatch + 3 {
		names = append(names, [2]string{fmt.Sprintf("Company %02d", i), ""})
	}
	review := func(spent app.BudgetSpent) (*app.Service, *model, *repo) {
		r, _ := naming(names...)
		m := &model{verdicts: competitors}
		svc := app.New(r, gateways{gw: newGateway()}, &directory{}, func() time.Time { return start }, app.Config{Owner: "a"}, func(uuid.UUID, error) {},
			app.WithSpawn(func(f func()) { f() }), app.WithReviewers(models{m: m}, &pages{}, func(err error) { t.Errorf("reported %v", err) }),
			app.WithBudget(spent))
		return svc, m, r
	}
	// Spent from the start: nothing is asked.
	svc, m, _ := review(func(context.Context) (bool, error) { return true, nil })
	svc.StartReview(t.Context())
	if st := svc.ReviewState(); st.Problem != app.ReviewProblemBudget || m.merged != 0 || len(m.checked) != 0 {
		t.Errorf("spent from the start: state %+v, merged %d, checked %d", st, m.merged, len(m.checked))
	}
	// Spent after the first batch: what it checked is kept (the model says nothing about the last
	// company of a batch, which is not stored).
	var checked *model
	svc, checked, r := review(func(context.Context) (bool, error) {
		checked.mu.Lock()
		defer checked.mu.Unlock()
		return len(checked.checked) >= 1, nil
	})
	if err := svc.Review(t.Context()); !errors.Is(err, app.ErrBudgetSpent) || checked.merged == 0 || len(checked.checked) != 1 || len(r.verdicts) != app.CheckBatch-1 {
		t.Errorf("spent after a batch = %v; checked %d times, %d verdicts", err, len(checked.checked), len(r.verdicts))
	}
	// The budget cannot be read: the review fails.
	failing, _, _ := review(func(context.Context) (bool, error) { return false, errBoom })
	if err := failing.Review(t.Context()); !errors.Is(err, errBoom) {
		t.Errorf("the budget unreadable = %v", err)
	}
}
