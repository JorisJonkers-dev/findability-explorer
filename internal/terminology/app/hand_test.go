package app_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/JorisJonkers-dev/findability-explorer/internal/terminology/app"
	"github.com/JorisJonkers-dev/findability-explorer/internal/terminology/domain"
)

// hand is an in-memory HandTerms; failOn names the methods that fail.
type hand struct {
	added   []domain.HandTerm
	removed []domain.Removal
	failOn  map[string]bool
}

func (h *hand) fail(op string) error {
	if h.failOn[op] {
		return errBoom
	}
	return nil
}

func (h *hand) Added(context.Context) ([]domain.HandTerm, error) { return h.added, h.fail("Added") }

func (h *hand) Add(_ context.Context, t domain.HandTerm) error {
	if err := h.fail("Add"); err != nil {
		return err
	}
	h.added = append(h.added, t)
	return nil
}

func (h *hand) Removed(context.Context) ([]domain.Removal, error) { return h.removed, h.fail("Removed") }

func (h *hand) Remove(_ context.Context, r domain.Removal) error {
	if err := h.fail("Remove"); err != nil {
		return err
	}
	h.removed = append(h.removed, r)
	return nil
}

func (h *hand) PutBack(_ context.Context, key domain.TermKey) (bool, error) {
	if err := h.fail("PutBack"); err != nil {
		return false, err
	}
	n := len(h.removed)
	h.removed = slices.DeleteFunc(h.removed, func(r domain.Removal) bool { return r.Key == key })
	return len(h.removed) < n, nil
}

func handService(t *testing.T, h *hand, obs []domain.Observation) *app.Service {
	t.Helper()
	research := []domain.Term{
		{Text: "spiral belt", Language: domain.LanguageEn, Family: domain.FamilyOpenSpiral, Tier: domain.TierProduct, Origin: domain.OriginResearch},
		{Text: "Drahtösengliederband", Language: domain.LanguageDe, Family: domain.FamilyEyelink, Tier: domain.TierProduct, Origin: domain.OriginResearch},
	}
	svc := app.New(catalogue{terms: research}, &reviews{}, evidence(obs, nil), sightings(nil, nil), func() time.Time { return now }).WithHandTerms(h)
	if err := svc.Load(t.Context()); err != nil {
		t.Fatal(err)
	}
	return svc
}

func texts(terms []domain.Term) []string {
	out := make([]string, 0, len(terms))
	for _, t := range terms {
		out = append(out, t.Text)
	}
	return out
}

// An administrator adds a term by hand: it joins the list, marked as added by hand, and is listed
// with who added it; a term the list has already is refused.
func TestAddATermByHand(t *testing.T) {
	h := &hand{}
	svc := handService(t, h, nil)
	got, err := svc.AddTerm(t.Context(), "joris", domain.Acceptance{Text: "oven belt", Language: "en", Family: "eyelink", Tier: "application"})
	if err != nil || got.Origin != domain.OriginHand || got.Text != "oven belt" {
		t.Fatalf("AddTerm = %+v, %v", got, err)
	}
	if !slices.Equal(texts(svc.Terms()), []string{"spiral belt", "Drahtösengliederband", "oven belt"}) {
		t.Errorf("terms = %v", texts(svc.Terms()))
	}
	if added := svc.HandTerms(); len(added) != 1 || added[0].By != "joris" || !added[0].At.Equal(now) || added[0].Term.Text != "oven belt" {
		t.Errorf("hand terms = %+v", added)
	}
	if _, err := svc.AddTerm(t.Context(), "joris", domain.Acceptance{Text: "Oven Belt", Language: "en", Family: "other", Tier: "broad"}); !errors.Is(err, domain.ErrTermExists) {
		t.Errorf("adding it twice: %v", err)
	}
	h.failOn = map[string]bool{"Add": true}
	if _, err := svc.AddTerm(t.Context(), "joris", domain.Acceptance{Text: "belt", Language: "en", Family: "other", Tier: "broad"}); !errors.Is(err, errBoom) {
		t.Errorf("a store that fails: %v", err)
	}
}

