package app_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/JorisJonkers-dev/findability-explorer/internal/terminology/app"
	"github.com/JorisJonkers-dev/findability-explorer/internal/terminology/domain"
)

type catalogue struct {
	terms     []domain.Term
	companies []domain.Company
}

func (c catalogue) Terms() []domain.Term        { return c.terms }
func (c catalogue) Companies() []domain.Company { return c.companies }

func many(n int) []domain.Term {
	out := make([]domain.Term, 0, n)
	for i := range n {
		out = append(out, domain.Term{Text: fmt.Sprintf("term %03d", i), Language: domain.LanguageEn, Family: domain.FamilyOther, Tier: domain.TierProduct})
	}
	return out
}

func TestSearchClampsTheLimit(t *testing.T) {
	svc := newService(catalogue{terms: many(10_200)})
	cases := map[int]int{0: app.DefaultLimit, -4: app.DefaultLimit, 7: 7, 50_000: app.MaxLimit}
	for limit, want := range cases {
		if got := svc.Search(domain.Query{Limit: limit}); len(got.Items) != want || got.Total != 10_200 {
			t.Errorf("Search(limit %d) = %d items of %d, want %d", limit, len(got.Items), got.Total, want)
		}
	}
	if len(svc.Terms()) != 10_200 {
		t.Error("Terms does not return the whole map")
	}
}

func TestCompaniesSorted(t *testing.T) {
	svc := newService(catalogue{companies: []domain.Company{{Name: "Zeta", GroupCompany: true}, {Name: "Beta"}, {Name: "alpha"}}})
	got := svc.Companies()
	if got[0].Name != "alpha" || got[1].Name != "Beta" || got[2].Name != "Zeta" {
		t.Fatalf("Companies = %+v", got)
	}
}

var errBoom = errors.New("boom")

var now = time.Date(2026, 10, 23, 9, 0, 0, 0, time.UTC)

// reviews is an in-memory Reviews; the err fields make a method fail.
type reviews struct {
	mu                          sync.Mutex
	decisions                   []domain.Decision
	listErr, decideErr, undoErr error
	listErrAfter, lists         int
}

func (r *reviews) Decisions(context.Context) ([]domain.Decision, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lists++
	if r.listErr != nil || (r.listErrAfter > 0 && r.lists >= r.listErrAfter) {
		return nil, errBoom
	}
	return r.decisions, nil
}

func (r *reviews) Decide(_ context.Context, d domain.Decision) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.decideErr != nil {
		return r.decideErr
	}
	r.decisions = append(r.decisions, d)
	return nil
}

func (r *reviews) Undo(_ context.Context, id string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.undoErr != nil {
		return false, r.undoErr
	}
	for i, d := range r.decisions {
		if d.CandidateID == id {
			r.decisions = append(r.decisions[:i], r.decisions[i+1:]...)
			return true, nil
		}
	}
	return false, nil
}

func newService(c catalogue) *app.Service {
	return app.New(c, &reviews{}, evidence(nil, nil), sightings(nil, nil), func() time.Time { return now })
}

func sightings(seen []domain.Sighting, err error) app.Sightings {
	return func(context.Context) ([]domain.Sighting, error) { return seen, err }
}

// The companies whose newest scan found a term use it: they are listed per term, sorted, once.
func TestFoundBy(t *testing.T) {
	seen := []domain.Sighting{
		{Text: "spiral belt", Language: "en", CompanyID: "x"},
		{Text: "spiral belt", Language: "en", CompanyID: "a"},
		{Text: "spiral belt", Language: "en", CompanyID: "x"},
		{Text: "Spiralband", Language: "de", CompanyID: "a"},
	}
	svc := app.New(catalogue{}, &reviews{}, evidence(nil, nil), sightings(seen, nil), func() time.Time { return now })
	got, err := svc.FoundBy(t.Context())
	if err != nil || len(got) != 2 || !slices.Equal(got[domain.TermKey{Text: "spiral belt", Language: "en"}], []string{"a", "x"}) ||
		!slices.Equal(got[domain.Term{Text: "Spiralband", Language: "de"}.Key()], []string{"a"}) {
		t.Fatalf("FoundBy = %v, %v", got, err)
	}
	failing := app.New(catalogue{}, &reviews{}, evidence(nil, nil), sightings(nil, errors.New("down")), func() time.Time { return now })
	if _, err := failing.FoundBy(t.Context()); err == nil {
		t.Fatal("FoundBy with failing sightings succeeded")
	}
}

func evidence(obs []domain.Observation, err error) app.Evidence {
	return func(context.Context) ([]domain.Observation, error) { return obs, err }
}

func observations() []domain.Observation {
	return []domain.Observation{
		{
			ScanID: "s1", CompanyID: "twentebelt", CompanyName: "Twentebelt", ScanURL: "https://www.twentebelt.com/", StartedAt: now,
			Phrase: "spiral freezer belts", Language: "en", Score: 30, Pages: 4, Tier: "application", Family: "tri_flex_turn_curve",
		},
		{
			ScanID: "s2", CompanyID: "martens", CompanyName: "Märtens", StartedAt: now.Add(-time.Hour),
			Phrase: "belt cleaning", Language: "en", Score: 5, Pages: 1, Tier: "broad",
		},
	}
}

