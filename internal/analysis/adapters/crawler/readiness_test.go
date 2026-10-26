package crawler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/JorisJonkers-dev/findability-explorer/internal/analysis/adapters/crawler"
	"github.com/JorisJonkers-dev/findability-explorer/internal/analysis/domain"
)

// readySite serves every readiness file: robots.txt listing a sitemap index, two sitemaps,
// llms.txt, llms-full.txt and a home page; the Markdown twin of the home page is missing.
func readySite(t *testing.T, hits *[]string) http.Handler {
	t.Helper()
	mux := http.NewServeMux()
	serve := func(contentType, body string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			*hits = append(*hits, r.URL.Path)
			if ua := r.Header.Get("User-Agent"); !strings.HasPrefix(ua, crawler.Agent+"/") {
				t.Errorf("User-Agent %q", ua)
			}
			w.Header().Set("Content-Type", contentType)
			_, _ = w.Write([]byte(body))
		}
	}
	mux.HandleFunc("/robots.txt", serve("text/plain", "User-agent: GPTBot\nDisallow: /\n\nSitemap: /sitemap_index.xml\n"))
	mux.HandleFunc("/sitemap_index.xml", serve("application/xml", `<sitemapindex><sitemap><loc>/pages.xml</loc></sitemap><sitemap><loc>/pages.xml</loc></sitemap><sitemap><loc>https://elsewhere.test/x.xml</loc></sitemap></sitemapindex>`))
	mux.HandleFunc("/pages.xml", serve("application/xml", `<urlset><url><loc>http://belts.test/de/a</loc></url><url><loc>http://belts.test/en/b</loc></url></urlset>`))
	mux.HandleFunc("/llms.txt", serve("text/plain", "# Belts\n- [A](/a.md)\n"))
	mux.HandleFunc("/llms-full.txt", serve("text/plain", "# Belts, in full"))
	mux.HandleFunc("/{$}", func(w http.ResponseWriter, r *http.Request) {
		*hits = append(*hits, r.URL.Path)
		w.Header().Set("Server", "cloudflare")
		w.Header().Add("Cf-Ray", "8f0")
		w.Header().Add("Cf-Ray", "8f1")
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><head><meta name="generator" content="WordPress 6.8"></head></html>`))
	})
	return mux
}

func TestReadinessFetchesTheFilesBeforeTheCrawl(t *testing.T) {
	var hits []string
	srv := httptest.NewServer(readySite(t, &hits))
	defer srv.Close()
	c := crawler.New(options(srv, crawler.Options{AllowAddr: loopback, AllowHost: anyHost}))
	files, err := c.Readiness(t.Context(), localTarget(t, srv, "/products"))
	if err != nil {
		t.Fatal(err)
	}
	if files.Robots.Status != 200 || !strings.Contains(files.Robots.Body, "GPTBot") {
		t.Errorf("robots = %+v", files.Robots)
	}
	if len(files.Sitemaps) != 2 || len(files.Sitemaps[1].Pages) != 2 {
		t.Errorf("sitemaps = %+v (the index and one sitemap, the other host's left out)", files.Sitemaps)
	}
	if files.LLMs.Status != 200 || files.LLMsFull.Status != 200 || files.HomeMarkdown.Status != 404 {
		t.Errorf("llms %d, llms-full %d, home twin %d", files.LLMs.Status, files.LLMsFull.Status, files.HomeMarkdown.Status)
	}
	if files.Home.Header["server"] != "cloudflare" || files.Home.Header["cf-ray"] != "8f0, 8f1" || !strings.Contains(files.Home.Body, "WordPress") {
		t.Errorf("home = %+v", files.Home)
	}
	r := domain.AssessReadiness(files)
	if r.Sitemap.Addresses != 2 || !slices.Equal(r.Sitemap.Languages, []string{"de", "en"}) || !r.Markdown || r.Stack.CMS != "WordPress" {
		t.Errorf("readiness = %+v", r)
	}
	want := []string{"/robots.txt", "/", "/llms.txt", "/llms-full.txt", "/sitemap_index.xml", "/pages.xml"}
	if !slices.Equal(hits, want) {
		t.Errorf("fetched %v, want %v (the home page, whatever page the scan starts from; /index.md is a 404 the site does not log here)", hits, want)
	}
}

func TestReadinessOfASiteThatServesNothing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()
	c := crawler.New(options(srv, crawler.Options{AllowAddr: loopback, AllowHost: anyHost}))
	files, err := c.Readiness(t.Context(), localTarget(t, srv, "/"))
	if err != nil {
		t.Fatal(err)
	}
	if files.Robots.Status != 403 || files.Home.Status != 403 || len(files.Sitemaps) != 0 {
		t.Errorf("files = %+v", files)
	}
	refused := crawler.New(options(srv, crawler.Options{AllowHost: anyHost})) // loopback is not public
	files, err = refused.Readiness(t.Context(), localTarget(t, srv, "/"))
	if err != nil || files.Robots.Status != 0 || files.Home.Status != 0 {
		t.Errorf("a site the crawler may not reach is recorded as not fetched: %+v, %v", files, err)
	}
}

func TestReadinessStopsWhenCtxEnds(t *testing.T) {
	var hits []string
	srv := httptest.NewServer(readySite(t, &hits))
	defer srv.Close()
	c := crawler.New(options(srv, crawler.Options{Delay: time.Hour, AllowAddr: loopback, AllowHost: anyHost}))
	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()
	if _, err := c.Readiness(ctx, localTarget(t, srv, "/")); err == nil {
		t.Fatal("Readiness went on after ctx ended")
	}
}
