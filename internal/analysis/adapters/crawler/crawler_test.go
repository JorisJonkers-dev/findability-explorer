package crawler_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/JorisJonkers-dev/findability-explorer/internal/analysis/adapters/crawler"
	"github.com/JorisJonkers-dev/findability-explorer/internal/analysis/domain"
)

//nolint:gosec // G101: the page holds a link with credentials on purpose, to test that it is skipped
const home = `<!doctype html><html lang="en-GB"><head>
<title>Metal belts | Example</title>
<meta name="description" content="  Spiral and eyelink belts. ">
<link rel="alternate" hreflang="de" href="/de/">
<link rel="stylesheet" href="/style.css">
<style>.x{}</style><script>var hidden = "no";</script>
</head><body>
<nav><a href="/products">Products</a> <a href="https://www.example.test/about#team">About</a></nav>
<h1>Spiral belts</h1><h2> Eyelink  belts </h2><h3></h3>
<p>We make <b>spiral belts</b> for freezers.</p>
<noscript>enable js</noscript>
<a href="/brochure.pdf">PDF</a> <a href="/products?sort=asc">sorted</a> <a href="mailto:x@example.test">mail</a>
<a href="https://other.test/">elsewhere</a> <a href="/private/secret">secret</a> <a href="/missing">gone</a>
<a href="/data.json">data</a> <a href="/redirect">moved</a> <a href="  ">blank</a> <a href="http://[::1">broken</a> <a href="http://u:p@belts.test/x">creds</a>
</body></html>`

type site struct {
	mu   sync.Mutex
	hits []string
}

func (s *site) handler(t *testing.T) http.Handler {
	t.Helper()
	mux := http.NewServeMux()
	page := func(body string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			s.mu.Lock()
			s.hits = append(s.hits, r.URL.Path)
			s.mu.Unlock()
			if ua := r.Header.Get("User-Agent"); !strings.HasPrefix(ua, crawler.Agent+"/") {
				t.Errorf("User-Agent %q", ua)
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(body))
		}
	}
	mux.HandleFunc("/robots.txt", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("User-agent: *\nDisallow: /private\n"))
	})
	mux.HandleFunc("/{$}", page(home))
	mux.HandleFunc("/products", page(`<html><title>Products</title><p>Eyelink belts.</p><a href="/">home</a></html>`))
	mux.HandleFunc("/about", page(`<html><title>About</title></html>`))
	mux.HandleFunc("/de/", page(`<html lang="de"><title>Förderbänder</title></html>`))
	mux.HandleFunc("/data.json", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	})
	mux.HandleFunc("/redirect", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/about", http.StatusFound)
	})
	return mux
}

func loopback(netip.Addr) bool { return true }
func anyHost(string) bool      { return true }

// localTarget returns a Target on the public-looking host belts.test; options() sends every
// connection for it to the test server, so no test needs DNS or the network.
func localTarget(t *testing.T, _ *httptest.Server, path string) domain.Target {
	t.Helper()
	tg, err := domain.NewTarget("http://belts.test" + path)
	if err != nil {
		t.Fatal(err)
	}
	return tg
}

func options(srv *httptest.Server, o crawler.Options) crawler.Options {
	addr := strings.TrimPrefix(srv.URL, "http://")
	o.Rewrite = func(string) string { return addr }
	if o.Delay == 0 {
		o.Delay = time.Millisecond
	}
	return o
}

func crawl(t *testing.T, c *crawler.Crawler, tg domain.Target, maxPages int) ([]domain.Page, error) {
	t.Helper()
	var pages []domain.Page
	err := c.Crawl(t.Context(), tg, maxPages, func(p domain.Page) error {
		pages = append(pages, p)
		return nil
	})
	return pages, err
}

func TestCrawlReadsTheSiteBreadthFirst(t *testing.T) {
	s := &site{}
	srv := httptest.NewServer(s.handler(t))
	defer srv.Close()
	c := crawler.New(options(srv, crawler.Options{Delay: time.Millisecond, AllowAddr: loopback, AllowHost: anyHost, Contact: "ops@example.test"}))
	pages, err := crawl(t, c, localTarget(t, srv, "/"), 10)
	if err != nil {
		t.Fatal(err)
	}
	first := pages[0]
	if first.Title != "Metal belts | Example" || first.Description != "Spiral and eyelink belts." || first.Language != "en-GB" {
		t.Fatalf("first page = %+v", first)
	}
	if strings.Join(first.Headings, "|") != "Spiral belts|Eyelink belts" {
		t.Errorf("headings = %q", first.Headings)
	}
	if !strings.Contains(first.Text, "\nWe make spiral belts for freezers.") || strings.Contains(first.Text, "hidden") ||
		strings.Contains(first.Text, "enable js") || strings.Contains(first.Text, "Metal belts") {
		t.Errorf("text = %q", first.Text)
	}
	var titles []string
	for _, p := range pages {
		titles = append(titles, p.Title)
	}
	// Breadth first: the alternate-language link, then the anchors in order, skipping the
	// disallowed, missing, non-HTML, off-site (www.example.test) and duplicate ones; the redirect
	// lands on /about.
	if got := strings.Join(titles, "|"); got != "Metal belts | Example|Förderbänder|Products|About" {
		t.Fatalf("crawled %q", got)
	}
	for _, hit := range s.hits {
		if strings.HasPrefix(hit, "/private") || hit == "/brochure.pdf" {
			t.Errorf("fetched %s", hit)
		}
	}
}

