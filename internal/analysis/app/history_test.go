package app_test

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/JorisJonkers-dev/findability-explorer/internal/analysis/app"
	"github.com/JorisJonkers-dev/findability-explorer/internal/analysis/domain"
)

func website(raw string) *url.URL {
	u, _ := url.Parse(raw)
	return u
}

// companies: x and y were scanned before, z was not, r is researched and was not, a is archived,
// n has no website, p a website that cannot be scanned.
func companies() []domain.Company {
	return []domain.Company{
		{ID: "x", Name: "X", Website: website("https://x.test/")},
		{ID: "y", Name: "Y", Website: website("https://y.test/")},
		{ID: "z", Name: "Z", Website: website("https://z.test/")},
		{ID: "r", Name: "R", Website: website("https://r.test/"), Researched: true},
		{ID: "a", Name: "A", Website: website("https://a.test/"), Archived: true},
		{ID: "n", Name: "N"},
		{ID: "p", Name: "P", Website: website("https://10.0.0.1/")},
	}
}

func scanned(r *repo, ids ...string) {
	for _, id := range ids {
		a := uuid.New()
		r.analyses[a] = &domain.Analysis{ID: a, CompetitorID: id, Status: domain.StatusDone, Number: 1}
		r.order = append(r.order, a)
	}
}

func rescanService(r *repo, d *directory, l app.Limits) *app.Service {
	return app.New(r, d, crawler{pages: pages()}, source{}, glossary, func() time.Time { return now }, l)
}

func TestRescanAllQueuesEveryCompanyWithData(t *testing.T) {
	r := newRepo()
	scanned(r, "x", "y", "a", "n")
	svc := rescanService(r, &directory{companies: companies()}, app.Limits{})
	queued, skipped, err := svc.Rescan(t.Context(), "admin", nil, true, 0)
	if err != nil || len(queued) != 3 || len(skipped) != 0 {
		t.Fatalf("Rescan = %+v, %+v, %v", queued, skipped, err)
	}
	// The desk research is data too: r is queued although no scan of it ran yet.
	want := map[string]int{"x": 2, "y": 2, "r": 1}
	for _, a := range queued {
		if !a.Rescan || a.MaxPages != domain.DefaultMaxPages || a.Number != want[a.CompetitorID] || a.CreatedBy != "admin" {
			t.Errorf("queued %+v", a)
		}
	}
}

func TestRescanNamedCompanies(t *testing.T) {
	r := running()
	svc := rescanService(r, &directory{companies: companies()}, app.Limits{})
	queued, skipped, err := svc.Rescan(t.Context(), "admin", []string{"z", "x", "n", "p", "unknown"}, false, 20)
	if err != nil || len(queued) != 1 || queued[0].CompetitorID != "z" || queued[0].MaxPages != 20 {
		t.Fatalf("Rescan = %+v, %v", queued, err)
	}
	want := map[string]string{
		"x":       "A scan of it is still queued or running.",
		"n":       "It has no website to scan.",
		"p":       "Its website cannot be scanned.",
		"unknown": "No company has this id.",
	}
	if len(skipped) != len(want) {
		t.Fatalf("skipped %+v", skipped)
	}
	for _, s := range skipped {
		if want[s.CompetitorID] != s.Reason {
			t.Errorf("skipped %s: %q", s.CompetitorID, s.Reason)
		}
	}
}

// A company whose site turned the last scan away is left out of re-scanning all: it would only
// be turned away again. Naming it still scans it.
func TestRescanAllLeavesOutABlockedSite(t *testing.T) {
	r := newRepo()
	scanned(r, "x", "y")
	for _, a := range r.analyses {
		if a.CompetitorID == "y" {
			a.Status, a.Failure = domain.StatusFailed, domain.BlockedReason("https://y.test/ answered 403 Forbidden")
		}
	}
	svc := rescanService(r, &directory{companies: companies()}, app.Limits{})
	queued, _, err := svc.Rescan(t.Context(), "admin", nil, true, 0)
	if err != nil || slices.ContainsFunc(queued, func(a domain.Analysis) bool { return a.CompetitorID == "y" }) {
		t.Fatalf("Rescan all = %+v, %v", queued, err)
	}
	named, _, err := svc.Rescan(t.Context(), "admin", []string{"y"}, false, 0)
	if err != nil || len(named) != 1 {
		t.Fatalf("Rescan y = %+v, %v", named, err)
	}
}