func review(r *reviews) *app.Service {
	return app.New(catalogue{terms: many(3)}, r, evidence(observations(), nil), sightings(nil, nil), func() time.Time { return now })
}

// The map holds only Dutch, German and English terms, and only phrases in them are candidates,
// and those in no language that could be told, for a person to set it.
func TestTheMapKeepsToTheFocusLanguages(t *testing.T) {
	terms := []domain.Term{
		{Text: "Gliederband", Language: domain.LanguageDe},
		{Text: "bande", Language: "fr"},
		{Text: "spiral belt", Language: "en-US"},
		{Text: "draadogenband", Language: domain.LanguageNl},
	}
	obs := append(observations(),
		domain.Observation{ScanID: "s3", CompanyID: "x", Phrase: "nastro a spirale", Language: "it", Score: 9, Pages: 2},
		domain.Observation{ScanID: "s3", CompanyID: "x", Phrase: "1.4301", Language: "und", Score: 9, Pages: 2})
	svc := app.New(catalogue{terms: terms}, &reviews{}, evidence(obs, nil), sightings(nil, nil), func() time.Time { return now })
	var got []string
	for _, term := range svc.Terms() {
		got = append(got, term.Text)
	}
	if !slices.Equal(got, []string{"Gliederband", "spiral belt", "draadogenband"}) {
		t.Fatalf("Terms = %q", got)
	}
	candidates, _, err := svc.Candidates(t.Context(), domain.StatusPending)
	if err != nil || slices.ContainsFunc(candidates, func(c domain.Candidate) bool { return c.Phrase == "nastro a spirale" }) ||
		!slices.ContainsFunc(candidates, func(c domain.Candidate) bool { return c.Phrase == "1.4301" && c.Language == "und" }) {
		t.Fatalf("Candidates = %+v, %v", candidates, err)
	}
}

func TestAcceptAddsTheTermToTheMap(t *testing.T) {
	r := &reviews{}
	svc := review(r)
	if err := svc.Load(t.Context()); err != nil {
		t.Fatal(err)
	}
	pending, counts, err := svc.Candidates(t.Context(), domain.StatusPending)
	if err != nil || len(pending) != 2 || counts.Pending != 2 {
		t.Fatalf("Candidates = %+v, %+v, %v", pending, counts, err)
	}
	id := pending[0].ID
	c, err := svc.Accept(t.Context(), "joris", id, domain.Acceptance{
		Text: "spiral freezer belt", Language: "en", Family: "tri_flex_turn_curve", Tier: "application", UsedBy: []string{"Twentebelt"},
	})
	if err != nil || c.Status != domain.StatusAccepted || c.Decision.By != "joris" || !c.Decision.At.Equal(now) {
		t.Fatalf("Accept = %+v, %v", c, err)
	}
	terms := svc.Terms()
	if len(terms) != 4 || terms[3].Origin != domain.OriginReview || terms[3].Source != "https://www.twentebelt.com/" {
		t.Fatalf("map after Accept = %+v", terms)
	}
	if got := svc.Search(domain.Query{Text: "freezer"}); got.Total != 1 {
		t.Fatalf("search after Accept = %+v", got)
	}
	if _, err := svc.Accept(t.Context(), "joris", id, domain.Acceptance{}); !errors.Is(err, domain.ErrDecided) {
		t.Fatalf("second Accept = %v", err)
	}
	c, err = svc.Undo(t.Context(), id)
	if err != nil || c.Status != domain.StatusPending || len(svc.Terms()) != 3 {
		t.Fatalf("Undo = %+v, %v", c, err)
	}
	if _, err := svc.Undo(t.Context(), id); !errors.Is(err, domain.ErrPending) {
		t.Fatalf("second Undo = %v", err)
	}
}

func TestRejectStaysRejected(t *testing.T) {
	r := &reviews{}
	svc := review(r)
	pending, _, _ := svc.Candidates(t.Context(), domain.StatusPending)
	id := pending[1].ID
	c, err := svc.Reject(t.Context(), "joris", id)
	if err != nil || c.Status != domain.StatusRejected {
		t.Fatalf("Reject = %+v, %v", c, err)
	}
	if _, err := svc.Reject(t.Context(), "joris", id); !errors.Is(err, domain.ErrDecided) {
		t.Fatalf("second Reject = %v", err)
	}
	rejected, counts, err := svc.Candidates(t.Context(), domain.StatusRejected)
	if err != nil || len(rejected) != 1 || counts.Rejected != 1 || counts.Pending != 1 {
		t.Fatalf("rejected = %+v, %+v, %v", rejected, counts, err)
	}
}

