package domain_test

import (
	"strings"
	"testing"

	"github.com/JorisJonkers-dev/findability-explorer/internal/analysis/domain"
)

func TestParseSitemapReadsPagesAndTheirLanguageVersions(t *testing.T) {
	sm, err := domain.ParseSitemap(strings.NewReader(`<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9" xmlns:xhtml="http://www.w3.org/1999/xhtml">
  <url><loc> https://x.test/en/eyelink </loc>
    <xhtml:link rel="alternate" hreflang="de" href="https://x.test/de/oesen"/>
    <xhtml:link rel="canonical" href="https://x.test/ignored"/>
    <xhtml:link rel="alternate" hreflang="fr"/>
  </url>
  <url><loc></loc></url>
  <loc>https://x.test/outside-an-entry</loc>
</urlset>`))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(sm.Pages, " ") != "https://x.test/en/eyelink https://x.test/de/oesen" || len(sm.Sitemaps) != 0 ||
		strings.Join(sm.Languages, " ") != "de" {
		t.Fatalf("sitemap = %+v", sm)
	}
}

func TestParseSitemapReadsAnIndex(t *testing.T) {
	sm, err := domain.ParseSitemap(strings.NewReader(`<sitemapindex><sitemap><loc>/a.xml</loc></sitemap><sitemap><loc>/b.xml.gz</loc></sitemap></sitemapindex>`))
	if err != nil || strings.Join(sm.Sitemaps, " ") != "/a.xml /b.xml.gz" || len(sm.Pages) != 0 {
		t.Fatalf("index = %+v, %v", sm, err)
	}
}

func TestParseSitemapKeepsWhatCameBeforeACut(t *testing.T) {
	sm, err := domain.ParseSitemap(strings.NewReader(`<urlset><url><loc>/a</loc></url><url><loc>/b</lo`))
	if err != nil || strings.Join(sm.Pages, " ") != "/a" {
		t.Fatalf("cut sitemap = %+v, %v", sm, err)
	}
	if _, err := domain.ParseSitemap(strings.NewReader(`<urlset><url><loc`)); err == nil {
		t.Fatal("a sitemap broken before its first entry parsed")
	}
}

func TestParseSitemapStopsAtItsLimit(t *testing.T) {
	var b strings.Builder
	b.WriteString("<urlset>")
	for range domain.MaxSitemapEntries + 10 {
		b.WriteString("<url><loc>/p</loc></url>")
	}
	b.WriteString("</urlset>")
	sm, err := domain.ParseSitemap(strings.NewReader(b.String()))
	if err != nil || len(sm.Pages) != domain.MaxSitemapEntries {
		t.Fatalf("read %d, %v", len(sm.Pages), err)
	}
}

func TestRobotsListsItsSitemaps(t *testing.T) {
	r := domain.ParseRobots("Sitemap: https://x.test/sitemap_index.xml\nUser-agent: *\nDisallow: /x\nSitemap:\nsitemap: /b.xml\n", "Bot")
	if strings.Join(r.Sitemaps(), " ") != "https://x.test/sitemap_index.xml /b.xml" {
		t.Fatalf("sitemaps = %q", r.Sitemaps())
	}
	named := domain.ParseRobots("User-agent: bot\nDisallow: /\nSitemap: /c.xml\n", "Bot")
	if strings.Join(named.Sitemaps(), " ") != "/c.xml" || named.Allowed("/") {
		t.Fatalf("named group = %+v", named)
	}
}
