// Package homepage implements the prompt-test HomePages port over HTTP: it fetches one company's
// home page, once, and keeps what it says of the company, clipped short: its title, its meta
// description, its first headings and the start of its text (ADR-0009). It is as polite as the
// analysis crawler: it says what it is in its User-Agent, gives up after a timeout, reads a
// limited part of the page, and never connects to an address that is not public.
package homepage

import (
	"cmp"
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

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	"golang.org/x/net/html/charset"

	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/app"
	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain"
	"github.com/JorisJonkers-dev/findability-explorer/internal/shared/site"
)

// Agent is the start of the User-Agent, the same robots.txt token the analysis crawler uses.
const Agent = "FindabilityExplorer"

const (
	maxBody      = 1 << 20 // bytes read of the page
	maxRedirects = 5
	maxTitle     = 200
	maxDesc      = 400
	maxHeadings  = 6
	maxHeading   = 120
	maxText      = 800
)

// errBlockedAddress is returned when a host resolves to an address that is not public.
var errBlockedAddress = errors.New("the website resolves to an address that is not public")

// Options tune a Fetcher; the zero value of each field picks the production default.
type Options struct {
	// Timeout bounds the request. Default 10s.
	Timeout time.Duration
	// Contact is put in the User-Agent so a site owner can reach the operator.
	Contact string
	// AllowAddr decides which resolved addresses may be dialled. Default site.PublicAddr.
	AllowAddr func(netip.Addr) bool
	// AllowHost decides which host names may be fetched. Default site.PublicHost.
	AllowHost func(string) bool
}

// Fetcher fetches home pages.
type Fetcher struct {
	client    *http.Client
	allowHost func(string) bool
	userAgent string
}

var _ app.HomePages = (*Fetcher)(nil)

// New returns a Fetcher.
func New(o Options) *Fetcher {
	if o.Timeout == 0 {
		o.Timeout = 10 * time.Second
	}
	if o.AllowAddr == nil {
		o.AllowAddr = site.PublicAddr
	}
	if o.AllowHost == nil {
		o.AllowHost = site.PublicHost
	}
	dialer := &net.Dialer{
		Timeout: 5 * time.Second,
		// Control runs on the address actually dialled, so a public name that resolves to a
		// private address is refused too.
		Control: func(_, address string, _ syscall.RawConn) error {
			ap, err := netip.ParseAddrPort(address)
			if err != nil || !o.AllowAddr(ap.Addr()) {
				return errBlockedAddress
			}
			return nil
		},
	}
	f := &Fetcher{allowHost: o.AllowHost, userAgent: strings.TrimSpace(Agent + "/1.0 (university research; " + o.Contact + ")")}
	f.client = &http.Client{
		Transport: &http.Transport{
			Proxy:                 nil, // never through a proxy the environment names: it would bypass Control
			DialContext:           dialer.DialContext,
			TLSHandshakeTimeout:   5 * time.Second,
			ResponseHeaderTimeout: o.Timeout,
			MaxIdleConns:          4,
			IdleConnTimeout:       30 * time.Second,
		},
		Timeout: o.Timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxRedirects {
				return fmt.Errorf("more than %d redirects", maxRedirects)
			}
			if (req.URL.Scheme != "http" && req.URL.Scheme != "https") || !f.allowHost(req.URL.Hostname()) {
				return fmt.Errorf("a redirect to %s was refused", req.URL.Redacted())
			}
			return nil
		},
	}
	return f
}

// Fetch fetches website's home page once and returns what it says, or, in Problem, why it could
// not be read.
func (f *Fetcher) Fetch(ctx context.Context, website string) domain.HomePage {
	page := domain.HomePage{URL: website}
	u, err := url.Parse(website)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || !f.allowHost(u.Hostname()) {
		page.Problem = "the address is not a public website"
		return page
	}
	if err := f.read(ctx, u, &page); err != nil {
		page.Problem = err.Error()
	}
	return page
}

func (f *Fetcher) read(ctx context.Context, u *url.URL, page *domain.HomePage) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return errors.New("the address is not a public website")
	}
	req.Header.Set("User-Agent", f.userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml;q=0.9,*/*;q=0.1")
	resp, err := f.client.Do(req)
	if err != nil {
		if errors.Is(err, errBlockedAddress) {
			return errBlockedAddress
		}
		return errors.New("the website did not answer")
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("the website answered %s", resp.Status)
	}
	contentType := resp.Header.Get("Content-Type")
	if mt, _, _ := mime.ParseMediaType(contentType); mt != "text/html" && mt != "application/xhtml+xml" {
		return fmt.Errorf("the page is %q, not HTML", mt)
	}
	body, err := charset.NewReader(io.LimitReader(resp.Body, maxBody), contentType)
	if err != nil {
		return errors.New("the page's encoding is unknown")
	}
	doc, err := html.Parse(body)
	if err != nil {
		return errors.New("the page could not be read")
	}
	r := reader{page: page}
	r.walk(doc)
	page.URL = resp.Request.URL.String()
	page.Title = clip(page.Title, maxTitle)
	page.Description = clip(cmp.Or(r.description, r.og), maxDesc)
	page.Text = clip(r.text.String(), maxText)
	return nil
}

// reader gathers what a page says as it walks it.
type reader struct {
	page        *domain.HomePage
	text        strings.Builder
	description string
	og          string
}

func (r *reader) walk(n *html.Node) {
	if n.Type == html.ElementNode {
		switch n.DataAtom { //nolint:exhaustive // every other element keeps its text
		case atom.Script, atom.Style, atom.Noscript, atom.Template, atom.Svg, atom.Nav, atom.Footer, atom.Iframe:
			return
		case atom.Title:
			if r.page.Title == "" {
				r.page.Title = textOf(n)
			}
			return
		case atom.Meta:
			r.meta(n)
		case atom.H1, atom.H2, atom.H3:
			if h := clip(textOf(n), maxHeading); h != "" && len(r.page.Headings) < maxHeadings {
				r.page.Headings = append(r.page.Headings, h)
			}
		}
	}
	if n.Type == html.TextNode && r.text.Len() < maxText*2 {
		if t := collapse(n.Data); t != "" {
			r.text.WriteString(t + " ")
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		r.walk(c)
	}
}

// meta keeps the page's description: its meta description, else its Open Graph one.
func (r *reader) meta(n *html.Node) {
	name := attr(n, "name")
	if name == "" {
		name = attr(n, "property")
	}
	name = strings.ToLower(name)
	switch {
	case name == "description":
		r.description = collapse(attr(n, "content"))
	case name == "og:description" && r.og == "":
		r.og = collapse(attr(n, "content"))
	}
}

func attr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if strings.EqualFold(a.Key, key) {
			return a.Val
		}
	}
	return ""
}

func textOf(n *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			b.WriteString(n.Data + " ")
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return collapse(b.String())
}

func collapse(s string) string { return strings.Join(strings.Fields(s), " ") }

// clip collapses s and cuts it to at most n characters.
func clip(s string, n int) string {
	s = collapse(s)
	if r := []rune(s); len(r) > n {
		return string(r[:n])
	}
	return s
}
