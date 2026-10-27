package app_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/JorisJonkers-dev/findability-explorer/internal/analysis/app"
	"github.com/JorisJonkers-dev/findability-explorer/internal/analysis/domain"
)

// siteCrawler reads robots.txt as found on every site but down.test, which cannot be read, and
// waits for release first when it is set.
type siteCrawler struct {
	crawler
	release chan struct{}
}

func (c siteCrawler) Readiness(_ context.Context, t domain.Target) (domain.ReadinessFiles, error) {
	if c.release != nil {
		<-c.release
	}
	if t.URL().Hostname() == "down.test" {
		return domain.ReadinessFiles{}, errBoom
	}
	return domain.ReadinessFiles{Robots: domain.File{Status: 200, ContentType: "text/plain", Body: "User-agent: *\nAllow: /\n"}}, nil
}

// scansOf stores one scan per company of the given status and website, the newest last.
func scansOf(t *testing.T, r *repo, scans map[string]domain.Analysis) {
	t.Helper()
	for id, a := range scans {
		a.ID, a.CompetitorID, a.CreatedAt = uuid.New(), id, time.Now()
		if _, err := r.Create(context.Background(), a); err != nil {
			t.Fatal(err)
		}
	}
}

func waitForCheck(t *testing.T, svc *app.Service) app.ReadinessCheck {
	t.Helper()
	for range 500 {
		if c := svc.ReadinessCheckState(); !c.Running {
			return c
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatal("the check never ended")
	return app.ReadinessCheck{}
}

func TestReadinessCheckRecordsEachNewestScanAgainWithoutACrawl(t *testing.T) {
	r := newRepo()
	scansOf(t, r, map[string]domain.Analysis{
		"x":    {URL: "https://x.test/", Status: domain.StatusDone},
		"y":    {URL: "https://y.test/", Status: domain.StatusFailed},
		"down": {URL: "https://down.test/", Status: domain.StatusDone},
		"p":    {URL: "https://10.0.0.1/", Status: domain.StatusDone},
		"busy": {URL: "https://busy.test/", Status: domain.StatusRunning},
		"next": {URL: "https://next.test/", Status: domain.StatusQueued},
	})
	now := time.Date(2026, 10, 27, 18, 30, 0, 0, time.UTC)
	release := make(chan struct{})
	svc := app.New(r, &directory{}, siteCrawler{release: release}, source{}, glossary, func() time.Time { return now }, limits())

	started, err := svc.StartReadinessCheck(context.Background())
	if err != nil || !started.Running || started.Total != 4 {
		t.Fatalf("start = %+v, %v", started, err)
	}
	// A second start while it runs changes nothing.
	if again, err := svc.StartReadinessCheck(context.Background()); err != nil || !again.Running || again.Total != 4 {
		t.Fatalf("again = %+v, %v", again, err)
	}
	close(release)
	if done := waitForCheck(t, svc); done.Checked != 4 || done.Failed != 2 {
		t.Fatalf("done = %+v", done)
	}
	for _, a := range r.analyses {
		switch a.CompetitorID {
		case "x", "y":
			if a.Readiness == nil || !a.Readiness.CheckedAt.Equal(now) || a.Readiness.Robots != domain.RobotsFound || len(a.Pages) != 0 {
				t.Errorf("%s = %+v", a.CompetitorID, a.Readiness)
			}
		default:
			if a.Readiness != nil {
				t.Errorf("%s was checked: %+v", a.CompetitorID, a.Readiness)
			}
		}
	}
}

func TestReadinessCheckFailures(t *testing.T) {
	failing := app.New(newRepo("CompetitorStats"), &directory{}, siteCrawler{}, source{}, glossary, time.Now, limits())
	if _, err := failing.StartReadinessCheck(context.Background()); err == nil {
		t.Error("a failing scan list starts a check")
	}

	r := newRepo("SaveReadiness")
	scansOf(t, r, map[string]domain.Analysis{"x": {URL: "https://x.test/", Status: domain.StatusDone}})
	svc := app.New(r, &directory{}, siteCrawler{}, source{}, glossary, time.Now, limits())
	if _, err := svc.StartReadinessCheck(context.Background()); err != nil {
		t.Fatal(err)
	}
	if done := waitForCheck(t, svc); done.Checked != 1 || done.Failed != 1 {
		t.Errorf("a check that cannot be recorded = %+v", done)
	}

	// Once the context ends, no further site is read.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	idle := app.New(newRepo(), &directory{}, siteCrawler{}, source{}, glossary, time.Now, limits())
	idle.CheckReadiness(ctx, []domain.Analysis{{ID: uuid.New(), URL: "https://x.test/"}})
	if c := idle.ReadinessCheckState(); c.Checked != 0 {
		t.Errorf("a cancelled check read %d sites", c.Checked)
	}
}
