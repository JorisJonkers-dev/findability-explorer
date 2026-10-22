package app_test

import (
	"context"
	"errors"
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
	count     int
	tokens    int64
	failOn    map[string]bool
	failedRun string
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

func (r *repo) Create(_ context.Context, a domain.Analysis) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.err("Create"); err != nil {
		return err
	}
	r.analyses[a.ID] = &a
	r.order = append(r.order, a.ID)
	return nil
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

func (r *repo) CountCreatedSince(context.Context, string, time.Time) (int, error) {
	return r.count, r.err("CountCreatedSince")
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

func (r *repo) SaveMatching(_ context.Context, id uuid.UUID, m domain.Matching, eqs []domain.Equivalent, _ domain.Usage) error {
	return r.update("SaveMatching", id, func(a *domain.Analysis) { a.Matching, a.Equivalents = m, eqs })
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

func (c crawler) Crawl(_ context.Context, _ domain.Target, maxPages int, onPage func(domain.Page) error) error {
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

var now = time.Date(2026, 10, 22, 9, 30, 0, 0, time.UTC)

func limits() app.Limits {
	return app.Limits{PerUserPerHour: 3, MonthlyTokens: 1000, SiteTextChars: 10_000, QueueSize: 2}
}

func glossary() []domain.GlossaryTerm {
	return []domain.GlossaryTerm{{Text: "spiral belt", Language: "en", Family: "tri_flex_turn_curve", Tier: "product"}}
}

func pages() []domain.Page {
	return []domain.Page{{URL: "https://x.test/", Language: "en", Title: "Spiral belt", Text: "Our spiral belt for freezers."}}
}

func service(r *repo, c domain.Crawler, m domain.Matcher, l app.Limits) *app.Service {
	return app.New(r, c, m, glossary, func() time.Time { return now }, l)
}

func TestStartQueuesAValidRequest(t *testing.T) {
	r := newRepo()
	a, err := service(r, crawler{}, nil, limits()).Start(t.Context(), "joris", "https://x.test", 0)
	if err != nil {
		t.Fatal(err)
	}
	if a.Status != domain.StatusQueued || a.MaxPages != domain.DefaultMaxPages || a.URL != "https://x.test/" ||
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
		"a private host":   {newRepo(), "https://10.0.0.1/", 5, domain.ErrInvalidTarget},
		"too many pages":   {newRepo(), "https://x.test/", 31, domain.ErrInvalidMaxPages},
		"a count failure":  {newRepo("CountCreatedSince"), "https://x.test/", 5, errBoom},
		"over the limit":   {&repo{analyses: map[uuid.UUID]*domain.Analysis{}, count: 3}, "https://x.test/", 5, app.ErrRateLimited},
		"a create failure": {newRepo("Create"), "https://x.test/", 5, errBoom},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := service(c.repo, crawler{}, nil, limits()).Start(t.Context(), "u", c.url, c.maxPages); !errors.Is(err, c.want) {
				t.Fatalf("Start error = %v, want %v", err, c.want)
			}
		})
	}
}

func TestStartFailsTheAnalysisWhenTheQueueIsFull(t *testing.T) {
	r := newRepo()
	svc := service(r, crawler{}, nil, app.Limits{PerUserPerHour: 10, QueueSize: 1})
	if _, err := svc.Start(t.Context(), "u", "https://x.test/", 1); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Start(t.Context(), "u", "https://x.test/", 1); !errors.Is(err, app.ErrBusy) {
		t.Fatalf("second Start error = %v, want ErrBusy", err)
	}
	second := r.analyses[r.order[1]]
	if second.Status != domain.StatusFailed || second.Failure != app.ErrBusy.Error() {
		t.Fatalf("the refused analysis is %+v", second)
	}
	r.failOn["Fail"] = true
	if _, err := svc.Start(t.Context(), "u", "https://x.test/", 1); !errors.Is(err, errBoom) {
		t.Fatalf("Start with a failing Fail = %v", err)
	}
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
	a, err := svc.Start(t.Context(), "u", "https://x.test/", 5)
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
		matcher domain.Matcher
		tokens  int64
		failOn  string
		pages   []domain.Page
		want    domain.Matching
	}{
		"no model":        {nil, 0, "", pages(), domain.MatchingDisabled},
		"budget spent":    {&matcher{}, 1000, "", pages(), domain.MatchingSkipped},
		"budget unknown":  {&matcher{}, 0, "TokensSince", pages(), domain.MatchingSkipped},
		"model fails":     {&matcher{err: errBoom}, 0, "", pages(), domain.MatchingFailed},
		"nothing to read": {&matcher{}, 0, "", []domain.Page{{}}, domain.MatchingSkipped},
		"nothing found":   {&matcher{}, 0, "", pages(), domain.MatchingDone},
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
			svc := service(r, crawler{pages: c.pages}, c.matcher, l)
			a, err := svc.Start(t.Context(), "u", "https://x.test/", 5)
			if err != nil {
				t.Fatal(err)
			}
			if err := svc.Run(t.Context(), a.ID); err != nil {
				t.Fatal(err)
			}
			if got := r.analyses[a.ID]; got.Matching != c.want || got.Status != domain.StatusDone {
				t.Fatalf("matching = %s (%s), want %s", got.Matching, got.Status, c.want)
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
			a, err := svc.Start(t.Context(), "u", "https://x.test/", 5)
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
			a, err := svc.Start(t.Context(), "u", "https://x.test/", 5)
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
	a, err := svc.Start(t.Context(), "u", "https://x.test/", 1)
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
	// More queued analyses than the queue holds, and nobody working: Recover waits until ctx ends.
	r := newRepo()
	for range 3 {
		id := uuid.New()
		r.analyses[id] = &domain.Analysis{ID: id, Status: domain.StatusQueued}
		r.order = append(r.order, id)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()
	if err := service(r, crawler{}, nil, app.Limits{QueueSize: 1}).Recover(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Recover with a full queue = %v", err)
	}
}