func TestCrawlStopsAtMaxPages(t *testing.T) {
	srv := httptest.NewServer((&site{}).handler(t))
	defer srv.Close()
	c := crawler.New(options(srv, crawler.Options{Delay: time.Millisecond, AllowAddr: loopback, AllowHost: anyHost}))
	if pages, err := crawl(t, c, localTarget(t, srv, "/"), 2); err != nil || len(pages) != 2 {
		t.Fatalf("crawled %d pages, %v; want 2", len(pages), err)
	}
}

func TestCrawlFailsWhenTheStartPageFails(t *testing.T) {
	srv := httptest.NewServer((&site{}).handler(t))
	defer srv.Close()
	c := crawler.New(options(srv, crawler.Options{Delay: time.Millisecond, AllowAddr: loopback, AllowHost: anyHost}))
	for path, want := range map[string]string{
		"/missing":       "404",
		"/data.json":     "not HTML",
		"/private/thing": "robots.txt does not allow",
	} {
		if _, err := crawl(t, c, localTarget(t, srv, path), 5); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("crawl %s error = %v, want %q", path, err, want)
		}
	}
}

func TestCrawlRefusesPrivateAddresses(t *testing.T) {
	srv := httptest.NewServer((&site{}).handler(t))
	defer srv.Close()
	// The default address check refuses loopback, even behind a public-looking name.
	c := crawler.New(options(srv, crawler.Options{Delay: time.Millisecond}))
	if _, err := crawl(t, c, localTarget(t, srv, "/"), 5); !errors.Is(err, crawler.ErrBlockedAddress) {
		t.Fatalf("crawl error = %v, want ErrBlockedAddress", err)
	}
}

func TestCrawlRefusesRedirectsOffTheAllowedHosts(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/to-private":
			http.Redirect(w, r, "http://intranet/", http.StatusFound)
		case "/to-ftp":
			http.Redirect(w, r, "ftp://example.test/", http.StatusFound)
		default:
			http.Redirect(w, r, r.URL.Path+"x", http.StatusFound) //nolint:gosec // a redirect loop on the test server
		}
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	c := crawler.New(options(srv, crawler.Options{Delay: time.Millisecond, AllowAddr: loopback}))
	for _, path := range []string{"/to-private", "/to-ftp", "/loop"} {
		if _, err := crawl(t, c, localTarget(t, srv, path), 5); err == nil {
			t.Errorf("crawl %s succeeded", path)
		}
	}
}

func TestCrawlStopsWhenOnPageFailsOrCtxEnds(t *testing.T) {
	srv := httptest.NewServer((&site{}).handler(t))
	defer srv.Close()
	c := crawler.New(options(srv, crawler.Options{Delay: 50 * time.Millisecond, AllowAddr: loopback, AllowHost: anyHost}))
	boom := errors.New("boom")
	if err := c.Crawl(t.Context(), localTarget(t, srv, "/"), 5, func(domain.Page) error { return boom }); !errors.Is(err, boom) {
		t.Errorf("onPage failure = %v", err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	err := c.Crawl(ctx, localTarget(t, srv, "/"), 5, func(domain.Page) error { cancel(); return nil })
	if !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled crawl = %v", err)
	}
}

func TestCrawlWithoutRobotsOrWithABrokenOne(t *testing.T) {
	for name, robots := range map[string]http.HandlerFunc{
		"missing": http.NotFound,
		"broken": func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Length", "100")
			_, _ = w.Write([]byte("User-agent: *"))
		},
	} {
		t.Run(name, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc("/robots.txt", robots)
			mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "text/html")
				_, _ = fmt.Fprint(w, `<title>Only</title>`)
			})
			srv := httptest.NewServer(mux)
			defer srv.Close()
			c := crawler.New(options(srv, crawler.Options{Delay: time.Millisecond, AllowAddr: loopback, AllowHost: anyHost}))
			if pages, err := crawl(t, c, localTarget(t, srv, "/"), 5); err != nil || len(pages) != 1 {
				t.Fatalf("crawled %d, %v", len(pages), err)
			}
		})
	}
}

func TestDefaultsAreSet(t *testing.T) {
	if crawler.New(crawler.Options{}) == nil {
		t.Fatal("New returned nil")
	}
}
