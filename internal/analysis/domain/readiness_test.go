package domain_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/JorisJonkers-dev/findability-explorer/internal/analysis/domain"
)

// ok is a file the site served.
func ok(contentType, body string) domain.File {
	return domain.File{Status: 200, ContentType: contentType, Body: body}
}

const homeHTML = `<!doctype html><html lang="nl"><head>
<meta name="generator" content="WordPress 6.8.1">
<!-- This site is optimized with the Yoast SEO plugin v25.1 - https://yoast.com/wordpress/plugins/seo/ -->
<link rel="alternate" hreflang="nl" href="https://x.test/">
<link rel="alternate" hreflang="de-DE" href="https://x.test/de/">
<link rel="alternate" hreflang="en-GB" href="https://x.test/en/">
<link rel="alternate" hreflang="fr" href="https://x.test/fr/">
<link rel="alternate" hreflang="x-default" href="https://x.test/">
<script type="application/ld+json">{"@context":"https://schema.org","@graph":[
  {"@type":"WebSite","publisher":{"@type":"Organization","name":"X"}},
  {"@type":["Product","Thing"],"name":"Eyelink belt"}]}</script>
<script type="application/ld+json">[{"@type":"BreadcrumbList"}]</script>
<script type="application/ld+json">not json</script>
<script>var notData = {"@type": "Person"}</script>
</head><body><p>Hello</p></body></html>`

func fullSite() domain.ReadinessFiles {
	return domain.ReadinessFiles{
		Robots: ok("text/plain", "User-agent: *\nAllow: /\n\nUser-agent: GPTBot\nDisallow: /\n\nSitemap: https://x.test/sitemap_index.xml\n"),
		Sitemaps: []domain.Sitemap{
			{Sitemaps: []string{"https://x.test/page-sitemap.xml"}},
			{
				Pages: []string{
					"https://x.test/", "https://x.test/de/oesenband", "https://x.test/en/eyelink", "https://x.test/en/eyelink",
					"https://x.test/fr/bande",
				},
				Languages: []string{"de-DE", "nl", "fr"},
			},
		},
		LLMs:         ok("text/markdown; charset=utf-8", "# X\n\n- [Eyelink](https://x.test/en/eyelink.md): belts\n- [About](/about.md)\n- [Contact](https://x.test/contact)\n"),
		LLMsFull:     ok("text/plain", "# X, every page"),
		Home:         domain.File{Status: 200, ContentType: "text/html", Body: homeHTML, Header: map[string]string{"server": "cloudflare", "cf-ray": "8f0"}},
		HomeMarkdown: domain.File{Status: 404},
	}
}

func TestReadinessOfASiteThatFollowsMostOfTheAdvice(t *testing.T) {
	r := domain.AssessReadiness(fullSite())
	if r.Robots != domain.RobotsFound {
		t.Errorf("robots = %s, want found", r.Robots)
	}
	if got := policyOf(t, r.Bots, "GPTBot"); got.Access != domain.AccessBlocked {
		t.Errorf("GPTBot = %+v", got)
	}
	want := domain.SitemapSummary{Declared: true, Files: 2, Addresses: 4, Languages: []string{"nl", "de", "en"}}
	if r.Sitemap.Declared != want.Declared || r.Sitemap.Files != want.Files || r.Sitemap.Addresses != want.Addresses ||
		!slices.Equal(r.Sitemap.Languages, want.Languages) {
		t.Errorf("sitemap = %+v, want %+v", r.Sitemap, want)
	}
	if r.LLMs != (domain.LLMsFacts{Present: true, Bytes: len(fullSite().LLMs.Body), Links: 3}) || !r.LLMsFull || !r.Markdown {
		t.Errorf("llms = %+v, full %v, markdown %v", r.LLMs, r.LLMsFull, r.Markdown)
	}
	if !r.Home || !slices.Equal(r.StructuredData, []string{"BreadcrumbList", "Organization", "Product", "Thing", "WebSite"}) {
		t.Errorf("structured data = %v (home read %v)", r.StructuredData, r.Home)
	}
	if !slices.Equal(r.Languages, []string{"nl", "de", "en", "fr"}) {
		t.Errorf("languages = %v", r.Languages)
	}
	if r.Stack != (domain.Stack{CMS: "WordPress", SEOPlugin: "Yoast SEO", BotProtection: "Cloudflare"}) {
		t.Errorf("stack = %+v", r.Stack)
	}
}

