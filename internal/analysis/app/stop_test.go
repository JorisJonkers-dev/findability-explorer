package app_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/JorisJonkers-dev/findability-explorer/internal/analysis/app"
	"github.com/JorisJonkers-dev/findability-explorer/internal/analysis/domain"
)

func (r *repo) StopQueued(_ context.Context, scope domain.StopScope, reason string, at time.Time) ([]uuid.UUID, error) {
	return r.stop("StopQueued", scope, domain.StatusQueued, func(a *domain.Analysis) {
		a.Status, a.Failure, a.FinishedAt = domain.StatusFailed, reason, at
		r.stops[a.ID] = true
	})
}

func (r *repo) RequestStop(_ context.Context, scope domain.StopScope, _ time.Time) ([]uuid.UUID, error) {
	return r.stop("RequestStop", scope, domain.StatusRunning, func(a *domain.Analysis) { r.stops[a.ID] = true })
}

// stop applies fn to every scan of status that scope picks, oldest first, and returns their ids.
func (r *repo) stop(op string, scope domain.StopScope, status domain.Status, fn func(*domain.Analysis)) ([]uuid.UUID, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.err(op); err != nil {
		return nil, err
	}
	var ids []uuid.UUID
	for _, id := range r.order {
		a := r.analyses[id]
		if a.Status == status && (scope.ID == uuid.Nil || scope.ID == id) && (!scope.RescanOnly || a.Rescan) {
			fn(a)
			ids = append(ids, id)
		}
	}
	return ids, nil
}

func (r *repo) StopRequested(_ context.Context, id uuid.UUID) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	asked := r.stops[id] || (r.stopAt > 0 && r.analyses[id].PagesCrawled >= r.stopAt)
	return asked, r.err("StopRequested")
}

// stopper is a site whose scan an administrator stops at one point: while its readiness files
// are read ("readiness"), after its first page ("crawl"), or while its products are matched
// ("match"). Like the real crawler and matcher, it ends once its context ends.
type stopper struct {
	at    string
	stop  func()
	pages []domain.Page
}

func (c *stopper) Readiness(ctx context.Context, _ domain.Target) (domain.ReadinessFiles, error) {
	if c.at == "readiness" {
		c.stop()
	}
	return domain.ReadinessFiles{}, ctx.Err()
}

func (c *stopper) Crawl(ctx context.Context, _ domain.Target, _ int, _ domain.LinkScorer, onPage func(domain.Page) error) error {
	for i, p := range c.pages {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := onPage(p); err != nil {
			return err
		}
		if i == 0 && c.at == "crawl" {
			c.stop()
		}
	}
	return nil
}

func (c *stopper) Match(ctx context.Context, _ string, _ bool, _ []domain.TribeltFamily) ([]domain.Equivalent, domain.Usage, error) {
	if c.at == "match" {
		c.stop()
	}
	return nil, domain.Usage{}, ctx.Err()
}

func threePages() []domain.Page {
	return []domain.Page{
		{URL: "https://x.test/", Language: "en", Text: "Spiral belt."},
		{URL: "https://x.test/a", Language: "en", Text: "Eyelink belt."},
		{URL: "https://x.test/b", Language: "en", Text: "Belts."},
	}
}

// A scan stopped while it runs on this instance ends at once, keeps the pages it read so far, and
// is recorded as stopped by hand, wherever it was.
func TestARunningScanStopsAtOnceAndKeepsThePagesItRead(t *testing.T) {
	for at, read := range map[string]int{"readiness": 0, "crawl": 1, "match": 3} {
		t.Run(at, func(t *testing.T) {
			r := newRepo()
			site := &stopper{at: at, pages: threePages()}
			svc := app.New(r, &directory{}, site, source{matcher: site}, glossary, func() time.Time { return now }, limits())
			a, err := svc.Start(t.Context(), "admin", "https://x.test/", 3, "")
			if err != nil {
				t.Fatal(err)
			}
			site.stop = func() {
				if err := svc.StopScan(t.Context(), a.ID); err != nil {
					t.Errorf("StopScan = %v", err)
				}
			}
			if err := svc.Run(t.Context(), a.ID); err != nil {
				t.Fatal(err)
			}
			got := r.analyses[a.ID]
			if !got.StoppedByHand() || len(got.Pages) != read || !got.FinishedAt.Equal(now) {
				t.Fatalf("stopped scan = %+v", got)
			}
		})
	}
}

