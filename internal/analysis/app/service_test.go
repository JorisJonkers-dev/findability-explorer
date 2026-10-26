package app_test

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/JorisJonkers-dev/findability-explorer/internal/analysis/app"
	"github.com/JorisJonkers-dev/findability-explorer/internal/analysis/domain"
)

var errBoom = errors.New("boom")

// repo is an in-memory domain.Repository; failOn names the methods that fail.
type repo struct {
	mu        sync.Mutex
	analyses  map[uuid.UUID]*domain.Analysis
	order     []uuid.UUID
	tokens    int64
	failOn    map[string]bool
	failedRun string
	first     map[domain.PhraseKey]int
}

func newRepo(failOn ...string) *repo {
	r := &repo{analyses: map[uuid.UUID]*domain.Analysis{}, failOn: map[string]bool{}}
	for _, f := range failOn {
		r.failOn[f] = true
	}
	return r
}

func (r *repo) err(op string) error {
	if r.failOn[op] {
		return errBoom
	}
	return nil
}

func (r *repo) Create(_ context.Context, a domain.Analysis) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.err("Create"); err != nil {
		return 0, err
	}
	for _, other := range r.analyses {
		if other.CompetitorID == a.CompetitorID {
			a.Number = max(a.Number, other.Number)
		}
	}
	a.Number++
	r.analyses[a.ID] = &a
	r.order = append(r.order, a.ID)
	return a.Number, nil
}

func (r *repo) CountActive(_ context.Context, competitorID string) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := 0
	for _, a := range r.analyses {
		if a.CompetitorID == competitorID && (a.Status == domain.StatusQueued || a.Status == domain.StatusRunning) {
			n++
		}
	}
	return n, r.err("CountActive")
}

func (r *repo) FirstScans(context.Context, string) (map[domain.PhraseKey]int, error) {
	return r.first, r.err("FirstScans")
}

func (r *repo) Get(_ context.Context, id uuid.UUID) (domain.Analysis, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.err("Get"); err != nil {
		return domain.Analysis{}, err
	}
	a, ok := r.analyses[id]
	if !ok {
		return domain.Analysis{}, domain.ErrNotFound
	}
	return *a, nil
}

func (r *repo) List(_ context.Context, limit int) ([]domain.Analysis, error) {
	if err := r.err("List"); err != nil {
		return nil, err
	}
	return make([]domain.Analysis, limit), nil
}

func (r *repo) ListByCompetitor(_ context.Context, competitorID string, limit int) ([]domain.Analysis, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []domain.Analysis{}
	for i := len(r.order) - 1; i >= 0 && len(out) < limit; i-- {
		if a := r.analyses[r.order[i]]; a.CompetitorID == competitorID {
			summary := *a
			summary.Keywords = nil
			out = append(out, summary)
		}
	}
	return out, r.err("ListByCompetitor")
}

func (r *repo) CompetitorStats(context.Context) (map[string]domain.CompetitorStats, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := map[string]domain.CompetitorStats{}
	for _, a := range r.analyses {
		s := out[a.CompetitorID]
		s.Count++
		out[a.CompetitorID] = s
	}
	return out, r.err("CompetitorStats")
}

func (r *repo) TokensSince(context.Context, time.Time) (int64, error) {
	return r.tokens, r.err("TokensSince")
}

