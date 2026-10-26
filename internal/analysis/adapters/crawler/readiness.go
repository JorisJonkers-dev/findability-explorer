package crawler

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/JorisJonkers-dev/findability-explorer/internal/analysis/domain"
)

// Readiness fetches the files that tell search engines and AI assistants about target's site, a
// pause before each after the first: robots.txt, the home page, llms.txt, llms-full.txt, the home
// page's Markdown twin (/index.md), and the sitemaps robots.txt lists (or /sitemap.xml) with the
// sitemaps an index lists. A file that fails is recorded as not fetched, with status 0; Readiness
// fails only when ctx ends. It asks as the crawler, as the page crawl does, never as a browser.
func (c *Crawler) Readiness(ctx context.Context, target domain.Target) (domain.ReadinessFiles, error) {
	start := target.URL()
	at := func(path string) *url.URL { return &url.URL{Scheme: start.Scheme, Host: start.Host, Path: path} }
	var files domain.ReadinessFiles
	steps := []struct {
		into *domain.File
		path string
	}{
		{&files.Robots, "/robots.txt"},
		{&files.Home, "/"},
		{&files.LLMs, "/llms.txt"},
		{&files.LLMsFull, "/llms-full.txt"},
		{&files.HomeMarkdown, "/index.md"},
	}
	for i, step := range steps {
		if i > 0 {
			if err := sleep(ctx, c.delay); err != nil {
				return domain.ReadinessFiles{}, err
			}
		}
		*step.into = c.file(ctx, at(step.path))
	}
	robots := domain.Robots{}
	if files.Robots.Status == http.StatusOK {
		robots = domain.ParseRobots(files.Robots.Body, Agent)
	}
	sitemaps, err := c.readinessSitemaps(ctx, start, robots)
	files.Sitemaps = sitemaps
	return files, err
}

// readinessSitemaps reads at most maxSitemaps sitemap files of the site, following indexes.
func (c *Crawler) readinessSitemaps(ctx context.Context, start *url.URL, robots domain.Robots) ([]domain.Sitemap, error) {
	queue := sitemapsOf(start, robots)
	seen := map[string]bool{}
	var out []domain.Sitemap
	for len(queue) > 0 && len(seen) < maxSitemaps {
		u := queue[0]
		queue = queue[1:]
		if seen[u.String()] || domain.SiteOf(u.Hostname()) != domain.SiteOf(start.Hostname()) {
			continue
		}
		seen[u.String()] = true
		if err := sleep(ctx, c.delay); err != nil {
			return nil, err
		}
		sm, err := c.sitemap(ctx, u)
		if err != nil {
			continue
		}
		out = append(out, sm)
		for _, raw := range sm.Sitemaps {
			if child, err := u.Parse(raw); err == nil {
				queue = append(queue, child)
			}
		}
	}
	return out, nil
}

// file fetches u as it is, whatever its status, with its headers by lower-case name.
func (c *Crawler) file(ctx context.Context, u *url.URL) domain.File {
	resp, err := c.get(ctx, u)
	if err != nil {
		return domain.File{}
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	header := make(map[string]string, len(resp.Header))
	for name, values := range resp.Header {
		header[strings.ToLower(name)] = strings.Join(values, ", ")
	}
	return domain.File{Status: resp.StatusCode, ContentType: resp.Header.Get("Content-Type"), Body: string(body), Header: header}
}
