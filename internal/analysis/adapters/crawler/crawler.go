// Package crawler implements the analysis Crawler port over HTTP: a polite, same-site,
// breadth-first crawl that follows robots.txt and never connects to a non-public address.
// It ports the crawl of github.com/RNgo-Sch/digital-strategy-webscraper.
package crawler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"path"
	"strings"
	"syscall"
	"time"

	"github.com/JorisJonkers-dev/findability-explorer/internal/analysis/domain"
)

// Agent is the crawler's robots.txt token and the start of its User-Agent.
const Agent = "FindabilityExplorer"

const (
	maxBody      = 2 << 20 // bytes read from one page
	maxQueue     = 500
	maxRedirects = 5
)

// ErrBlockedAddress is returned when a host resolves to an address the crawler may not connect to.
var ErrBlockedAddress = errors.New("the site resolves to an address that is not public")

// Options tune a Crawler; the zero value of each field picks the production default.
type Options struct {
	// Delay is the pause between two requests to the site. Default 500ms.
	Delay time.Duration
	// Timeout bounds one request. Default 10s.
	Timeout time.Duration
	// AllowAddr decides which resolved addresses may be dialled. Default domain.PublicAddr; tests
	// that crawl an httptest server on loopback pass their own.
	AllowAddr func(netip.Addr) bool
	// AllowHost decides which host names may be crawled. Default domain.PublicHost.
	AllowHost func(string) bool
	// Contact is put in the User-Agent so a site owner can reach the operator.
	Contact string
	// Rewrite, when set, changes the address dialled before the address check runs. Tests use it
	// to send a public-looking host to an httptest server; production leaves it nil.
	Rewrite func(address string) string
}

// Crawler fetches pages.
type Crawler struct {
	client    *http.Client
	delay     time.Duration
	allowHost func(string) bool
	userAgent string
}

var _ domain.Crawler = (*Crawler)(nil)

// New returns a Crawler.
func New(o Options) *Crawler {
	if o.Delay == 0 {
		o.Delay = 500 * time.Millisecond
	}
	if o.Timeout == 0 {
		o.Timeout = 10 * time.Second
	}
	if o.AllowAddr == nil {
		o.AllowAddr = domain.PublicAddr
	}
	if o.AllowHost == nil {
		o.AllowHost = domain.PublicHost
	}
	dialer := &net.Dialer{
		Timeout: 5 * time.Second,
		// Control runs after name resolution, on the address actually dialled, so a public name
		// that resolves to a private address is refused too.
		Control: func(_, address string, _ syscall.RawConn) error {
			ap, err := netip.ParseAddrPort(address)
			if err != nil || !o.AllowAddr(ap.Addr()) {
				return ErrBlockedAddress
			}
			return nil
		},
	}
	dial := dialer.DialContext
	if o.Rewrite != nil {
		dial = func(ctx context.Context, network, address string) (net.Conn, error) {
			return dialer.DialContext(ctx, network, o.Rewrite(address))
		}
	}
	transport := &http.Transport{
		Proxy:                 nil, // never through a proxy the environment names: it would bypass Control
		DialContext:           dial,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: o.Timeout,
		MaxIdleConns:          4,
		IdleConnTimeout:       30 * time.Second,
	}
	c := &Crawler{
		delay: o.Delay, allowHost: o.AllowHost,
		userAgent: strings.TrimSpace(Agent + "/1.0 (university research; " + o.Contact + ")"),
	}
	c.client = &http.Client{
		Transport: transport,
		Timeout:   o.Timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxRedirects {
				return fmt.Errorf("more than %d redirects", maxRedirects)
			}
			if (req.URL.Scheme != "http" && req.URL.Scheme != "https") || !c.allowHost(req.URL.Hostname()) {
				return fmt.Errorf("redirect to %s refused", req.URL.Redacted())
			}
			return nil
		},
	}
	return c
}

// Crawl fetches at most maxPages HTML pages on target's site, breadth first, and calls onPage
// for each. A page that fails is skipped; Crawl returns an error only when the start page fails,
// onPage fails, or ctx ends.
func (c *Crawler) Crawl(ctx context.Context, target domain.Target, maxPages int, onPage func(domain.Page) error) error {
	start := target.URL()
	f := frontier{site: siteOf(start.Hostname()), queue: []*url.URL{start}, seen: map[string]bool{key(start): true}}
	robots := c.robots(ctx, start)
	if !robots.Allowed(start.EscapedPath()) {
		return errors.New("robots.txt does not allow the start page")
	}
	fetched := 0
	for next := f.pop(); next != nil && fetched < maxPages; next = f.pop() {
		if !robots.Allowed(next.EscapedPath()) {
			continue
		}
		page, links, err := c.visit(ctx, next, next == start)
		if err != nil {
			if next == start || ctx.Err() != nil {
				return err
			}
			continue
		}
		fetched++
		if err := onPage(page); err != nil {
			return err
		}
		f.push(links)
	}
	return nil
}

