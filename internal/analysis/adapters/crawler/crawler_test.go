package crawler_test

import (
	"compress/gzip"
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
	"unicode/utf8"

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
	err := c.Crawl(t.Context(), tg, maxPages, scorer(), func(p domain.Page) error {
		pages = append(pages, p)
		return nil
	})
	return pages, err
}

// scorer ranks links by a small belt vocabulary.
func scorer() domain.LinkScorer {
	return domain.NewLinkScorer(domain.NewLexicon([]domain.GlossaryTerm{
		{Text: "eyelink belt", Language: "en", Family: "eyelink", Tier: "product"},
		{Text: "spiral belt", Language: "en", Family: "open_spiral", Tier: "product"},
	}))
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
	// Best first: the start page's language version, then the products section, then the rest in
	// order, skipping the disallowed, missing, non-HTML, off-site (www.example.test) and duplicate
	// ones; the redirect lands on /about.
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

// A page in another encoding than UTF-8 is read in it: a Spanish site in Latin-1, declared in the
// Content-Type or only in a <meta> tag, yields "metálicas", not the bytes that broke the database.
func TestCrawlReadsPagesInTheirOwnEncoding(t *testing.T) {
	latin1 := "<html lang=\"es\"><title>Cintas met\xe1licas</title><p>Transportadoras de malla met\xe1lica\x00.</p></html>"
	for name, serve := range map[string]func(w http.ResponseWriter){
		"declared in the header": func(w http.ResponseWriter) {
			w.Header().Set("Content-Type", "text/html; charset=iso-8859-1")
			_, _ = w.Write([]byte(latin1))
		},
		"declared in a meta tag": func(w http.ResponseWriter) {
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte(`<meta charset="windows-1252">` + latin1))
		},
		"invalid bytes in a UTF-8 page": func(w http.ResponseWriter) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte("<title>Cintas met\xe1licas</title><p>malla met\xc3\xa1lica</p>"))
		},
	} {
		t.Run(name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/robots.txt" {
					http.NotFound(w, r)
					return
				}
				serve(w)
			}))
			defer srv.Close()
			c := crawler.New(options(srv, crawler.Options{AllowAddr: loopback, AllowHost: anyHost}))
			pages, err := crawl(t, c, localTarget(t, srv, "/"), 1)
			if err != nil || len(pages) != 1 {
				t.Fatalf("crawl = %+v, %v", pages, err)
			}
			p := pages[0]
			for _, s := range []string{p.Title, p.Text} {
				if !utf8.ValidString(s) || strings.ContainsRune(s, 0) {
					t.Errorf("not clean UTF-8: %q", s)
				}
			}
			if name != "invalid bytes in a UTF-8 page" && (p.Title != "Cintas metálicas" || !strings.Contains(p.Text, "malla metálica")) {
				t.Errorf("page = %q / %q", p.Title, p.Text)
			}
		})
	}
}