func TestReviewFailures(t *testing.T) {
	ctx := t.Context()
	accept := domain.Acceptance{Text: "x", Language: "en", Family: "other", Tier: "broad"}
	id := domain.CandidateID("belt cleaning", "en")
	failing := func(r *reviews, e app.Evidence) *app.Service {
		return app.New(catalogue{}, r, e, sightings(nil, nil), func() time.Time { return now })
	}
	checks := map[string]func() error{
		"Load": func() error { return failing(&reviews{listErr: errBoom}, evidence(nil, nil)).Load(ctx) },
		"Candidates evidence": func() error {
			_, _, err := failing(&reviews{}, evidence(nil, errBoom)).Candidates(ctx, domain.StatusPending)
			return err
		},
		"Candidates decisions": func() error {
			_, _, err := failing(&reviews{listErr: errBoom}, evidence(nil, nil)).Candidates(ctx, domain.StatusPending)
			return err
		},
		"Accept unknown": func() error {
			_, err := failing(&reviews{}, evidence(nil, nil)).Accept(ctx, "u", "0000000000000000", accept)
			return err
		},
		"Accept evidence": func() error {
			_, err := failing(&reviews{}, evidence(nil, errBoom)).Accept(ctx, "u", id, accept)
			return err
		},
		"Accept invalid": func() error {
			_, err := failing(&reviews{}, evidence(observations(), nil)).Accept(ctx, "u", id, domain.Acceptance{})
			return err
		},
		"Accept store": func() error {
			_, err := failing(&reviews{decideErr: errBoom}, evidence(observations(), nil)).Accept(ctx, "u", id, accept)
			return err
		},
		"Accept reload": func() error {
			_, err := failing(&reviews{listErrAfter: 2}, evidence(observations(), nil)).Accept(ctx, "u", id, accept)
			return err
		},
		"Reject unknown": func() error { _, err := failing(&reviews{}, evidence(nil, nil)).Reject(ctx, "u", id); return err },
		"Undo unknown":   func() error { _, err := failing(&reviews{}, evidence(nil, nil)).Undo(ctx, id); return err },
		"Undo store": func() error {
			r := &reviews{decisions: []domain.Decision{{CandidateID: id, Status: domain.StatusRejected}}, undoErr: errBoom}
			_, err := failing(r, evidence(observations(), nil)).Undo(ctx, id)
			return err
		},
		"Undo reload": func() error {
			r := &reviews{decisions: []domain.Decision{{CandidateID: id, Status: domain.StatusRejected}}, listErrAfter: 2}
			_, err := failing(r, evidence(observations(), nil)).Undo(ctx, id)
			return err
		},
	}
	for name, check := range checks {
		if err := check(); err == nil {
			t.Errorf("%s did not fail", name)
		}
	}
}

// A decision made on a phrase before candidates were kept apart by language is undone by its own
// id, through the candidate it applies to.
func TestUndoRemovesAnOldDecisionOnThePhrase(t *testing.T) {
	r := &reviews{decisions: []domain.Decision{{CandidateID: domain.PhraseID("belt cleaning"), Phrase: "belt cleaning", Status: domain.StatusRejected}}}
	svc := review(r)
	c, err := svc.Undo(t.Context(), domain.CandidateID("belt cleaning", "en"))
	if err != nil || c.Status != domain.StatusPending || len(r.decisions) != 0 {
		t.Fatalf("Undo = %+v, %v; decisions left %+v", c, err, r.decisions)
	}
}

// A candidate becomes a term only in the language it was found in.
func TestAcceptKeepsToTheCandidatesLanguage(t *testing.T) {
	svc := review(&reviews{})
	_, err := svc.Accept(t.Context(), "joris", domain.CandidateID("belt cleaning", "en"), domain.Acceptance{
		Text: "belt cleaning", Language: "de", Family: "all", Tier: "broad",
	})
	if !errors.Is(err, domain.ErrOtherLanguage) {
		t.Fatalf("Accept in German = %v", err)
	}
}

// Once tribelt.nl has a finished scan, the scan says which terms it uses; before, the research.
func TestSiteUseKnowsWhetherTribeltWasScanned(t *testing.T) {
	seen := []domain.Sighting{{Text: "spiral belt", Language: "en", CompanyID: domain.TribeltID}}
	never := app.New(catalogue{}, &reviews{}, evidence(nil, nil), sightings(seen, nil), func() time.Time { return now })
	if u, err := never.SiteUse(t.Context()); err != nil || u.TribeltScanned || len(u.FoundBy) != 1 {
		t.Fatalf("SiteUse without scans = %+v, %v", u, err)
	}
	scanned := never.WithTribeltScans(func(context.Context) (bool, error) { return true, nil })
	if u, err := scanned.SiteUse(t.Context()); err != nil || !u.TribeltScanned || u.Source() != domain.UsageScan {
		t.Fatalf("SiteUse after a scan = %+v, %v", u, err)
	}
	if _, err := scanned.WithTribeltScans(func(context.Context) (bool, error) { return false, errBoom }).SiteUse(t.Context()); !errors.Is(err, errBoom) {
		t.Fatalf("a failing scan list = %v", err)
	}
	failing := app.New(catalogue{}, &reviews{}, evidence(nil, nil), sightings(nil, errBoom), func() time.Time { return now })
	if _, err := failing.SiteUse(t.Context()); !errors.Is(err, errBoom) {
		t.Fatalf("failing sightings = %v", err)
	}
}
