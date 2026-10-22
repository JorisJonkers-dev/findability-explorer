package app_test

import (
	"context"
	"errors"
	"net/url"
	"testing"
	"time"

	"github.com/JorisJonkers-dev/findability-explorer/internal/companies/app"
	"github.com/JorisJonkers-dev/findability-explorer/internal/companies/domain"
)

var errBoom = errors.New("boom")

var now = time.Date(2026, 10, 22, 9, 30, 0, 0, time.UTC)

// repo is an in-memory Repository. listErrAfter makes List fail from that call on (0: never);
// sneak adds a company between the first and second List, as a concurrent scan would.
type repo struct {
	added        []domain.Company
	addErr       error
	listErrAfter int
	lists        int
	sneak        *domain.Company
}

func (r *repo) List(context.Context) ([]domain.Company, error) {
	r.lists++
	if r.listErrAfter > 0 && r.lists >= r.listErrAfter {
		return nil, errBoom
	}
	if r.sneak != nil && r.lists == 2 {
		r.added = append(r.added, *r.sneak)
	}
	return r.added, nil
}

func (r *repo) Add(_ context.Context, c domain.Company) error {
	if r.addErr != nil {
		return r.addErr
	}
	r.added = append(r.added, c)
	return nil
}

type scans struct {
	statsErr, byErr error
	stats           map[string]domain.Stats
	list            []domain.Scan
}

func (s scans) Stats(context.Context) (map[string]domain.Stats, error) { return s.stats, s.statsErr }

func (s scans) ByCompany(_ context.Context, _ string, limit int) ([]domain.Scan, error) {
	if limit != app.MaxScans {
		return nil, errors.New("unexpected limit")
	}
	return s.list, s.byErr
}

func research() []domain.Company {
	return []domain.Company{
		{ID: "twentebelt", Name: "Twentebelt", Host: "twentebelt.com", Source: domain.SourceResearch},
		{ID: "tribelt", Name: "Tribelt B.V.", Host: "tribelt.nl", GroupCompany: true, Source: domain.SourceResearch},
	}
}

func service(r *repo, s scans) *app.Service {
	return app.New(research, r, s, func() time.Time { return now })
}

func TestListAndGet(t *testing.T) {
	last := domain.Scan{URL: "https://www.twentebelt.com/"}
	s := scans{stats: map[string]domain.Stats{"twentebelt": {Count: 3, Last: &last}}, list: []domain.Scan{last}}
	svc := service(&repo{added: []domain.Company{{ID: "example", Name: "Example", Host: "example.com", Source: domain.SourceAdded}}}, s)
	listed, err := svc.List(t.Context())
	if err != nil || len(listed) != 3 || listed[0].ID != "twentebelt" || listed[0].Stats.Count != 3 ||
		listed[1].ID != "example" || listed[2].ID != "tribelt" {
		t.Fatalf("List = %+v, %v", listed, err)
	}
	got, list, err := svc.Get(t.Context(), "twentebelt")
	if err != nil || got.Name != "Twentebelt" || got.Stats.Count != 3 || len(list) != 1 {
		t.Fatalf("Get = %+v, %v, %v", got, list, err)
	}
	if _, _, err := svc.Get(t.Context(), "nobody"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("Get(nobody) = %v", err)
	}
}

func TestListAndGetFailures(t *testing.T) {
	if _, err := service(&repo{listErrAfter: 1}, scans{}).List(t.Context()); !errors.Is(err, errBoom) {
		t.Errorf("List with a failing repo = %v", err)
	}
	if _, err := service(&repo{}, scans{statsErr: errBoom}).List(t.Context()); !errors.Is(err, errBoom) {
		t.Errorf("List with failing stats = %v", err)
	}
	if _, _, err := service(&repo{listErrAfter: 1}, scans{}).Get(t.Context(), "twentebelt"); !errors.Is(err, errBoom) {
		t.Errorf("Get with a failing repo = %v", err)
	}
	if _, _, err := service(&repo{}, scans{statsErr: errBoom}).Get(t.Context(), "twentebelt"); !errors.Is(err, errBoom) {
		t.Errorf("Get with failing stats = %v", err)
	}
	if _, _, err := service(&repo{}, scans{byErr: errBoom}).Get(t.Context(), "twentebelt"); !errors.Is(err, errBoom) {
		t.Errorf("Get with failing scans = %v", err)
	}
}

