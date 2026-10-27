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
	overrides    []domain.Override
	overridesErr error
	changeErr    error
	deleted      []string
}

func (r *repo) Update(_ context.Context, c domain.Company) error {
	if r.changeErr != nil {
		return r.changeErr
	}
	for i := range r.added {
		if r.added[i].ID == c.ID {
			r.added[i] = c
		}
	}
	return nil
}

func (r *repo) Delete(_ context.Context, id string) error {
	r.deleted = append(r.deleted, id)
	return r.changeErr
}

func (r *repo) Overrides(context.Context) ([]domain.Override, error) {
	return r.overrides, r.overridesErr
}

func (r *repo) SaveOverride(_ context.Context, o domain.Override) error {
	if r.changeErr != nil {
		return r.changeErr
	}
	r.overrides = []domain.Override{o}
	return nil
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
	statsErr, byErr      error
	stats                map[string]domain.Stats
	list                 []domain.Scan
	active               int
	activeErr, deleteErr error
	deleted              *[]string
}

func (s scans) Active(context.Context, string) (int, error) { return s.active, s.activeErr }

func (s scans) DeleteByCompany(_ context.Context, id string) error {
	if s.deleted != nil {
		*s.deleted = append(*s.deleted, id)
	}
	return s.deleteErr
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

func ptr[T any](v T) *T { return &v }

func added() *repo {
	u, _ := url.Parse("https://example.com/")
	return &repo{added: []domain.Company{{ID: "example", Name: "Example", Host: "example.com", URL: u, Source: domain.SourceAdded}}}
}

func TestUpdateAnAddedCompany(t *testing.T) {
	r := added()
	l, err := service(r, scans{}).Update(t.Context(), "admin", "example", domain.Change{Name: ptr(" Example Belts "), URL: ptr("https://www.example-belts.com")})
	if err != nil || l.Name != "Example Belts" || l.Host != "example-belts.com" || r.added[0].Host != "example-belts.com" {
		t.Fatalf("Update = %+v, %v", l, err)
	}
	// An added company is archived in its row, and keeps its scans; restoring it keeps it.
	r = added()
	l, err = service(r, scans{}).Update(t.Context(), "admin", "example", domain.Change{Archived: ptr(true)})
	if err != nil || !l.Archived || !r.added[0].Archived || r.added[0].ArchivedBy != "admin" || len(r.deleted) != 0 {
		t.Fatalf("archiving an added company = %+v, %v, %+v", l, err, r)
	}
	l, err = service(r, scans{}).Update(t.Context(), "joris", "example", domain.Change{Archived: ptr(false)})
	if err != nil || l.Archived || r.added[0].KeptBy != "joris" {
		t.Fatalf("restoring an added company = %+v, %v", l, err)
	}
}

func TestArchiveUnlessKept(t *testing.T) {
	r := added()
	svc := service(r, scans{})
	if ok, err := svc.ArchiveUnlessKept(t.Context(), "language model", "example"); err != nil || !ok || r.added[0].ArchivedBy != "language model" {
		t.Fatalf("ArchiveUnlessKept(added) = %v, %v, %+v", ok, err, r.added)
	}
	if ok, err := svc.ArchiveUnlessKept(t.Context(), "language model", "example"); err != nil || ok {
		t.Fatalf("ArchiveUnlessKept(archived) = %v, %v", ok, err)
	}
	if ok, err := svc.ArchiveUnlessKept(t.Context(), "language model", "tribelt"); err != nil || ok {
		t.Fatalf("ArchiveUnlessKept(Tribelt) = %v, %v", ok, err)
	}
	research := &repo{}
	if ok, err := service(research, scans{}).ArchiveUnlessKept(t.Context(), "language model", "twentebelt"); err != nil || !ok ||
		research.overrides[0].ArchivedBy != "language model" {
		t.Fatalf("ArchiveUnlessKept(researched) = %v, %v, %+v", ok, err, research.overrides)
	}
	kept := &repo{overrides: []domain.Override{{ID: "twentebelt", KeptBy: "joris", KeptAt: now}}}
	if ok, err := service(kept, scans{}).ArchiveUnlessKept(t.Context(), "language model", "twentebelt"); err != nil || ok {
		t.Fatalf("ArchiveUnlessKept(kept) = %v, %v", ok, err)
	}
	cases := map[string]struct {
		repo *repo
		id   string
		want error
	}{
		"an unknown company": {added(), "nobody", domain.ErrNotFound},
		"a failing list":     {&repo{listErrAfter: 1}, "example", errBoom},
		"a failing update":   {&repo{added: added().added, changeErr: errBoom}, "example", errBoom},
	}
	for name, c := range cases {
		if _, err := service(c.repo, scans{}).ArchiveUnlessKept(t.Context(), "u", c.id); !errors.Is(err, c.want) {
			t.Errorf("%s: ArchiveUnlessKept = %v, want %v", name, err, c.want)
		}
	}
}

func TestUpdateAResearchedCompanyStoresAnOverride(t *testing.T) {
	r := &repo{}
	svc := service(r, scans{})
	l, err := svc.Update(t.Context(), "admin", "twentebelt", domain.Change{URL: ptr("https://www.twentebelt.nl"), Archived: ptr(true)})
	if err != nil || !l.Archived || l.ArchivedBy != "admin" || l.Host != "twentebelt.nl" {
		t.Fatalf("Update = %+v, %v", l, err)
	}
	if len(r.overrides) != 1 || r.overrides[0].Host != "twentebelt.nl" || r.overrides[0].ArchivedAt.IsZero() {
		t.Fatalf("overrides %+v", r.overrides)
	}
	got, _, err := svc.Get(t.Context(), "twentebelt")
	if err != nil || !got.Archived || got.Host != "twentebelt.nl" {
		t.Fatalf("Get after the change = %+v, %v", got, err)
	}
	// Restoring keeps the new website.
	l, err = svc.Update(t.Context(), "admin", "twentebelt", domain.Change{Archived: ptr(false)})
	if err != nil || l.Archived || l.Host != "twentebelt.nl" {
		t.Fatalf("restore = %+v, %v", l, err)
	}
}

func TestUpdateRefuses(t *testing.T) {
	cases := map[string]struct {
		repo  *repo
		scans scans
		id    string
		ch    domain.Change
		want  error
	}{
		"an unknown company": {added(), scans{}, "nobody", domain.Change{}, domain.ErrNotFound},
		"a bad name":         {added(), scans{}, "example", domain.Change{Name: ptr(" ")}, domain.ErrInvalidName},
		"a bad website":      {added(), scans{}, "example", domain.Change{URL: ptr("https://localhost/")}, domain.ErrInvalidWebsite},
		"a failing list":     {&repo{listErrAfter: 1}, scans{}, "example", domain.Change{}, errBoom},
		"failing overrides":  {&repo{overridesErr: errBoom}, scans{}, "example", domain.Change{}, errBoom},
		"a failing update":   {&repo{added: added().added, changeErr: errBoom}, scans{}, "example", domain.Change{}, errBoom},
		"a failing override": {&repo{changeErr: errBoom}, scans{}, "twentebelt", domain.Change{}, errBoom},
		"failing statistics": {added(), scans{statsErr: errBoom}, "example", domain.Change{}, errBoom},
	}
	for name, c := range cases {
		if _, err := service(c.repo, c.scans).Update(t.Context(), "u", c.id, c.ch); !errors.Is(err, c.want) {
			t.Errorf("%s: Update = %v, want %v", name, err, c.want)
		}
	}
	var known domain.AlreadyKnownError
	if _, err := service(added(), scans{}).Update(t.Context(), "u", "example", domain.Change{URL: ptr("https://twentebelt.com")}); !errors.As(err, &known) {
		t.Errorf("a taken host = %v", err)
	}
}

func TestDelete(t *testing.T) {
	r := added()
	var deletedScans []string
	if err := service(r, scans{deleted: &deletedScans}).Delete(t.Context(), "admin", "example"); err != nil ||
		len(r.deleted) != 1 || len(deletedScans) != 1 {
		t.Fatalf("Delete(added) = %v, %v, %v", err, r.deleted, deletedScans)
	}
	research := &repo{}
	if err := service(research, scans{}).Delete(t.Context(), "admin", "twentebelt"); err != nil || len(research.overrides) != 1 ||
		research.overrides[0].ArchivedBy != "admin" {
		t.Fatalf("Delete(researched) = %v, %+v", err, research.overrides)
	}
	cases := map[string]struct {
		repo  *repo
		scans scans
		id    string
		want  error
	}{
		"an unknown company":    {added(), scans{}, "nobody", domain.ErrNotFound},
		"a failing list":        {&repo{listErrAfter: 1}, scans{}, "example", errBoom},
		"a scan running":        {added(), scans{active: 1}, "example", app.ErrScanActive},
		"a failing count":       {added(), scans{activeErr: errBoom}, "example", errBoom},
		"failing scan deletion": {added(), scans{deleteErr: errBoom}, "example", errBoom},
		"failing deletion":      {&repo{added: added().added, changeErr: errBoom}, scans{}, "example", errBoom},
		"a failing archive":     {&repo{changeErr: errBoom}, scans{}, "twentebelt", errBoom},
	}
	for name, c := range cases {
		if err := service(c.repo, c.scans).Delete(t.Context(), "u", c.id); !errors.Is(err, c.want) {
			t.Errorf("%s: Delete = %v, want %v", name, err, c.want)
		}
	}
}
