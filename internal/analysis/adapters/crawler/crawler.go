// Package crawler implements the analysis Crawler port over HTTP: a polite, same-site crawl that
// follows robots.txt and the site's sitemaps, fetches the pages about belts first, and never
// connects to a non-public address.
// It ports the crawl of github.com/RNgo-Sch/digital-strategy-webscraper.
package crawler

import (
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"syscall"
	"time"

	"golang.org/x/net/html/charset"

	"github.com/JorisJonkers-dev/findability-explorer/internal/analysis/domain"
)

// Agent is the crawler's robots.txt token and the start of its User-Agent.
const Agent = "FindabilityExplorer"

const (
	maxBody      = 2 << 20 // bytes read from one page
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

// Crawl fetches at most maxPages HTML pages on target's site and calls onPage for each. It starts
// with the start page, then seeds the frontier with the site's sitemaps (the ones robots.txt
// lists, or /sitemap.xml) and fetches the pages scorer ranks highest first, so product,
// specification and application pages come before news, contact and legal ones. Each language
// version a page declares ranks just above that page. A page that fails is skipped; Crawl returns
// an error only when the start page fails, onPage fails, or ctx ends.
func (c *Crawler) Crawl(ctx context.Context, target domain.Target, maxPages int, scorer domain.LinkScorer, onPage func(domain.Page) error) error {
	start := target.URL()
	f := domain.NewFrontier(start.Hostname(), scorer)
	f.Seen(start)
	robots := c.robots(ctx, start)
	if !robots.Allowed(start.EscapedPath()) {
		return errors.New("robots.txt does not allow the start page")
	}
	page, links, err := c.fetch(ctx, start)
	if err != nil {
		return err
	}
	f.Fetched(page.URL)
	if err := onPage(page); err != nil {
		return err
	}
	f.Push(links, scorer.Score(start, page.Title))
	if maxPages > 1 {
		if err := c.seedSitemaps(ctx, start, robots, f); err != nil {
			return err
		}
	}
	return c.crawlFrontier(ctx, f, robots, maxPages, onPage)
}

// crawlFrontier fetches the frontier's best pages, after the start page, until maxPages pages
// were fetched or the frontier is empty.
func (c *Crawler) crawlFrontier(ctx context.Context, f *domain.Frontier, robots domain.Robots, maxPages int, onPage func(domain.Page) error) error {
	for fetched := 1; fetched < maxPages; {
		next, score := f.Pop()
		if next == nil {
			break
		}
		if !robots.Allowed(next.EscapedPath()) {
			continue
		}
		page, links, ok, err := c.visit(ctx, f, next)
		if err != nil {
			return err
		}
		if !ok {
			continue
		}
		fetched++
		if err := onPage(page); err != nil {
			return err
		}
		f.Push(links, score)
	}
	return nil
}

// visit pauses, then fetches u. It reports false for a page that failed, or that a redirect led
// to a page fetched already, and fails only when ctx ends.
func (c *Crawler) visit(ctx context.Context, f *domain.Frontier, u *url.URL) (domain.Page, []domain.Link, bool, error) {
	if err := sleep(ctx, c.delay); err != nil {
		return domain.Page{}, nil, false, err
	}
	page, links, err := c.fetch(ctx, u)
	if err != nil {
		return domain.Page{}, nil, false, ctx.Err()
	}
	return page, links, f.Fetched(page.URL), nil
}

// Sitemap limits: how many sitemap files one crawl reads, and how many bytes of each.
const (
	maxSitemaps    = 10
	maxSitemapBody = 10 << 20
)

// seedSitemaps reads the sitemaps robots.txt lists, or /sitemap.xml when it lists none, following
// sitemap indexes, and queues every page of the site they list. A sitemap that fails is skipped.
func (c *Crawler) seedSitemaps(ctx context.Context, start *url.URL, robots domain.Robots, f *domain.Frontier) error {
	queue := sitemapsOf(start, robots)
	seen := map[string]bool{}
	for read := 0; len(queue) > 0 && read < maxSitemaps; {
		u := queue[0]
		queue = queue[1:]
		if seen[u.String()] || domain.SiteOf(u.Hostname()) != domain.SiteOf(start.Hostname()) {
			continue
		}
		seen[u.String()] = true
		read++
		if err := sleep(ctx, c.delay); err != nil {
			return err
		}
		queue = append(queue, c.readSitemap(ctx, u, f)...)
	}
	return nil
}

// sitemapsOf are the sitemaps robots.txt lists, or the site's /sitemap.xml.
func sitemapsOf(start *url.URL, robots domain.Robots) []*url.URL {
	var out []*url.URL
	for _, raw := range robots.Sitemaps() {
		if u, err := start.Parse(raw); err == nil {
			out = append(out, u)
		}
	}
	if len(out) == 0 {
		out = append(out, &url.URL{Scheme: start.Scheme, Host: start.Host, Path: "/sitemap.xml"})
	}
	return out
}

// readSitemap queues the pages one sitemap lists and returns the sitemaps it lists.
func (c *Crawler) readSitemap(ctx context.Context, u *url.URL, f *domain.Frontier) []*url.URL {
	sm, err := c.sitemap(ctx, u)
	if err != nil {
		return nil
	}
	var links []domain.Link
	for _, raw := range sm.Pages {
		if p, err := u.Parse(raw); err == nil {
			links = append(links, domain.Link{URL: p})
		}
	}
	f.Push(links, 0)
	var children []*url.URL
	for _, raw := range sm.Sitemaps {
		if child, err := u.Parse(raw); err == nil {
			children = append(children, child)
		}
	}
	return children
}

// sitemap fetches and parses one sitemap file, gzipped or not.
func (c *Crawler) sitemap(ctx context.Context, u *url.URL) (domain.Sitemap, error) {
	resp, err := c.get(ctx, u)
	if err != nil {
		return domain.Sitemap{}, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return domain.Sitemap{}, fmt.Errorf("%s answered %s", u.Redacted(), resp.Status)
	}
	body := io.Reader(io.LimitReader(resp.Body, maxSitemapBody))
	if strings.HasSuffix(strings.ToLower(u.Path), ".gz") {
		zr, err := gzip.NewReader(body)
		if err != nil {
			return domain.Sitemap{}, err
		}
		defer func() { _ = zr.Close() }()
		body = io.LimitReader(zr, maxSitemapBody)
	}
	return domain.ParseSitemap(body)
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

func (c *Crawler) fetch(ctx context.Context, u *url.URL) (domain.Page, []domain.Link, error) {
	resp, err := c.get(ctx, u)
	if err != nil {
		return domain.Page{}, nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if blocked(resp) {
		return domain.Page{}, nil, fmt.Errorf("%w: %s answered %s", domain.ErrBlockedBySite, u.Redacted(), resp.Status)
	}
	if resp.StatusCode != http.StatusOK {
		return domain.Page{}, nil, fmt.Errorf("%s answered %s", u.Redacted(), resp.Status)
	}
	contentType := resp.Header.Get("Content-Type")
	if mt, _, _ := mime.ParseMediaType(contentType); mt != "text/html" && mt != "application/xhtml+xml" {
		return domain.Page{}, nil, fmt.Errorf("%s is %q, not HTML", u.Redacted(), mt)
	}
	// Read the page in its own encoding (the Content-Type's charset, a byte order mark, or a
	// <meta charset>), so a Latin-1 page yields UTF-8 text.
	body, err := charset.NewReader(io.LimitReader(resp.Body, maxBody), contentType)
	if err != nil {
		return domain.Page{}, nil, fmt.Errorf("%s: %w", u.Redacted(), err)
	}
	return parse(body, resp.Request.URL)
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

// blocked reports whether a site turned the crawler away rather than failing: a 401, 403 or 429,
// or a bot challenge such as Cloudflare's (the cf-mitigated header). A person's browser passes
// such checks; the crawler, which says honestly what it is, does not.
func blocked(resp *http.Response) bool {
	switch resp.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden, http.StatusTooManyRequests:
		return true
	default:
		return resp.Header.Get("Cf-Mitigated") != ""
	}
}

// unwrapURLError drops the *url.Error wrapper, whose message repeats the address.
func unwrapURLError(err error) error {
	var ue *url.Error
	if errors.As(err, &ue) {
		return ue.Err
	}
	return err
}

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