func TestAdd(t *testing.T) {
	r := &repo{}
	c, err := service(r, scans{}).Add(t.Context(), "joris", " Example Belts ", "https://www.example-belts.com")
	if err != nil || c.ID != "example-belts" || c.Name != "Example Belts" || c.Host != "example-belts.com" ||
		!c.AddedAt.Equal(now) || len(r.added) != 1 {
		t.Fatalf("Add = %+v, %v", c, err)
	}
	cases := map[string]struct {
		repo      *repo
		name, url string
		want      error
	}{
		"a bad name":      {&repo{}, " ", "https://x.example", domain.ErrInvalidName},
		"a bad website":   {&repo{}, "X", "https://localhost/", domain.ErrInvalidWebsite},
		"a failing list":  {&repo{listErrAfter: 1}, "X", "https://x.example", errBoom},
		"a failing store": {&repo{addErr: errBoom}, "X", "https://x.example", errBoom},
	}
	for name, c := range cases {
		if _, err := service(c.repo, scans{}).Add(t.Context(), "u", c.name, c.url); !errors.Is(err, c.want) {
			t.Errorf("%s: Add = %v, want %v", name, err, c.want)
		}
	}
	var known domain.AlreadyKnownError
	if _, err := service(&repo{}, scans{}).Add(t.Context(), "u", "Copy", "https://twentebelt.com"); !errors.As(err, &known) {
		t.Errorf("Add on a known host = %v", err)
	}
}

func TestResolve(t *testing.T) {
	r := &repo{}
	svc := service(r, scans{})
	site := func(raw string) *url.URL { u, _ := url.Parse(raw); return u }
	if id, err := svc.Resolve(t.Context(), "u", site("https://www.twentebelt.com/conveyor-belts/"), "ignored"); err != nil || id != "twentebelt" {
		t.Fatalf("Resolve(known) = %q, %v", id, err)
	}
	id, err := svc.Resolve(t.Context(), "u", site("https://new-belts.example/products/x"), "")
	if err != nil || id != "new-belts-example" || r.added[0].Website != "https://new-belts.example/" || r.added[0].Name != "new-belts.example" {
		t.Fatalf("Resolve(unknown, no name) = %q, %v, %+v", id, err, r.added)
	}
	if id, err := svc.Resolve(t.Context(), "u", site("https://other.example/"), "Other Belts"); err != nil || id != "other-belts" {
		t.Fatalf("Resolve(unknown, named) = %q, %v", id, err)
	}
	if _, err := svc.Resolve(t.Context(), "u", site("https://long.example/"), string(make([]byte, 300))); !errors.Is(err, domain.ErrInvalidName) {
		t.Fatalf("Resolve with a too-long name = %v", err)
	}
	if _, err := service(&repo{listErrAfter: 1}, scans{}).Resolve(t.Context(), "u", site("https://x.example/"), ""); !errors.Is(err, errBoom) {
		t.Fatalf("Resolve with a failing repo = %v", err)
	}
	if _, err := service(&repo{addErr: errBoom}, scans{}).Resolve(t.Context(), "u", site("https://x.example/"), ""); !errors.Is(err, errBoom) {
		t.Fatalf("Resolve with a failing store = %v", err)
	}
	// A scan of the same new site started a moment earlier added it in between.
	racing := &repo{sneak: &domain.Company{ID: "race", Name: "Race", Host: "race.example", Source: domain.SourceAdded}}
	if id, err := service(racing, scans{}).Resolve(t.Context(), "u", site("https://race.example/"), "Race Belts"); err != nil || id != "race" {
		t.Fatalf("Resolve during a race = %q, %v", id, err)
	}
}