// visit fetches one page, pausing first unless it is the first request of the crawl.
func (c *Crawler) visit(ctx context.Context, u *url.URL, first bool) (domain.Page, []*url.URL, error) {
	if !first {
		if err := sleep(ctx, c.delay); err != nil {
			return domain.Page{}, nil, err
		}
	}
	return c.fetch(ctx, u)
}

// frontier is the crawl's queue of same-site pages not yet visited.
type frontier struct {
	site  string
	queue []*url.URL
	seen  map[string]bool
}

func (f *frontier) pop() *url.URL {
	if len(f.queue) == 0 {
		return nil
	}
	next := f.queue[0]
	f.queue = f.queue[1:]
	return next
}

func (f *frontier) push(links []*url.URL) {
	for _, l := range links {
		l = clean(l)
		if l == nil || siteOf(l.Hostname()) != f.site || f.seen[key(l)] || len(f.queue) >= maxQueue {
			continue
		}
		f.seen[key(l)] = true
		f.queue = append(f.queue, l)
	}
}

func (c *Crawler) robots(ctx context.Context, start *url.URL) domain.Robots {
	u := &url.URL{Scheme: start.Scheme, Host: start.Host, Path: "/robots.txt"}
	resp, err := c.get(ctx, u)
	if err != nil {
		return domain.Robots{}
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return domain.Robots{}
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	if err != nil {
		return domain.Robots{}
	}
	return domain.ParseRobots(string(body), Agent)
}

func (c *Crawler) fetch(ctx context.Context, u *url.URL) (domain.Page, []*url.URL, error) {
	resp, err := c.get(ctx, u)
	if err != nil {
		return domain.Page{}, nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return domain.Page{}, nil, fmt.Errorf("%s answered %s", u.Redacted(), resp.Status)
	}
	if mt, _, _ := mime.ParseMediaType(resp.Header.Get("Content-Type")); mt != "text/html" && mt != "application/xhtml+xml" {
		return domain.Page{}, nil, fmt.Errorf("%s is %q, not HTML", u.Redacted(), mt)
	}
	return parse(io.LimitReader(resp.Body, maxBody), resp.Request.URL)
}

func (c *Crawler) get(ctx context.Context, u *url.URL) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml;q=0.9,*/*;q=0.1")
	resp, err := c.client.Do(req)
	if err != nil {
		if errors.Is(err, ErrBlockedAddress) {
			return nil, ErrBlockedAddress
		}
		return nil, fmt.Errorf("%s could not be fetched: %w", u.Redacted(), unwrapURLError(err))
	}
	return resp, nil
}

// unwrapURLError drops the *url.Error wrapper, whose message repeats the address.
func unwrapURLError(err error) error {
	var ue *url.Error
	if errors.As(err, &ue) {
		return ue.Err
	}
	return err
}

// skippedExtensions are file types that are never HTML pages.
func skippedExtension(p string) bool {
	switch strings.ToLower(path.Ext(p)) {
	case ".pdf", ".jpg", ".jpeg", ".png", ".gif", ".webp", ".svg", ".ico", ".css", ".js", ".json",
		".xml", ".zip", ".mp4", ".mp3", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx", ".dwg", ".step", ".stp":
		return true
	default:
		return false
	}
}

// clean returns l without fragment and query, or nil when it is not an http(s) page link. Query
// strings are dropped because they mostly spell filters and sessions that repeat the same page.
func clean(l *url.URL) *url.URL {
	if (l.Scheme != "http" && l.Scheme != "https") || l.User != nil || skippedExtension(l.Path) {
		return nil
	}
	c := *l
	c.Fragment, c.RawQuery, c.RawFragment = "", "", ""
	if c.Path == "" {
		c.Path = "/"
	}
	return &c
}

// siteOf is the host without a leading "www.", so www.example.com and example.com are one site.
func siteOf(host string) string { return strings.TrimPrefix(strings.ToLower(host), "www.") }

func key(u *url.URL) string { return siteOf(u.Hostname()) + strings.TrimSuffix(u.EscapedPath(), "/") }

func sleep(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