// A removed term, of any origin, is no longer on the list (so not counted, not looked for, and not
// a candidate again), is listed as removed, and can be put back.
func TestRemoveATermAndPutItBack(t *testing.T) {
	h := &hand{}
	obs := []domain.Observation{{ScanID: "s1", CompanyID: "x", Phrase: "spiral belt", Language: "en", Score: 5, Pages: 1, StartedAt: now}}
	svc := handService(t, h, obs)
	key := domain.TermKey{Text: "SPIRAL belt", Language: domain.LanguageEn}
	if err := svc.RemoveTerm(t.Context(), "joris", key); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(texts(svc.Terms()), []string{"Drahtösengliederband"}) {
		t.Errorf("terms after removing = %v", texts(svc.Terms()))
	}
	removed := svc.RemovedTerms()
	if len(removed) != 1 || removed[0].Term.Text != "spiral belt" || removed[0].By != "joris" || removed[0].Term.Origin != domain.OriginResearch {
		t.Fatalf("removed = %+v", removed)
	}
	if list, _, err := svc.Candidates(t.Context(), domain.StatusPending); err != nil || len(list) != 0 {
		t.Errorf("a removed term came back as a candidate: %+v, %v", list, err)
	}
	if err := svc.RemoveTerm(t.Context(), "joris", key); !errors.Is(err, domain.ErrTermNotFound) {
		t.Errorf("removing it twice: %v", err)
	}
	if _, err := svc.AddTerm(t.Context(), "joris", domain.Acceptance{Text: "spiral belt", Language: "en", Family: "other", Tier: "broad"}); !errors.Is(err, domain.ErrTermExists) {
		t.Errorf("adding a removed term again, rather than putting it back: %v", err)
	}
	if err := svc.PutBack(t.Context(), domain.TermKey{Text: "spiral belt", Language: domain.LanguageEn}); err != nil {
		t.Fatal(err)
	}
	if len(svc.Terms()) != 2 || len(svc.RemovedTerms()) != 0 {
		t.Errorf("after putting back: %v, removed %+v", texts(svc.Terms()), svc.RemovedTerms())
	}
	if err := svc.PutBack(t.Context(), domain.TermKey{Text: "spiral belt", Language: domain.LanguageEn}); !errors.Is(err, domain.ErrNotRemoved) {
		t.Errorf("putting back a term on the list: %v", err)
	}
}

func TestHandTermsReportTheStoresFailures(t *testing.T) {
	key := domain.TermKey{Text: "spiral belt", Language: domain.LanguageEn}
	for _, op := range []string{"Remove", "PutBack"} {
		h := &hand{removed: []domain.Removal{{Key: key}}}
		svc := handService(t, h, nil)
		h.removed = nil
		h.failOn = map[string]bool{op: true}
		var err error
		if op == "Remove" {
			err = svc.RemoveTerm(t.Context(), "joris", domain.TermKey{Text: "Drahtösengliederband", Language: domain.LanguageDe})
		} else {
			err = svc.PutBack(t.Context(), key)
		}
		if !errors.Is(err, errBoom) {
			t.Errorf("%s failing: %v", op, err)
		}
	}
	for _, op := range []string{"Added", "Removed"} {
		h := &hand{failOn: map[string]bool{op: true}}
		svc := app.New(catalogue{}, &reviews{}, evidence(nil, nil), sightings(nil, nil), func() time.Time { return now }).WithHandTerms(h)
		if err := svc.Load(t.Context()); !errors.Is(err, errBoom) {
			t.Errorf("Load with %s failing: %v", op, err)
		}
	}
}

// Without a store for hand-added terms, the list has none, and adding or removing says so.
func TestWithoutHandTerms(t *testing.T) {
	svc := app.New(catalogue{terms: many(1)}, &reviews{}, evidence(nil, nil), sightings(nil, nil), func() time.Time { return now })
	if err := svc.Load(t.Context()); err != nil {
		t.Fatal(err)
	}
	if len(svc.HandTerms()) != 0 || len(svc.RemovedTerms()) != 0 {
		t.Error("no store, yet hand terms")
	}
	ctx := t.Context()
	key := domain.TermKey{Text: "term 000", Language: domain.LanguageEn}
	if _, err := svc.AddTerm(ctx, "j", domain.Acceptance{Text: "x", Language: "en", Family: "other", Tier: "broad"}); !errors.Is(err, app.ErrNoHandTerms) {
		t.Errorf("AddTerm = %v", err)
	}
	if err := svc.RemoveTerm(ctx, "j", key); !errors.Is(err, app.ErrNoHandTerms) {
		t.Errorf("RemoveTerm = %v", err)
	}
	if err := svc.PutBack(ctx, key); !errors.Is(err, app.ErrNoHandTerms) {
		t.Errorf("PutBack = %v", err)
	}
}

// A removal of a term the map no longer has (the research dropped it) is still listed, by its text.
func TestARemovalOfATermTheMapNoLongerHas(t *testing.T) {
	key := domain.TermKey{Text: "gone belt", Language: domain.LanguageEn}
	svc := handService(t, &hand{removed: []domain.Removal{{Key: key, By: "joris", At: now}}}, nil)
	if got := svc.RemovedTerms(); len(got) != 1 || got[0].Term.Text != "gone belt" || got[0].Term.Language != domain.LanguageEn {
		t.Errorf("removed = %+v", got)
	}
}
