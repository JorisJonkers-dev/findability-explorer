package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/JorisJonkers-dev/findability-explorer/internal/companies/app"
)

// candidates records the candidates added, or fails.
type candidates struct {
	recorded map[string]string
	err      error
}

func (c *candidates) RecordAdded(_ context.Context, _, candidateID, companyID string) error {
	if c.err != nil {
		return c.err
	}
	c.recorded[candidateID] = companyID
	return nil
}

// scanner counts the scans it starts.
type scanner struct {
	ids []string
	err error
}

func (s *scanner) Scan(_ context.Context, _ string, ids []string) (int, error) {
	s.ids = ids
	return len(ids), s.err
}

func batchService(r *repo) (*app.Service, *candidates, *scanner) {
	svc := service(r, scans{})
	c, s := &candidates{recorded: map[string]string{}}, &scanner{}
	svc.WithBatch(c, s)
	return svc, c, s
}

func TestAddManyAddsSkipsAndRejects(t *testing.T) {
	r := &repo{}
	svc, c, s := batchService(r)
	got, err := svc.AddMany(t.Context(), "joris", []app.NewItem{
		{Name: "Intralox", Website: "https://www.intralox.com", CandidateID: "3a7bd3e2360a9f01"},
		{Name: "Habasit", Website: "https://habasit.com"},
		{Name: "Twente Belt", Website: "https://www.twentebelt.com/nl"},
		{Name: "Intralox LLC", Website: "https://intralox.de"},
		{Name: "Lumsden Belting"},
		{Name: "Bad", Website: "ftp://bad.example"},
		{Name: " ", Website: "https://x.example"},
		{Name: "Habasit"}, // a duplicate by name, without a website
	}, true)
	if err != nil {
		t.Fatal(err)
	}
	want := []struct {
		outcome app.Outcome
		id      string
	}{
		{app.OutcomeAdded, "intralox"},
		{app.OutcomeAdded, "habasit"},
		{app.OutcomeDuplicate, "twentebelt"},
		{app.OutcomeDuplicate, "intralox"},
		{app.OutcomeRejected, ""},
		{app.OutcomeRejected, ""},
		{app.OutcomeRejected, ""},
		{app.OutcomeDuplicate, "habasit"},
	}
	for i, w := range want {
		if g := got.Items[i]; g.Outcome != w.outcome || g.CompanyID != w.id {
			t.Errorf("item %d = %+v, want %s %s", i, g, w.outcome, w.id)
		}
	}
	if got.Items[2].Reason != "Twentebelt is a competitor already." || got.Items[4].Reason != "Add its website first." || got.Items[6].Name != " " {
		t.Errorf("reasons = %+v", got.Items)
	}
	if len(r.added) != 2 || c.recorded["3a7bd3e2360a9f01"] != "intralox" || got.Scans != 2 || len(s.ids) != 2 {
		t.Errorf("added %d, recorded %v, scans %d", len(r.added), c.recorded, got.Scans)
	}
}

func TestAddManyWithoutScanningAndItsLimits(t *testing.T) {
	svc, _, s := batchService(&repo{})
	if got, err := svc.AddMany(t.Context(), "joris", []app.NewItem{{Name: "Habasit", Website: "https://habasit.com"}}, false); err != nil || got.Scans != 0 || s.ids != nil {
		t.Errorf("no scan = %+v, %v", got, err)
	}
	if _, err := svc.AddMany(t.Context(), "joris", nil, false); !errors.Is(err, app.ErrBatchSize) {
		t.Errorf("an empty batch = %v", err)
	}
	if _, err := svc.AddMany(t.Context(), "joris", make([]app.NewItem, app.MaxBatch+1), false); !errors.Is(err, app.ErrBatchSize) {
		t.Errorf("too large a batch = %v", err)
	}
}

func TestAddManyFailures(t *testing.T) {
	item := []app.NewItem{{Name: "Intralox", Website: "https://intralox.com", CandidateID: "x"}}
	if _, err := service(&repo{}, scans{}).AddMany(t.Context(), "u", item, false); err == nil {
		t.Error("a batch without its ports added companies")
	}
	unwired, _, _ := batchService(&repo{listErrAfter: 1})
	if _, err := unwired.AddMany(t.Context(), "u", item, false); !errors.Is(err, errBoom) {
		t.Errorf("a failing list = %v", err)
	}
	svc, _, _ := batchService(&repo{addErr: errBoom})
	if _, err := svc.AddMany(t.Context(), "u", item, false); !errors.Is(err, errBoom) {
		t.Errorf("a failing store = %v", err)
	}
	svc, c, s := batchService(&repo{})
	c.err = app.ErrCandidateGone
	if got, err := svc.AddMany(t.Context(), "u", item, false); err != nil || got.Items[0].Outcome != app.OutcomeAdded {
		t.Errorf("a decided candidate still adds the company: %+v, %v", got, err)
	}
	c.err = errBoom
	if _, err := svc.AddMany(t.Context(), "u", []app.NewItem{{Name: "Habasit", Website: "https://habasit.com", CandidateID: "y"}}, false); !errors.Is(err, errBoom) {
		t.Errorf("a failing candidate = %v", err)
	}
	c.err, s.err = nil, errBoom
	if _, err := svc.AddMany(t.Context(), "u", []app.NewItem{{Name: "Fenner", Website: "https://fenner.com"}}, true); !errors.Is(err, errBoom) {
		t.Errorf("a failing scan = %v", err)
	}
}