func (r *repo) Queued(context.Context) ([]uuid.UUID, error) {
	if err := r.err("Queued"); err != nil {
		return nil, err
	}
	var ids []uuid.UUID
	for _, id := range r.order {
		if r.analyses[id].Status == domain.StatusQueued {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

func (r *repo) FailRunning(_ context.Context, reason string, _ time.Time) error {
	r.failedRun = reason
	return r.err("FailRunning")
}

func (r *repo) update(op string, id uuid.UUID, fn func(*domain.Analysis)) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.err(op); err != nil {
		return err
	}
	fn(r.analyses[id])
	return nil
}

func (r *repo) MarkRunning(_ context.Context, id uuid.UUID) error {
	return r.update("MarkRunning", id, func(a *domain.Analysis) { a.Status = domain.StatusRunning })
}

func (r *repo) AddPage(_ context.Context, id uuid.UUID, p domain.PageSummary) error {
	return r.update("AddPage", id, func(a *domain.Analysis) { a.Pages = append(a.Pages, p); a.PagesCrawled++ })
}

func (r *repo) SaveFindings(_ context.Context, id uuid.UUID, kws []domain.Keyword, known []domain.KnownTerm) error {
	return r.update("SaveFindings", id, func(a *domain.Analysis) { a.Keywords, a.KnownTerms = kws, known })
}

func (r *repo) SaveMatching(_ context.Context, id uuid.UUID, m domain.Matching, model string, eqs []domain.Equivalent, _ domain.Usage) error {
	return r.update("SaveMatching", id, func(a *domain.Analysis) { a.Matching, a.MatchingModel, a.Equivalents = m, model, eqs })
}

func (r *repo) Finish(_ context.Context, id uuid.UUID, at time.Time) error {
	return r.update("Finish", id, func(a *domain.Analysis) { a.Status, a.FinishedAt = domain.StatusDone, at })
}

func (r *repo) Fail(_ context.Context, id uuid.UUID, reason string, at time.Time) error {
	return r.update("Fail", id, func(a *domain.Analysis) {
		a.Status, a.Failure, a.FinishedAt = domain.StatusFailed, reason, at
	})
}

// crawler hands out pages, then err.
type crawler struct {
	pages []domain.Page
	err   error
}

func (c crawler) Crawl(_ context.Context, _ domain.Target, maxPages int, _ domain.LinkScorer, onPage func(domain.Page) error) error {
	for _, p := range c.pages[:min(maxPages, len(c.pages))] {
		if err := onPage(p); err != nil {
			return err
		}
	}
	return c.err
}

type matcher struct {
	eqs   []domain.Equivalent
	err   error
	calls int
}

func (m *matcher) Match(context.Context, string, bool, []domain.TribeltFamily) ([]domain.Equivalent, domain.Usage, error) {
	m.calls++
	return m.eqs, domain.Usage{InputTokens: 100, OutputTokens: 10}, m.err
}

// source hands out one matcher, or none, or fails.
type source struct {
	matcher domain.Matcher
	err     error
}

const model = "test: model-1"

func (s source) Current(context.Context) (domain.Matcher, string, error) {
	if s.matcher == nil {
		return nil, "", s.err
	}
	return s.matcher, model, s.err
}

var now = time.Date(2026, 10, 22, 9, 30, 0, 0, time.UTC)

func limits() app.Limits {
	return app.Limits{MonthlyTokens: 1000, SiteTextChars: 10_000}
}

func glossary() []domain.GlossaryTerm {
	return []domain.GlossaryTerm{{Text: "spiral belt", Language: "en", Family: "tri_flex_turn_curve", Tier: "product"}}
}

func pages() []domain.Page {
	return []domain.Page{{URL: "https://x.test/", Language: "en", Title: "Spiral belt", Text: "Our spiral belt for freezers."}}
}

// directory names every website's company after its host's first label, or fails with err.
type directory struct {
	err          error
	lastName     string
	companies    []domain.Company
	companiesErr error
}

func (d *directory) Companies(context.Context) ([]domain.Company, error) {
	return d.companies, d.companiesErr
}

func (d *directory) Resolve(_ context.Context, _ string, website *url.URL, name string) (string, error) {
	d.lastName = name
	if d.err != nil {
		return "", d.err
	}
	host, _, _ := strings.Cut(website.Hostname(), ".")
	return host, nil
}

func service(r *repo, c domain.Crawler, m domain.Matcher, l app.Limits) *app.Service {
	return app.New(r, &directory{}, c, source{matcher: m}, glossary, func() time.Time { return now }, l)
}

func TestStartQueuesAValidRequest(t *testing.T) {
	r := newRepo()
	a, err := service(r, crawler{}, nil, limits()).Start(t.Context(), "joris", "https://x.test", 0, "")
	if err != nil {
		t.Fatal(err)
	}
	if a.Status != domain.StatusQueued || a.MaxPages != domain.DefaultMaxPages || a.URL != "https://x.test/" ||
		a.CompetitorID != "x" ||
		a.CreatedBy != "joris" || !a.CreatedAt.Equal(now) || a.Matching != domain.MatchingPending {
		t.Fatalf("started %+v", a)
	}
	if _, ok := r.analyses[a.ID]; !ok {
		t.Fatal("not stored")
	}
}

func TestStartRefuses(t *testing.T) {
	cases := map[string]struct {
		repo     *repo
		url      string
		maxPages int
		want     error
	}{
		"a private host":          {newRepo(), "https://10.0.0.1/", 5, domain.ErrInvalidTarget},
		"too many pages":          {newRepo(), "https://x.test/", 101, domain.ErrInvalidMaxPages},
		"a scan running":          {running(), "https://x.test/", 5, app.ErrScanActive},
		"an active count failure": {newRepo("CountActive"), "https://x.test/", 5, errBoom},
		"a create failure":        {newRepo("Create"), "https://x.test/", 5, errBoom},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := service(c.repo, crawler{}, nil, limits()).Start(t.Context(), "u", c.url, c.maxPages, ""); !errors.Is(err, c.want) {
				t.Fatalf("Start error = %v, want %v", err, c.want)
			}
		})
	}
}