func TestReadinessAdviceMarksEachCheck(t *testing.T) {
	advice := map[domain.Check]bool{}
	for _, a := range domain.AssessReadiness(fullSite()).Advice() {
		advice[a.Check] = a.OK
	}
	want := map[domain.Check]bool{
		domain.CheckSitemapDeclared: true, domain.CheckSitemapLanguages: true, domain.CheckLLMs: true,
		domain.CheckMarkdown: true, domain.CheckOrganization: true, domain.CheckProduct: true,
		domain.CheckLanguages: true,
	}
	if len(advice) != len(want) {
		t.Fatalf("advice = %v", advice)
	}
	for check, ok := range want {
		if advice[check] != ok {
			t.Errorf("%s = %v, want %v", check, advice[check], ok)
		}
	}
	bare := domain.AssessReadiness(domain.ReadinessFiles{Robots: domain.File{Status: 404}})
	if bare.Gaps() != 7 {
		t.Errorf("a bare site has %d gaps, want every check", bare.Gaps())
	}
	blocked := fullSite()
	blocked.Robots = ok("text/plain", "User-agent: *\nDisallow: /\nSitemap: /sitemap.xml\n")
	if got := domain.AssessReadiness(blocked).Gaps(); got != len(domain.Bots()) {
		t.Errorf("a site that blocks every crawler has %d gaps, want one per crawler", got)
	}
}

func TestReadinessOfABareSite(t *testing.T) {
	r := domain.AssessReadiness(domain.ReadinessFiles{
		Robots:   domain.File{Status: 404, ContentType: "text/html", Body: "<html>Not found</html>"},
		LLMs:     ok("text/html", "<!DOCTYPE html><html><body>Page not found</body></html>"),
		LLMsFull: domain.File{Status: 404},
		Home:     domain.File{Status: 200, ContentType: "text/html", Body: "<html><body>Hi</body></html>"},
		// A home page twin that is the HTML page again is no Markdown version.
		HomeMarkdown: ok("text/html", "<html></html>"),
	})
	if r.Robots != domain.RobotsMissing || len(r.Bots) != len(domain.Bots()) || r.Bots[0].Access != domain.AccessAllowed {
		t.Errorf("without robots.txt every crawler is allowed: %s %+v", r.Robots, r.Bots)
	}
	if r.Sitemap.Declared || r.Sitemap.Files != 0 || len(r.Sitemap.Languages) != 0 || r.LLMs.Present || r.LLMsFull || r.Markdown {
		t.Errorf("readiness = %+v", r)
	}
	if !r.Home || len(r.StructuredData) != 0 || len(r.Languages) != 0 || r.Stack != (domain.Stack{}) {
		t.Errorf("home = %v %v %v %+v", r.Home, r.StructuredData, r.Languages, r.Stack)
	}
}