// A stop asked on another instance reaches the scan through the database: the scan sees it
// between pages and stops there.
func TestAStopAskedElsewhereIsSeenBetweenPages(t *testing.T) {
	r := newRepo()
	r.stopAt = 2
	svc := service(r, crawler{pages: threePages()}, nil, limits())
	a, err := svc.Start(t.Context(), "admin", "https://x.test/", 3, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Run(t.Context(), a.ID); err != nil {
		t.Fatal(err)
	}
	if got := r.analyses[a.ID]; !got.StoppedByHand() || len(got.Pages) != 2 || got.Keywords != nil {
		t.Fatalf("stopped scan = %+v", got)
	}
}

// When the stop cannot be read, the scan runs on: a stop is never made up.
func TestAScanWhoseStopCannotBeReadRunsOn(t *testing.T) {
	r := newRepo("StopRequested")
	svc := service(r, crawler{pages: threePages()}, nil, limits())
	a, err := svc.Start(t.Context(), "admin", "https://x.test/", 3, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Run(t.Context(), a.ID); err != nil {
		t.Fatal(err)
	}
	if got := r.analyses[a.ID]; got.Status != domain.StatusDone || len(got.Pages) != 3 {
		t.Fatalf("scan = %+v", got)
	}
}

// A scan running on another instance is asked to stop; the stop succeeds although nothing here
// runs it.
func TestAScanRunningElsewhereIsAskedToStop(t *testing.T) {
	r := running()
	id := r.order[0]
	if err := service(r, crawler{}, nil, limits()).StopScan(t.Context(), id); err != nil {
		t.Fatal(err)
	}
	if !r.stops[id] || r.analyses[id].Status != domain.StatusRunning {
		t.Fatalf("scan = %+v, asked %v", r.analyses[id], r.stops[id])
	}
}

// A waiting scan that is removed never runs; the scans behind it do.
func TestARemovedWaitingScanNeverRuns(t *testing.T) {
	r := newRepo()
	svc := service(r, crawler{pages: pages()}, nil, app.Limits{SiteTextChars: 10_000, Workers: 1})
	first, err := svc.Start(t.Context(), "admin", "https://x.test/", 1, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Start(t.Context(), "admin", "https://y.test/", 1, ""); err != nil {
		t.Fatal(err)
	}
	if err := svc.StopScan(t.Context(), first.ID); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		svc.Work(ctx, func(id uuid.UUID, err error) { t.Errorf("run %s: %v", id, err) })
		close(done)
	}()
	waitAll(t, r, 1, domain.StatusDone)
	cancel()
	<-done
	if got := r.analyses[first.ID]; !got.StoppedByHand() || got.PagesCrawled != 0 {
		t.Fatalf("removed scan = %+v", got)
	}
}

// Every waiting scan is removed at once; a running one runs on.
func TestRemovingTheWaitingScansLeavesTheRunningOnes(t *testing.T) {
	r := running()
	svc := service(r, crawler{}, nil, limits())
	var waiting []uuid.UUID
	for _, site := range []string{"https://y.test/", "https://z.test/"} {
		a, err := svc.Start(t.Context(), "admin", site, 1, "")
		if err != nil {
			t.Fatal(err)
		}
		waiting = append(waiting, a.ID)
	}
	if err := svc.RemoveWaitingScans(t.Context()); err != nil {
		t.Fatal(err)
	}
	for _, id := range waiting {
		if !r.analyses[id].StoppedByHand() {
			t.Errorf("waiting scan = %+v", r.analyses[id])
		}
	}
	if running := r.analyses[r.order[0]]; running.Status != domain.StatusRunning || r.stops[running.ID] {
		t.Errorf("running scan = %+v", running)
	}
	if err := svc.RemoveWaitingScans(t.Context()); !errors.Is(err, app.ErrNothingToStop) {
		t.Errorf("removing an empty queue = %v", err)
	}
}

// Stopping a rescan stops each of its scans, waiting or running, and no other scan.
func TestStoppingARescanStopsItsScansOnly(t *testing.T) {
	r := newRepo()
	add := func(company string, status domain.Status, rescan bool) uuid.UUID {
		id := uuid.New()
		r.analyses[id] = &domain.Analysis{ID: id, CompetitorID: company, Status: status, Rescan: rescan}
		r.order = append(r.order, id)
		return id
	}
	waiting, scanning := add("a", domain.StatusQueued, true), add("b", domain.StatusRunning, true)
	single, other := add("c", domain.StatusQueued, false), add("d", domain.StatusRunning, false)
	svc := service(r, crawler{}, nil, limits())
	if err := svc.StopRescan(t.Context()); err != nil {
		t.Fatal(err)
	}
	if !r.analyses[waiting].StoppedByHand() || !r.stops[scanning] {
		t.Errorf("the rescan's scans: %+v, asked %v", r.analyses[waiting], r.stops[scanning])
	}
	if r.analyses[single].Status != domain.StatusQueued || r.stops[other] {
		t.Errorf("other scans: %+v, asked %v", r.analyses[single], r.stops[other])
	}
	r.analyses[scanning].Status = domain.StatusFailed
	if err := svc.StopRescan(t.Context()); !errors.Is(err, app.ErrNothingToStop) {
		t.Errorf("stopping no rescan = %v", err)
	}
}

// A scan that is neither waiting nor running has nothing to stop, and neither has an unknown id.
func TestAFinishedOrUnknownScanHasNothingToStop(t *testing.T) {
	r := newRepo()
	id := uuid.New()
	r.analyses[id] = &domain.Analysis{ID: id, Status: domain.StatusDone}
	r.order = append(r.order, id)
	svc := service(r, crawler{}, nil, limits())
	for _, id := range []uuid.UUID{id, uuid.New()} {
		if err := svc.StopScan(t.Context(), id); !errors.Is(err, app.ErrNothingToStop) {
			t.Errorf("StopScan(%s) = %v", id, err)
		}
	}
}

func TestAStopThatCannotBeRecordedFails(t *testing.T) {
	for _, op := range []string{"StopQueued", "RequestStop"} {
		svc := service(newRepo(op), crawler{}, nil, limits())
		stops := map[string]func(context.Context) error{
			"scan":    func(ctx context.Context) error { return svc.StopScan(ctx, uuid.New()) },
			"rescan":  svc.StopRescan,
			"waiting": svc.RemoveWaitingScans,
		}
		for name, stop := range stops {
			if name == "waiting" && op == "RequestStop" {
				continue // removing the waiting scans asks no running one
			}
			if err := stop(t.Context()); !errors.Is(err, errBoom) {
				t.Errorf("%s with a failing %s = %v", name, op, err)
			}
		}
	}
}

// A readiness check that is stopped ends without reading further sites and says it was stopped;
// a site it was reading is not counted.
func TestAReadinessCheckStopsAndSaysSo(t *testing.T) {
	r := newRepo()
	scansOf(t, r, map[string]domain.Analysis{
		"x": {URL: "https://x.test/", Status: domain.StatusDone},
		"y": {URL: "https://y.test/", Status: domain.StatusDone},
		"z": {URL: "https://z.test/", Status: domain.StatusDone},
	})
	release, reading := make(chan struct{}), make(chan struct{}, 1)
	svc := app.New(r, &directory{}, siteCrawler{release: release, reading: reading}, source{}, glossary, time.Now, limits())
	if err := svc.StopReadinessCheck(t.Context()); !errors.Is(err, app.ErrNothingToStop) {
		t.Fatalf("stopping no check = %v", err)
	}
	if _, err := svc.StartReadinessCheck(context.Background()); err != nil {
		t.Fatal(err)
	}
	<-reading // the check reads its first site
	if err := svc.StopReadinessCheck(t.Context()); err != nil {
		t.Fatal(err)
	}
	close(release)
	if done := waitForCheck(t, svc); !done.Stopped || done.Checked != 0 || done.Total != 3 {
		t.Fatalf("stopped check = %+v", done)
	}
	// A new check starts afresh.
	again, err := svc.StartReadinessCheck(context.Background())
	if err != nil || again.Stopped {
		t.Fatalf("a new check = %+v, %v", again, err)
	}
	if done := waitForCheck(t, svc); done.Stopped || done.Checked != 3 {
		t.Fatalf("the new check = %+v", done)
	}
}
