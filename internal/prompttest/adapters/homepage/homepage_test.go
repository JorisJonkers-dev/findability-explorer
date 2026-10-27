package homepage_test

import (
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/adapters/homepage"
)

const page = `<!doctype html><html><head>
<title>  Metiz   Wire Mesh </title><title>A second title</title>
<meta name="keywords" content="belts">
<meta property="og:description" content="Open Graph description">
<meta name="Description" content="Wire mesh   conveyor belts for ovens.">
<script>var tracking = 1;</script><style>body{}</style>
</head><body>
<nav>Home Products Contact</nav>
<h1>Conveyor <em>belts</em></h1>
<p>We make eyelink and spiral belts in Taiwan since 1983.</p>
<h2>Ovens</h2><h2>Freezers</h2><h3>Washing</h3><h2>Sintering</h2><h2>Glass</h2><h2>Too many</h2><h2> </h2>
<footer>Privacy</footer>
</body></html>`

// fetcher may reach the test server on loopback.
func fetcher(timeout time.Duration) *homepage.Fetcher {
	return homepage.New(homepage.Options{
		Timeout: timeout, Contact: "research@example.com",
		AllowAddr: func(netip.Addr) bool { return true },
		AllowHost: func(h string) bool { return h == "127.0.0.1" },
	})
}

func TestAHomePageIsReadOnce(t *testing.T) {
	var hits atomic.Int32
	var agent string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		switch r.URL.Path {
		case "/":
			http.Redirect(w, r, "/en/", http.StatusFound)
		case "/en/":
			agent = r.Header.Get("User-Agent")
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(page))
		}
	}))
	defer srv.Close()
	got := fetcher(0).Fetch(t.Context(), srv.URL+"/")
	if got.Problem != "" || got.URL != srv.URL+"/en/" || got.Title != "Metiz Wire Mesh" || got.Description != "Wire mesh conveyor belts for ovens." {
		t.Fatalf("page = %+v", got)
	}
	if strings.Join(got.Headings, "|") != "Conveyor belts|Ovens|Freezers|Washing|Sintering|Glass" {
		t.Errorf("headings = %q", got.Headings)
	}
	if got.Text != "Conveyor belts We make eyelink and spiral belts in Taiwan since 1983. Ovens Freezers Washing Sintering Glass Too many" {
		t.Errorf("text = %q", got.Text)
	}
	if agent != "FindabilityExplorer/1.0 (university research; research@example.com)" || hits.Load() != 2 {
		t.Errorf("agent %q, %d requests", agent, hits.Load())
	}
}

func TestAPageThatCannotBeRead(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/missing":
			http.NotFound(w, r)
		case "/pdf":
			w.Header().Set("Content-Type", "application/pdf")
		case "/latin":
			w.Header().Set("Content-Type", "text/html; charset=no-such-charset")
		case "/slow":
			time.Sleep(300 * time.Millisecond)
		case "/away":
			http.Redirect(w, r, "https://intranet.local/", http.StatusFound)
		case "/loop":
			http.Redirect(w, r, "/loop", http.StatusFound)
		}
	}))
	defer srv.Close()
	f := fetcher(100 * time.Millisecond)
	for path, want := range map[string]string{
		"/missing": "the website answered 404 Not Found",
		"/pdf":     `the page is "application/pdf", not HTML`,
		"/latin":   "the page's encoding is unknown",
		"/slow":    "the website did not answer",
		"/away":    "the website did not answer",
		"/loop":    "the website did not answer",
	} {
		if got := f.Fetch(t.Context(), srv.URL+path); got.Problem != want || got.URL != srv.URL+path {
			t.Errorf("%s: %+v, want %q", path, got, want)
		}
	}
	for _, bad := range []string{"ftp://127.0.0.1/", "https://intranet.local/", "://", "http://127.0.0.1:1/\x7f"} {
		if got := f.Fetch(t.Context(), bad); got.Problem != "the address is not a public website" {
			t.Errorf("%q: %+v", bad, got)
		}
	}
}

func TestAPrivateAddressIsNeverDialled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Error("the server was reached")
	}))
	defer srv.Close()
	f := homepage.New(homepage.Options{AllowHost: func(string) bool { return true }})
	if got := f.Fetch(t.Context(), srv.URL+"/"); got.Problem != "the website resolves to an address that is not public" {
		t.Errorf("page = %+v", got)
	}
}