func TestReadinessOfASiteBehindBotProtection(t *testing.T) {
	r := domain.AssessReadiness(domain.ReadinessFiles{
		Robots: domain.File{Status: 403, Header: map[string]string{"cf-mitigated": "challenge"}},
		Home:   domain.File{Status: 403, ContentType: "text/html", Body: "<html>Just a moment...</html>", Header: map[string]string{"cf-mitigated": "challenge"}},
	})
	if r.Robots != domain.RobotsUnreachable || len(r.Bots) != 0 {
		t.Errorf("robots.txt the site would not serve is unreachable, not read: %s %v", r.Robots, r.Bots)
	}
	if r.Home || r.Stack.BotProtection != "Cloudflare" {
		t.Errorf("home %v, stack %+v", r.Home, r.Stack)
	}
	// Without robots.txt read, its crawler checks are not counted as gaps.
	if r.Gaps() != 7 {
		t.Errorf("gaps = %d", r.Gaps())
	}
	if unreachable := domain.AssessReadiness(domain.ReadinessFiles{Robots: domain.File{Status: 503}}); unreachable.Robots != domain.RobotsUnreachable {
		t.Errorf("a robots.txt that fails is unreachable, got %s", unreachable.Robots)
	}
}

func TestReadinessTellsTheToolsBehindASite(t *testing.T) {
	cases := []struct {
		name  string
		home  domain.File
		stack domain.Stack
	}{
		{"typo3 by generator", domain.File{Status: 200, Body: `<meta name="generator" content="TYPO3 CMS">`}, domain.Stack{CMS: "TYPO3"}},
		{"wordpress by its paths, rank math", domain.File{Status: 200, Body: `<link href="/wp-content/themes/x.css"><!-- Search Engine Optimization by Rank Math -->`}, domain.Stack{CMS: "WordPress", SEOPlugin: "Rank Math"}},
		{"aioseo", domain.File{Status: 200, Body: `<!-- All in One SEO 4.6 - aioseo.com -->`}, domain.Stack{SEOPlugin: "All in One SEO"}},
		{"seopress", domain.File{Status: 200, Body: `<meta name="generator" content="SEOPress">`}, domain.Stack{SEOPlugin: "SEOPress"}},
		{"drupal by header", domain.File{Status: 200, Header: map[string]string{"x-generator": "Drupal 10 (https://www.drupal.org)"}}, domain.Stack{CMS: "Drupal"}},
		{"drupal cache", domain.File{Status: 200, Header: map[string]string{"x-drupal-cache": "HIT"}}, domain.Stack{CMS: "Drupal"}},
		{"shopify", domain.File{Status: 200, Header: map[string]string{"x-shopify-stage": "production"}}, domain.Stack{CMS: "Shopify"}},
		{"wix", domain.File{Status: 200, Header: map[string]string{"x-wix-request-id": "1"}}, domain.Stack{CMS: "Wix"}},
		{"joomla", domain.File{Status: 200, Body: `<meta name="generator" content="Joomla! - Open Source Content Management">`}, domain.Stack{CMS: "Joomla"}},
		{"webflow", domain.File{Status: 200, Body: `<meta content="Webflow" name="generator">`}, domain.Stack{CMS: "Webflow"}},
		{"another generator", domain.File{Status: 200, Body: `<meta name="generator" content="Contao Open Source CMS">`}, domain.Stack{CMS: "Contao Open Source CMS"}},
		{"a long generator", domain.File{Status: 200, Body: `<meta name="generator" content="` + strings.Repeat("x", 70) + `">`}, domain.Stack{CMS: strings.Repeat("x", 60)}},
		{"akamai", domain.File{Status: 200, Header: map[string]string{"server": "AkamaiGHost"}}, domain.Stack{BotProtection: "Akamai"}},
		{"sucuri", domain.File{Status: 200, Header: map[string]string{"x-sucuri-id": "1"}}, domain.Stack{BotProtection: "Sucuri"}},
		{"imperva", domain.File{Status: 200, Header: map[string]string{"x-iinfo": "1"}}, domain.Stack{BotProtection: "Imperva"}},
		{"datadome", domain.File{Status: 200, Header: map[string]string{"x-datadome": "protected"}}, domain.Stack{BotProtection: "DataDome"}},
	}
	for _, c := range cases {
		if got := domain.AssessReadiness(domain.ReadinessFiles{Home: c.home}).Stack; got != c.stack {
			t.Errorf("%s: stack = %+v, want %+v", c.name, got, c.stack)
		}
	}
}