// A re-scan takes every company, however many: nothing is skipped for load.
func TestRescanTakesEveryCompanyHoweverMany(t *testing.T) {
	var many []domain.Company
	for i := range 300 {
		id := fmt.Sprintf("c%d", i)
		many = append(many, domain.Company{ID: id, Name: id, Website: website("https://" + id + ".test/"), Researched: true})
	}
	queued, skipped, err := rescanService(newRepo(), &directory{companies: many}, app.Limits{}).Rescan(t.Context(), "admin", nil, true, 1)
	if err != nil || len(queued) != 300 || len(skipped) != 0 {
		t.Fatalf("Rescan = %d queued, %+v, %v", len(queued), skipped, err)
	}
}

func TestRescanRefuses(t *testing.T) {
	cases := map[string]struct {
		repo     *repo
		dir      *directory
		ids      []string
		all      bool
		maxPages int
		want     error
	}{
		"neither":            {newRepo(), &directory{}, nil, false, 0, app.ErrInvalidRescan},
		"both":               {newRepo(), &directory{}, []string{"x"}, true, 0, app.ErrInvalidRescan},
		"too many pages":     {newRepo(), &directory{}, nil, true, 101, domain.ErrInvalidMaxPages},
		"no companies":       {newRepo(), &directory{companiesErr: errBoom}, nil, true, 0, errBoom},
		"no statistics":      {newRepo("CompetitorStats"), &directory{companies: companies()}, nil, true, 0, errBoom},
		"a create failure":   {newRepo("Create"), &directory{companies: companies()}, []string{"x"}, false, 0, errBoom},
		"a counting failure": {newRepo("CountActive"), &directory{companies: companies()}, []string{"x"}, false, 0, errBoom},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if _, _, err := rescanService(c.repo, c.dir, app.Limits{}).Rescan(t.Context(), "u", c.ids, c.all, c.maxPages); !errors.Is(err, c.want) {
				t.Fatalf("Rescan error = %v, want %v", err, c.want)
			}
		})
	}
}

// history stores three scans of x: 1 done, 2 failed, 3 done, with keywords.
func history(r *repo) (first, third uuid.UUID) {
	add := func(n int, status domain.Status, kws ...domain.Keyword) uuid.UUID {
		id := uuid.New()
		r.analyses[id] = &domain.Analysis{ID: id, CompetitorID: "x", Number: n, Status: status, Keywords: kws}
		r.order = append(r.order, id)
		return id
	}
	first = add(1, domain.StatusDone, domain.Keyword{
		Phrase: "eyelink belt", Language: "en", Score: 10, Pages: 1,
		Classification: domain.Classification{Tier: "product", Family: "eyelink"},
	})
	add(2, domain.StatusFailed)
	third = add(3, domain.StatusDone,
		domain.Keyword{Phrase: "eyelink belt", Language: "en", Score: 30, Pages: 4, Classification: domain.Classification{Tier: "product", Family: "eyelink"}},
		domain.Keyword{Phrase: "spiral freezer", Language: "en", Score: 12, Pages: 2, Classification: domain.Classification{Tier: "application", Family: "tri_flex_turn_curve"}},
	)
	r.first = map[domain.PhraseKey]int{{Phrase: "eyelink belt", Language: "en"}: 1}
	return first, third
}

func TestProfileIsTheLatestFinishedScan(t *testing.T) {
	r := newRepo()
	history(r)
	p, err := rescanService(r, &directory{companies: companies()}, limits()).Profile(t.Context(), "x")
	if err != nil {
		t.Fatal(err)
	}
	if p.Scan == nil || p.Scan.Number != 3 || p.FinishedScans != 2 || len(p.Groups) != len(domain.Groups()) {
		t.Fatalf("profile %+v", p)
	}
	if eyelink := p.Groups[0].Terms; len(eyelink) != 1 || eyelink[0].FirstScan != 1 {
		t.Fatalf("eyelink terms %+v", eyelink)
	}
	if triflex := p.Groups[3].Terms; len(triflex) != 1 || triflex[0].FirstScan != 3 {
		t.Fatalf("tri-flex terms %+v", triflex)
	}
	empty, err := rescanService(newRepo(), &directory{companies: companies()}, limits()).Profile(t.Context(), "y")
	if err != nil || empty.Scan != nil || len(empty.Groups) != 0 || empty.Languages == nil {
		t.Fatalf("profile without scans = %+v, %v", empty, err)
	}
}

