package persistence_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/JorisJonkers-dev/findability-explorer/internal/analysis/domain"
)

func TestReadinessRoundTrips(t *testing.T) {
	t.Parallel()
	repo := repository(t)
	ctx := t.Context()
	older := queued(t, repo, "joris", created)
	newer := queued(t, repo, "joris", created.Add(time.Hour))
	none := queued(t, repo, "joris", created.Add(2*time.Hour)) // the newest, but without readiness

	files := domain.ReadinessFiles{
		Robots:   domain.File{Status: 200, ContentType: "text/plain", Body: "User-agent: GPTBot\nDisallow: /\nSitemap: /s.xml\n"},
		Sitemaps: []domain.Sitemap{{Pages: []string{"https://www.example.test/de/a"}}},
		LLMs:     domain.File{Status: 200, ContentType: "text/plain", Body: strings.Repeat("- [A](/a.md)\n", 10_000)},
		Home: domain.File{Status: 200, ContentType: "text/html", Body: `<link rel="alternate" hreflang="nl" href="/">` +
			`<script type="application/ld+json">{"@type":"Organization"}</script>`, Header: map[string]string{"cf-ray": "1"}},
	}
	rd := domain.AssessReadiness(files)
	raw := domain.RawFilesOf(files)
	if err := repo.SaveReadiness(ctx, older.ID, domain.AssessReadiness(domain.ReadinessFiles{}), domain.RawFiles{}); err != nil {
		t.Fatal(err)
	}
	if err := repo.SaveReadiness(ctx, newer.ID, rd, raw); err != nil {
		t.Fatal(err)
	}

	got, err := repo.Get(ctx, newer.ID)
	if err != nil || got.Readiness == nil {
		t.Fatalf("Get = %+v, %v", got, err)
	}
	if !reflect.DeepEqual(*got.Readiness, rd) {
		t.Errorf("readiness = %+v\nwant %+v", *got.Readiness, rd)
	}
	if blank, _ := repo.Get(ctx, none.ID); blank.Readiness != nil {
		t.Errorf("a scan without readiness reads %+v", blank.Readiness)
	}
	back, err := repo.ReadinessFiles(ctx, newer.ID)
	if err != nil || back != raw || !back.LLMsTruncated || back.Robots == "" {
		t.Errorf("raw files = %+v, %v", back.LLMsTruncated, err)
	}
	if _, err := repo.ReadinessFiles(ctx, uuid.New()); !errors.Is(err, domain.ErrNoReadiness) {
		t.Errorf("raw files of nothing = %v", err)
	}
	latest, err := repo.LatestReadiness(ctx)
	if err != nil || len(latest) != 1 {
		t.Fatalf("LatestReadiness = %+v, %v", latest, err)
	}
	if a := latest["example"]; a.ID != newer.ID || a.Readiness == nil || !reflect.DeepEqual(*a.Readiness, rd) {
		t.Errorf("latest = %+v", a)
	}

	// The crawl measures the coverage after the readiness was stored; a scan without one has none to set.
	if err := repo.SaveSitemapCoverage(ctx, newer.ID, domain.SitemapCoverage{Pages: 4, Listed: 3}); err != nil {
		t.Fatal(err)
	}
	if err := repo.SaveSitemapCoverage(ctx, none.ID, domain.SitemapCoverage{Pages: 1}); !errors.Is(err, domain.ErrNoReadiness) {
		t.Errorf("coverage of a scan without readiness = %v", err)
	}
	if latest, err = repo.LatestReadiness(ctx); err != nil || latest["example"].Readiness.Sitemap.Coverage == nil ||
		*latest["example"].Readiness.Sitemap.Coverage != (domain.SitemapCoverage{Pages: 4, Listed: 3}) {
		t.Errorf("coverage = %+v, %v", latest["example"].Readiness, err)
	}
	if old, _ := repo.Get(ctx, older.ID); old.Readiness == nil || old.Readiness.Sitemap.Coverage != nil {
		t.Errorf("a readiness without coverage = %+v", old.Readiness)
	}

	// A check again replaces the scan's readiness, its crawler rules and raw files, and is dated.
	again := domain.AssessReadiness(domain.ReadinessFiles{Robots: domain.File{Status: 200, ContentType: "text/plain", Body: "User-agent: *\nAllow: /\n"}})
	again.CheckedAt = created.Add(48 * time.Hour)
	again.Sitemap.Coverage = &domain.SitemapCoverage{Pages: 4, Listed: 4}
	if err := repo.SaveReadiness(ctx, newer.ID, again, domain.RawFiles{Robots: "User-agent: *"}); err != nil {
		t.Fatal(err)
	}
	latest, err = repo.LatestReadiness(ctx)
	if err != nil {
		t.Fatal(err)
	}
	checked := latest["example"].Readiness
	if checked == nil || !checked.CheckedAt.Equal(again.CheckedAt) || checked.LLMs.Present || len(checked.Bots) != len(again.Bots) ||
		checked.Sitemap.Coverage == nil || checked.Sitemap.Coverage.Listed != 4 {
		t.Errorf("checked again = %+v", checked)
	}
	if back, _ := repo.ReadinessFiles(ctx, newer.ID); back.Robots != "User-agent: *" || back.LLMs != "" {
		t.Errorf("raw files checked again = %+v", back)
	}
}