func TestReadinessFindsAMarkdownTwinOfTheHomePage(t *testing.T) {
	r := domain.AssessReadiness(domain.ReadinessFiles{HomeMarkdown: ok("text/markdown", "# Home")})
	if !r.Markdown {
		t.Error("a .md twin of the home page is a Markdown version")
	}
	plain := domain.AssessReadiness(domain.ReadinessFiles{LLMs: ok("text/plain", "- [A](https://x.test/a.md?v=1#top)")})
	if !plain.Markdown || plain.LLMs.Links != 1 {
		t.Errorf("llms.txt linking to a .md page offers Markdown versions: %+v", plain)
	}
}

func TestRawFilesAreCapped(t *testing.T) {
	big := "a" + strings.Repeat("é", domain.MaxRawFile) // two bytes each, so the limit falls inside one
	raw := domain.RawFilesOf(domain.ReadinessFiles{
		Robots:   ok("text/plain", "User-agent: *\x00\nDisallow:"),
		LLMs:     ok("text/plain", big),
		LLMsFull: domain.File{Status: 404, Body: "not found"},
	})
	if raw.Robots != "User-agent: *\nDisallow:" {
		t.Errorf("robots = %q", raw.Robots)
	}
	if len(raw.LLMs) > domain.MaxRawFile || !strings.HasPrefix(big, raw.LLMs) || !raw.LLMsTruncated {
		t.Errorf("llms.txt kept %d bytes, truncated %v", len(raw.LLMs), raw.LLMsTruncated)
	}
	if raw.LLMsFull != "" || raw.RobotsTruncated {
		t.Errorf("a file the site did not serve is not kept: %+v", raw)
	}
}

func TestCompareReadiness(t *testing.T) {
	before := domain.AssessReadiness(fullSite())
	site := fullSite()
	site.Robots = ok("text/plain", "User-agent: *\nAllow: /\n\nUser-agent: ClaudeBot\nDisallow: /\n")
	site.LLMs, site.LLMsFull, site.Sitemaps = domain.File{Status: 404}, domain.File{Status: 404}, nil
	site.Home.Header = nil
	after := domain.AssessReadiness(site)
	changes := domain.CompareReadiness(&before, &after)
	got := map[string][2]string{}
	for _, c := range changes {
		got[c.Field] = [2]string{c.From, c.To}
	}
	want := map[string][2]string{
		"bot:GPTBot":       {"blocked", "allowed"},
		"bot:ClaudeBot":    {"allowed", "blocked"},
		"sitemapDeclared":  {"yes", "no"},
		"sitemapLanguages": {"nl, de, en", "none"},
		"llmsTxt":          {"yes", "no"},
		"llmsFullTxt":      {"yes", "no"},
		"markdown":         {"yes", "no"},
		"botProtection":    {"Cloudflare", "none"},
	}
	if len(got) != len(want) {
		t.Fatalf("changes = %+v", changes)
	}
	for field, fromTo := range want {
		if got[field] != fromTo {
			t.Errorf("%s = %v, want %v", field, got[field], fromTo)
		}
	}
	if domain.CompareReadiness(nil, &after) != nil || len(domain.CompareReadiness(&after, &after)) != 0 {
		t.Error("a scan without readiness, or the same readiness, compares to no change")
	}
	unread := domain.AssessReadiness(domain.ReadinessFiles{Robots: domain.File{Status: 503}})
	for _, c := range domain.CompareReadiness(&before, &unread) {
		if c.Field == "robots" && c.To != "unreachable" {
			t.Errorf("robots change = %+v", c)
		}
		if strings.HasPrefix(c.Field, "bot:") {
			t.Errorf("an unread robots.txt says nothing of a crawler: %+v", c)
		}
	}
}