// running is a repo with a scan of x running.
func running() *repo {
	r := newRepo()
	id := uuid.New()
	r.analyses[id] = &domain.Analysis{ID: id, CompetitorID: "x", Status: domain.StatusRunning}
	r.order = append(r.order, id)
	return r
}

func TestStartNumbersEachCompanysScans(t *testing.T) {
	r := newRepo()
	svc := service(r, crawler{pages: pages()}, nil, limits())
	for want := 1; want <= 2; want++ {
		a, err := svc.Start(t.Context(), "u", "https://x.test/", 1, "")
		if err != nil || a.Number != want {
			t.Fatalf("scan %d = %+v, %v", want, a, err)
		}
		if err := svc.Run(t.Context(), a.ID); err != nil {
			t.Fatal(err)
		}
	}
}

// An administrator may start as many scans as they like: none is refused for load, and every one
// runs once workers are free.
func TestStartNeverRefusesAScanForLoad(t *testing.T) {
	r := newRepo()
	svc := service(r, crawler{pages: pages()}, nil, app.Limits{MonthlyTokens: 1000, SiteTextChars: 10_000, Workers: 3})
	const scans = 300
	for i := range scans {
		if _, err := svc.Start(t.Context(), "admin", fmt.Sprintf("https://site%d.test/", i), 1, ""); err != nil {
			t.Fatalf("scan %d refused: %v", i, err)
		}
	}
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		svc.Work(ctx, func(id uuid.UUID, err error) { t.Errorf("run %s: %v", id, err) })
		close(done)
	}()
	waitAll(t, r, scans, domain.StatusDone)
	cancel()
	<-done
}

// waitAll waits until n analyses have status, or fails the test after a few seconds.
func waitAll(t *testing.T, r *repo, n int, status domain.Status) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		r.mu.Lock()
		got := 0
		for _, a := range r.analyses {
			if a.Status == status {
				got++
			}
		}
		r.mu.Unlock()
		if got == n {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("not all %d analyses reached %s", n, status)
}

func TestGetAndListClampTheLimit(t *testing.T) {
	r := newRepo()
	svc := service(r, crawler{}, nil, limits())
	if _, err := svc.Get(t.Context(), uuid.New()); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("Get(unknown) = %v", err)
	}
	for limit, want := range map[int]int{0: app.DefaultListLimit, 7: 7, 500: app.MaxListLimit} {
		if got, err := svc.List(t.Context(), limit); err != nil || len(got) != want {
			t.Errorf("List(%d) = %d, %v; want %d", limit, len(got), err, want)
		}
	}
}