// A site that lets people in but turns automated readers away says so, apart from other failures.
func TestCrawlTellsASiteThatBlocksAutomatedReaders(t *testing.T) {
	for name, answer := range map[string]func(w http.ResponseWriter){
		"403": func(w http.ResponseWriter) { w.WriteHeader(http.StatusForbidden) },
		"429": func(w http.ResponseWriter) { w.WriteHeader(http.StatusTooManyRequests) },
		"a Cloudflare challenge": func(w http.ResponseWriter) {
			w.Header().Set("Cf-Mitigated", "challenge")
			w.WriteHeader(http.StatusServiceUnavailable)
		},
	} {
		t.Run(name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/robots.txt" {
					http.NotFound(w, r)
					return
				}
				answer(w)
			}))
			defer srv.Close()
			c := crawler.New(options(srv, crawler.Options{AllowAddr: loopback, AllowHost: anyHost}))
			if _, err := crawl(t, c, localTarget(t, srv, "/"), 3); !errors.Is(err, domain.ErrBlockedBySite) {
				t.Fatalf("crawl error = %v, want ErrBlockedBySite", err)
			}
		})
	}
	// A page that is simply gone is not a block.
	srv := httptest.NewServer((&site{}).handler(t))
	defer srv.Close()
	c := crawler.New(options(srv, crawler.Options{Delay: time.Millisecond, AllowAddr: loopback, AllowHost: anyHost}))
	if _, err := crawl(t, c, localTarget(t, srv, "/missing"), 1); err == nil || errors.Is(err, domain.ErrBlockedBySite) {
		t.Fatalf("404 error = %v", err)
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
	if err := c.Crawl(t.Context(), localTarget(t, srv, "/"), 5, scorer(), func(domain.Page) error { return boom }); !errors.Is(err, boom) {
		t.Errorf("onPage failure = %v", err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	err := c.Crawl(ctx, localTarget(t, srv, "/"), 5, scorer(), func(domain.Page) error { cancel(); return nil })
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

// deepSite has news first in its menu, a products page about eyelink belts reached only through
// its sitemap index, and a German version of that page.
func deepSite(t *testing.T, robots string, gzipped bool) (*httptest.Server, *[]string) {
	t.Helper()
	var mu sync.Mutex
	var hits []string
	mux := http.NewServeMux()
	html := func(body string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			hits = append(hits, r.URL.Path)
			mu.Unlock()
			w.Header().Set("Content-Type", "text/html")
			_, _ = fmt.Fprint(w, body)
		}
	}
	mux.HandleFunc("/robots.txt", func(w http.ResponseWriter, _ *http.Request) { _, _ = fmt.Fprint(w, robots) })
	mux.HandleFunc("/{$}", html(`<title>Home</title><a href="/news/">News</a> <a href="/contact">Contact</a>`))
	mux.HandleFunc("/news/", html(`<title>News</title>`))
	mux.HandleFunc("/contact", html(`<title>Contact</title>`))
	mux.HandleFunc("/products/eyelink-belts", html(`<html lang="en"><title>Eyelink belts</title>
<link rel="alternate" hreflang="de" href="/de/produkte/oesen"></html>`))
	mux.HandleFunc("/de/produkte/oesen", html(`<html lang="de"><title>Ösenbänder</title></html>`))
	index := `<?xml version="1.0"?><sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
<sitemap><loc>/pages.xml</loc></sitemap><sitemap><loc>https://elsewhere.test/x.xml</loc></sitemap></sitemapindex>`
	pages := `<urlset><url><loc>/products/eyelink-belts</loc></url><url><loc>/news/</loc></url></urlset>`
	mux.HandleFunc("/index.xml", func(w http.ResponseWriter, _ *http.Request) { _, _ = fmt.Fprint(w, index) })
	mux.HandleFunc("/sitemap.xml", func(w http.ResponseWriter, _ *http.Request) { _, _ = fmt.Fprint(w, index) })
	mux.HandleFunc("/pages.xml", func(w http.ResponseWriter, _ *http.Request) { _, _ = fmt.Fprint(w, pages) })
	mux.HandleFunc("/sitemap.xml.gz", func(w http.ResponseWriter, _ *http.Request) {
		if !gzipped {
			_, _ = fmt.Fprint(w, "not gzip")
			return
		}
		zw := gzip.NewWriter(w)
		_, _ = fmt.Fprint(zw, pages)
		_ = zw.Close()
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, &hits
}

func TestCrawlFollowsSitemapsAndFetchesBeltPagesFirst(t *testing.T) {
	for name, robots := range map[string]string{
		"robots lists an index":   "User-agent: *\nDisallow:\nSitemap: /index.xml\n",
		"default sitemap.xml":     "User-agent: *\nDisallow:\n",
		"robots lists a gzip one": "Sitemap: /sitemap.xml.gz\nSitemap: /missing.xml\n",
	} {
		t.Run(name, func(t *testing.T) {
			srv, _ := deepSite(t, robots, true)
			c := crawler.New(options(srv, crawler.Options{AllowAddr: loopback, AllowHost: anyHost}))
			pages, err := crawl(t, c, localTarget(t, srv, "/"), 4)
			if err != nil {
				t.Fatal(err)
			}
			var titles []string
			for _, p := range pages {
				titles = append(titles, p.Title)
			}
			// The eyelink page from the sitemap, then its German version, before news and contact.
			if got := strings.Join(titles, "|"); got != "Home|Eyelink belts|Ösenbänder|News" {
				t.Fatalf("crawled %q", got)
			}
		})
	}
}

func TestCrawlSkipsABrokenGzipSitemap(t *testing.T) {
	srv, _ := deepSite(t, "Sitemap: /sitemap.xml.gz\n", false)
	c := crawler.New(options(srv, crawler.Options{AllowAddr: loopback, AllowHost: anyHost}))
	pages, err := crawl(t, c, localTarget(t, srv, "/"), 10)
	if err != nil || len(pages) != 3 {
		t.Fatalf("crawled %d, %v; want home, news and contact", len(pages), err)
	}
}

func TestCrawlOfOnePageReadsNoSitemap(t *testing.T) {
	srv, hits := deepSite(t, "Sitemap: /index.xml\n", true)
	c := crawler.New(options(srv, crawler.Options{AllowAddr: loopback, AllowHost: anyHost}))
	if pages, err := crawl(t, c, localTarget(t, srv, "/"), 1); err != nil || len(pages) != 1 || len(*hits) != 1 {
		t.Fatalf("crawled %d, %v, hits %v", len(pages), err, *hits)
	}
}

func TestCrawlStopsWhenCtxEndsWhileReadingSitemaps(t *testing.T) {
	srv, _ := deepSite(t, "Sitemap: /index.xml\n", true)
	c := crawler.New(options(srv, crawler.Options{Delay: time.Hour, AllowAddr: loopback, AllowHost: anyHost}))
	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()
	if err := c.Crawl(ctx, localTarget(t, srv, "/"), 5, scorer(), func(domain.Page) error { return nil }); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("crawl = %v", err)
	}
}