func TestProfileAndCompareFailures(t *testing.T) {
	cases := map[string]struct {
		failOn string
		dir    *directory
		id     string
		want   error
	}{
		"unknown company":  {"", &directory{companies: companies()}, "nobody", app.ErrUnknownCompany},
		"no companies":     {"", &directory{companiesErr: errBoom}, "x", errBoom},
		"no scans":         {"ListByCompetitor", &directory{companies: companies()}, "x", errBoom},
		"no scan":          {"Get", &directory{companies: companies()}, "x", errBoom},
		"no first scans":   {"FirstScans", &directory{companies: companies()}, "x", errBoom},
		"compare unknown":  {"", &directory{companies: companies()}, "nobody", app.ErrUnknownCompany},
		"compare no scans": {"ListByCompetitor", &directory{companies: companies()}, "x", errBoom},
		"compare no scan":  {"Get", &directory{companies: companies()}, "x", errBoom},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			r := newRepo()
			history(r)
			r.failOn[c.failOn] = true
			svc := rescanService(r, c.dir, limits())
			var err error
			if strings.HasPrefix(name, "compare") {
				_, _, _, err = svc.Compare(t.Context(), c.id, uuid.Nil, uuid.Nil)
			} else {
				_, err = svc.Profile(t.Context(), c.id)
			}
			if !errors.Is(err, c.want) {
				t.Fatalf("error = %v, want %v", err, c.want)
			}
		})
	}
}

func TestCompareTheLatestTwoFinishedScans(t *testing.T) {
	r := newRepo()
	first, third := history(r)
	svc := rescanService(r, &directory{companies: companies()}, limits())
	from, to, groups, err := svc.Compare(t.Context(), "x", uuid.Nil, uuid.Nil)
	if err != nil || from.ID != first || to.ID != third || len(groups) != len(domain.Groups()) {
		t.Fatalf("Compare = %v %v %d, %v", from.ID, to.ID, len(groups), err)
	}
	if len(groups[0].Moved) != 1 || len(groups[3].Added) != 1 {
		t.Fatalf("groups %+v", groups)
	}
	if _, _, _, err := svc.Compare(t.Context(), "x", third, third); !errors.Is(err, domain.ErrNotComparable) {
		t.Fatalf("Compare of one scan = %v", err)
	}
}

func TestCompareReportsTheSecondLoadFailing(t *testing.T) {
	r := newRepo()
	history(r)
	g := &failingSecondGet{repo: r}
	svc := app.New(g, &directory{companies: companies()}, crawler{}, source{}, glossary, func() time.Time { return now }, limits())
	if _, _, _, err := svc.Compare(t.Context(), "x", uuid.Nil, uuid.Nil); !errors.Is(err, errBoom) {
		t.Fatalf("Compare = %v", err)
	}
}

// failingSecondGet fails every Get after the first.
type failingSecondGet struct {
	*repo
	gets int
}

func (f *failingSecondGet) Get(ctx context.Context, id uuid.UUID) (domain.Analysis, error) {
	f.gets++
	if f.gets > 1 {
		return domain.Analysis{}, errBoom
	}
	return f.repo.Get(ctx, id)
}

func TestWorkRunsScansSideBySide(t *testing.T) {
	r := newRepo()
	block := make(chan struct{})
	var inFlight, most atomic.Int32
	c := blockingCrawler{block: block, inFlight: &inFlight, most: &most}
	svc := app.New(r, &directory{}, c, source{}, glossary, func() time.Time { return now },
		app.Limits{Workers: 2})
	for _, host := range []string{"https://x.test/", "https://y.test/"} {
		if _, err := svc.Start(t.Context(), "u", host, 1, ""); err != nil {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() { svc.Work(ctx, func(uuid.UUID, error) {}); close(done) }()
	deadline := time.After(5 * time.Second)
	for most.Load() < 2 {
		select {
		case <-deadline:
			t.Fatal("the two scans never ran at once")
		case <-time.After(5 * time.Millisecond):
		}
	}
	close(block)
	cancel()
	<-done
}

type blockingCrawler struct {
	block          chan struct{}
	inFlight, most *atomic.Int32
}

func (b blockingCrawler) Crawl(ctx context.Context, _ domain.Target, _ int, _ domain.LinkScorer, _ func(domain.Page) error) error {
	n := b.inFlight.Add(1)
	for {
		m := b.most.Load()
		if n <= m || b.most.CompareAndSwap(m, n) {
			break
		}
	}
	select {
	case <-b.block:
	case <-ctx.Done():
	}
	b.inFlight.Add(-1)
	return nil
}