func TestRunRecordsAllFindings(t *testing.T) {
	r := newRepo()
	m := &matcher{eqs: []domain.Equivalent{
		{Product: "TwenteFlex", Family: "tri_flex_turn_curve", Reason: "spiral tower"},
		{Product: "Plastic belt", Family: "plastic"},
	}}
	svc := service(r, crawler{pages: pages()}, m, limits())
	a, err := svc.Start(t.Context(), "u", "https://x.test/", 5, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Run(t.Context(), a.ID); err != nil {
		t.Fatal(err)
	}
	got := r.analyses[a.ID]
	if got.Status != domain.StatusDone || got.PagesCrawled != 1 || len(got.Keywords) == 0 ||
		len(got.KnownTerms) != 1 || got.Matching != domain.MatchingDone || len(got.Equivalents) != 1 {
		t.Fatalf("after Run: %+v", got)
	}
	// A finished analysis is not run again.
	if err := svc.Run(t.Context(), a.ID); err != nil || m.calls != 1 {
		t.Fatalf("second Run = %v, %d matcher calls", err, m.calls)
	}
}

func TestRunMatching(t *testing.T) {
	cases := map[string]struct {
		source source
		tokens int64
		failOn string
		pages  []domain.Page
		want   domain.Matching
		model  string
	}{
		"no model":        {source{}, 0, "", pages(), domain.MatchingDisabled, ""},
		"settings fail":   {source{err: errBoom}, 0, "", pages(), domain.MatchingFailed, ""},
		"budget spent":    {source{matcher: &matcher{}}, 1000, "", pages(), domain.MatchingSkipped, ""},
		"budget unknown":  {source{matcher: &matcher{}}, 0, "TokensSince", pages(), domain.MatchingSkipped, ""},
		"model fails":     {source{matcher: &matcher{err: errBoom}}, 0, "", pages(), domain.MatchingFailed, model},
		"nothing to read": {source{matcher: &matcher{}}, 0, "", []domain.Page{{}}, domain.MatchingSkipped, ""},
		"nothing found":   {source{matcher: &matcher{}}, 0, "", pages(), domain.MatchingDone, model},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			r := newRepo()
			if c.failOn != "" {
				r.failOn[c.failOn] = true
			}
			r.tokens = c.tokens
			l := limits()
			if name == "nothing to read" {
				l.SiteTextChars = 0
			}
			svc := app.New(r, &directory{}, crawler{pages: c.pages}, c.source, glossary, func() time.Time { return now }, l)
			a, err := svc.Start(t.Context(), "u", "https://x.test/", 5, "")
			if err != nil {
				t.Fatal(err)
			}
			if err := svc.Run(t.Context(), a.ID); err != nil {
				t.Fatal(err)
			}
			if got := r.analyses[a.ID]; got.Matching != c.want || got.Status != domain.StatusDone || got.MatchingModel != c.model {
				t.Fatalf("matching = %s by %q (%s), want %s by %q", got.Matching, got.MatchingModel, got.Status, c.want, c.model)
			}
		})
	}
}

func TestRunFailsTheAnalysisWhenNoPageComesBack(t *testing.T) {
	for name, c := range map[string]struct {
		crawl crawler
		want  string
	}{
		"silent site":  {crawler{}, "No page could be fetched."},
		"broken site":  {crawler{err: errors.New("dial tcp: refused")}, "No page could be fetched: dial tcp: refused"},
		"long failure": {crawler{err: errors.New(strings.Repeat("x", 600))}, ""},
	} {
		t.Run(name, func(t *testing.T) {
			r := newRepo()
			svc := service(r, c.crawl, nil, limits())
			a, err := svc.Start(t.Context(), "u", "https://x.test/", 5, "")
			if err != nil {
				t.Fatal(err)
			}
			if err := svc.Run(t.Context(), a.ID); err != nil {
				t.Fatal(err)
			}
			got := r.analyses[a.ID]
			if got.Status != domain.StatusFailed {
				t.Fatalf("status %s", got.Status)
			}
			if c.want != "" && got.Failure != c.want {
				t.Fatalf("failure %q, want %q", got.Failure, c.want)
			}
			if n := len([]rune(got.Failure)); n > 500 {
				t.Fatalf("failure of %d characters", n)
			}
		})
	}
}

func TestRunFailsAStoredAddressThatIsNoLongerValid(t *testing.T) {
	r := newRepo()
	id := uuid.New()
	r.analyses[id] = &domain.Analysis{ID: id, URL: "https://10.0.0.1/", Status: domain.StatusQueued}
	if err := service(r, crawler{}, nil, limits()).Run(t.Context(), id); err != nil {
		t.Fatal(err)
	}
	if r.analyses[id].Status != domain.StatusFailed {
		t.Fatalf("status %s", r.analyses[id].Status)
	}
}

func TestRunReportsWhatCouldNotBeRecorded(t *testing.T) {
	for _, op := range []string{"Get", "MarkRunning", "AddPage", "SaveFindings", "SaveMatching", "Finish", "Fail"} {
		t.Run(op, func(t *testing.T) {
			r := newRepo()
			c := crawler{pages: pages()}
			if op == "Fail" {
				c = crawler{}
			}
			svc := service(r, c, nil, limits())
			a, err := svc.Start(t.Context(), "u", "https://x.test/", 5, "")
			if err != nil {
				t.Fatal(err)
			}
			r.failOn[op] = true
			err = svc.Run(t.Context(), a.ID)
			if !errors.Is(err, errBoom) {
				t.Fatalf("Run with a failing %s = %v", op, err)
			}
		})
	}
}

func TestRecoverAndWork(t *testing.T) {
	r := newRepo()
	queued, running := uuid.New(), uuid.New()
	r.analyses[queued] = &domain.Analysis{ID: queued, URL: "https://x.test/", MaxPages: 1, Status: domain.StatusQueued}
	r.analyses[running] = &domain.Analysis{ID: running, URL: "https://x.test/", MaxPages: 1, Status: domain.StatusRunning}
	r.order = []uuid.UUID{running, queued}
	svc := service(r, crawler{pages: pages()}, nil, limits())
	if err := svc.Recover(t.Context()); err != nil {
		t.Fatal(err)
	}
	if r.failedRun == "" {
		t.Fatal("running analyses were not failed")
	}
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		svc.Work(ctx, func(uuid.UUID, error) {})
		close(done)
	}()
	deadline := time.After(5 * time.Second)
	for {
		a, _ := r.Get(t.Context(), queued)
		if a.Status == domain.StatusDone {
			break
		}
		select {
		case <-deadline:
			t.Fatal("the queued analysis never ran")
		case <-time.After(10 * time.Millisecond):
		}
	}
	cancel()
	<-done
}

func TestWorkReportsErrors(t *testing.T) {
	r := newRepo()
	svc := service(r, crawler{pages: pages()}, nil, limits())
	a, err := svc.Start(t.Context(), "u", "https://x.test/", 1, "")
	if err != nil {
		t.Fatal(err)
	}
	r.failOn["MarkRunning"] = true
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	reported := make(chan uuid.UUID, 1)
	go svc.Work(ctx, func(id uuid.UUID, _ error) { reported <- id })
	select {
	case id := <-reported:
		if id != a.ID {
			t.Fatalf("reported %s, want %s", id, a.ID)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("no error reported")
	}
}

func TestRecoverFailures(t *testing.T) {
	if err := service(newRepo("FailRunning"), crawler{}, nil, limits()).Recover(t.Context()); !errors.Is(err, errBoom) {
		t.Errorf("FailRunning failure = %v", err)
	}
	if err := service(newRepo("Queued"), crawler{}, nil, limits()).Recover(t.Context()); !errors.Is(err, errBoom) {
		t.Errorf("Queued failure = %v", err)
	}
	// Every waiting analysis is queued again, however many, and runs once workers start.
	r := newRepo()
	for range 3 {
		id := uuid.New()
		r.analyses[id] = &domain.Analysis{ID: id, URL: "https://x.test/", MaxPages: 1, Status: domain.StatusQueued}
		r.order = append(r.order, id)
	}
	svc := service(r, crawler{pages: pages()}, nil, app.Limits{SiteTextChars: 10_000, Workers: 1})
	if err := svc.Recover(t.Context()); err != nil {
		t.Fatalf("Recover = %v", err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		svc.Work(ctx, func(uuid.UUID, error) {})
		close(done)
	}()
	waitAll(t, r, 3, domain.StatusDone)
	cancel()
	<-done
}

func TestStartNamesTheCompany(t *testing.T) {
	d := &directory{}
	svc := app.New(newRepo(), d, crawler{}, source{}, glossary, func() time.Time { return now }, limits())
	if _, err := svc.Start(t.Context(), "u", "https://belts.test/", 1, "Belts Ltd"); err != nil || d.lastName != "Belts Ltd" {
		t.Fatalf("Start = %v, name %q", err, d.lastName)
	}
	d.err = errBoom
	if _, err := svc.Start(t.Context(), "u", "https://belts.test/", 1, ""); !errors.Is(err, errBoom) {
		t.Fatalf("Start with a failing directory = %v", err)
	}
}
