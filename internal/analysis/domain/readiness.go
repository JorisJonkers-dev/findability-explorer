package domain

import (
	"cmp"
	"net/url"
	"path"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/JorisJonkers-dev/findability-explorer/internal/shared/lang"
)

// File is one file a scan asked the site for: its status (0 when it could not be fetched), type,
// body as far as it was read, and, for the home page, its response headers by lower-case name.
type File struct {
	Status      int
	ContentType string
	Body        string
	Header      map[string]string
}

// served reports whether the site served the file itself: a 200 that is not an HTML page, as a
// site answers for every address it does not have.
func (f File) served() bool { return f.Status == 200 && !f.html() }

func (f File) html() bool {
	start := strings.ToLower(strings.TrimSpace(f.Body))
	return strings.Contains(strings.ToLower(f.ContentType), "html") ||
		strings.HasPrefix(start, "<!doctype") || strings.HasPrefix(start, "<html")
}

// ReadinessFiles are the files that tell search engines and AI assistants about a site, as the
// crawler fetched them before the page crawl.
type ReadinessFiles struct {
	Robots File
	// Sitemaps are the sitemap files read: the ones robots.txt lists, or /sitemap.xml, and the
	// sitemaps an index lists.
	Sitemaps     []Sitemap
	LLMs         File
	LLMsFull     File
	Home         File
	HomeMarkdown File
}

// RobotsStatus is what became of a site's robots.txt.
type RobotsStatus string

const (
	// RobotsFound means the site serves robots.txt.
	RobotsFound RobotsStatus = "found"
	// RobotsMissing means it has none, so every crawler may read every page.
	RobotsMissing RobotsStatus = "missing"
	// RobotsUnreachable means it would not serve the file to the scan (bot protection, or an
	// error), so its crawler policy is unknown.
	RobotsUnreachable RobotsStatus = "unreachable"
)

// SitemapSummary is what a site's sitemaps list, without the addresses themselves.
type SitemapSummary struct {
	// Declared is true when robots.txt lists a sitemap.
	Declared bool
	// Files is how many sitemap files were read.
	Files int
	// Addresses is how many different addresses they list.
	Addresses int
	// Languages are the ones of Dutch, German and English the addresses cover, by their path
	// (/de/) or their hreflang.
	Languages []string
}

// LLMsFacts is what a site's llms.txt holds.
type LLMsFacts struct {
	Present bool
	Bytes   int
	Links   int
}

// Stack is the tools behind a site, as far as its headers and home page tell; empty when unknown.
type Stack struct {
	CMS           string
	SEOPlugin     string
	BotProtection string
}

// Readiness is a site's AI readiness: what it tells search engines and AI assistants about itself.
type Readiness struct {
	Robots RobotsStatus
	// Bots is each of Bots' access; empty when robots.txt could not be read.
	Bots     []BotRule
	Sitemap  SitemapSummary
	LLMs     LLMsFacts
	LLMsFull bool
	// Markdown is true when the site offers Markdown versions of its pages.
	Markdown bool
	// Home is true when the home page could be read; the structured data and languages need it.
	Home bool
	// StructuredData are the JSON-LD types on the home page, alphabetically.
	StructuredData []string
	// Languages are the language versions the home page declares (hreflang), Dutch, German and
	// English first.
	Languages []string
	Stack     Stack
	// CheckedAt is when an administrator checked the readiness again after its scan, without a
	// crawl; zero when the scan itself read it.
	CheckedAt time.Time
}

// AssessReadiness reads a site's readiness files.
func AssessReadiness(f ReadinessFiles) Readiness {
	r := Readiness{Robots: robotsStatus(f.Robots)}
	robotsBody := ""
	if r.Robots == RobotsFound {
		robotsBody = f.Robots.Body
	}
	if r.Robots != RobotsUnreachable {
		r.Bots = ReadCrawlerPolicy(robotsBody)
	}
	r.Sitemap = summariseSitemaps(len(ParseRobots(robotsBody, "*").Sitemaps()) > 0, f.Sitemaps)
	if f.LLMs.served() {
		links := llmsLinks(f.LLMs.Body)
		r.LLMs = LLMsFacts{Present: true, Bytes: len(f.LLMs.Body), Links: len(links)}
		r.Markdown = slices.ContainsFunc(links, isMarkdownLink)
	}
	r.LLMsFull = f.LLMsFull.served()
	r.Markdown = r.Markdown || f.HomeMarkdown.served()
	home := readHome(f.Home.Body)
	if f.Home.Status == 200 {
		r.Home, r.StructuredData, r.Languages = true, home.types, home.languages
	}
	r.Stack = stackOf(f.Home, home)
	return r
}

func robotsStatus(f File) RobotsStatus {
	switch {
	case f.served():
		return RobotsFound
	case f.Status == 200, f.Status == 404, f.Status == 410:
		return RobotsMissing
	default:
		return RobotsUnreachable
	}
}

func summariseSitemaps(declared bool, sitemaps []Sitemap) SitemapSummary {
	addresses := map[string]bool{}
	var languages []string
	for _, sm := range sitemaps {
		for _, p := range sm.Pages {
			addresses[p] = true
			if u, err := url.Parse(p); err == nil {
				first, _, _ := strings.Cut(strings.TrimPrefix(u.Path, "/"), "/")
				languages = append(languages, first)
			}
		}
		languages = append(languages, sm.Languages...)
	}
	return SitemapSummary{Declared: declared, Files: len(sitemaps), Addresses: len(addresses), Languages: focusLanguages(languages)}
}

// focusLanguages are the ones of Dutch, German and English among tags, in that order.
func focusLanguages(tags []string) []string {
	var out []string
	for _, l := range lang.Focus() {
		if slices.ContainsFunc(tags, func(t string) bool { return langTag(t) && lang.Primary(t) == l }) {
			out = append(out, l)
		}
	}
	return out
}

// langTag reports whether s reads as a language tag (de, de-DE, en_gb), not any path segment.
var langTagPattern = regexp.MustCompile(`^[A-Za-z]{2,3}([-_][A-Za-z0-9]{2,8})?$`)

func langTag(s string) bool { return langTagPattern.MatchString(s) }

// markdownLink is a Markdown link's target.
var markdownLink = regexp.MustCompile(`\[[^\]]*\]\(([^)\s]+)`)

func llmsLinks(body string) []string {
	var out []string
	for _, m := range markdownLink.FindAllStringSubmatch(body, -1) {
		out = append(out, m[1])
	}
	return out
}

func isMarkdownLink(target string) bool {
	u, err := url.Parse(target)
	return err == nil && strings.EqualFold(path.Ext(u.Path), ".md")
}

// MaxRawFile is the most bytes a scan keeps of each raw readiness file.
const MaxRawFile = 100_000

// RawFiles are the readiness files a scan keeps as the site served them, for administrators to
// check the classification against; each at most MaxRawFile bytes, empty when not served.
type RawFiles struct {
	Robots            string
	RobotsTruncated   bool
	LLMs              string
	LLMsTruncated     bool
	LLMsFull          string
	LLMsFullTruncated bool
}

// RawFilesOf keeps the robots.txt, llms.txt and llms-full.txt the site served.
func RawFilesOf(f ReadinessFiles) RawFiles {
	var r RawFiles
	r.Robots, r.RobotsTruncated = rawOf(f.Robots)
	r.LLMs, r.LLMsTruncated = rawOf(f.LLMs)
	r.LLMsFull, r.LLMsFullTruncated = rawOf(f.LLMsFull)
	return r
}

func rawOf(f File) (string, bool) {
	if !f.served() {
		return "", false
	}
	body := strings.ReplaceAll(strings.ToValidUTF8(f.Body, ""), "\x00", "")
	if len(body) <= MaxRawFile {
		return body, false
	}
	cut := MaxRawFile
	for cut > 0 && !utf8.RuneStart(body[cut]) {
		cut--
	}
	return body[:cut], true
}

// Check is one of the report's recommendations a site's readiness is held against, besides each
// crawler's access.
type Check string

const (
	CheckSitemapDeclared  Check = "sitemapDeclared"
	CheckSitemapLanguages Check = "sitemapLanguages"
	CheckLLMs             Check = "llmsTxt"
	CheckMarkdown         Check = "markdown"
	CheckOrganization     Check = "organization"
	CheckProduct          Check = "product"
	CheckLanguages        Check = "languages"
)

// Advice is whether a site follows one recommendation.
type Advice struct {
	Check Check
	OK    bool
}

// Advice holds the readiness against the report's recommendations: sitemaps declared in
// robots.txt and covering Dutch, German and English, llms.txt and Markdown versions, Organization
// and Product structured data, and language versions in the three languages.
func (r Readiness) Advice() []Advice {
	all := func(have []string) bool { return len(focusLanguages(have)) == len(lang.Focus()) }
	return []Advice{
		{CheckSitemapDeclared, r.Sitemap.Declared},
		{CheckSitemapLanguages, all(r.Sitemap.Languages)},
		{CheckLLMs, r.LLMs.Present},
		{CheckMarkdown, r.Markdown},
		{CheckOrganization, slices.ContainsFunc(r.StructuredData, isOrganization)},
		{CheckProduct, slices.ContainsFunc(r.StructuredData, isProduct)},
		{CheckLanguages, all(r.Languages)},
	}
}

// Gaps counts the recommendations the site does not follow, each crawler's access included.
func (r Readiness) Gaps() int {
	n := 0
	for _, b := range r.Bots {
		if !b.FollowsAdvice() {
			n++
		}
	}
	for _, a := range r.Advice() {
		if !a.OK {
			n++
		}
	}
	return n
}

func isOrganization(t string) bool {
	return slices.Contains([]string{"Organization", "Corporation", "LocalBusiness", "OnlineBusiness", "OnlineStore"}, t)
}

func isProduct(t string) bool {
	return slices.Contains([]string{"Product", "ProductGroup", "ProductModel", "IndividualProduct"}, t)
}

// ReadinessChange is one field of the readiness that differs between two scans, both values as
// they read: "yes", "blocked", "nl, de, en", "Cloudflare", "none".
type ReadinessChange struct {
	Field string
	From  string
	To    string
}

// CompareReadiness lists what changed from one scan's readiness to another's; nothing when either
// has none. A crawler's access is compared only when both scans read robots.txt.
func CompareReadiness(from, to *Readiness) []ReadinessChange {
	if from == nil || to == nil {
		return nil
	}
	a, b := from.fields(), to.fields()
	out := []ReadinessChange{}
	for _, f := range a {
		i := slices.IndexFunc(b, func(g field) bool { return g.name == f.name })
		if i >= 0 && b[i].value != f.value {
			out = append(out, ReadinessChange{Field: f.name, From: f.value, To: b[i].value})
		}
	}
	return out
}

type field struct{ name, value string }

func (r Readiness) fields() []field {
	out := []field{{"robots", string(r.Robots)}}
	for _, b := range r.Bots {
		out = append(out, field{"bot:" + b.Bot, string(b.Access)})
	}
	return append(out,
		field{string(CheckSitemapDeclared), yesNo(r.Sitemap.Declared)},
		field{string(CheckSitemapLanguages), list(r.Sitemap.Languages)},
		field{string(CheckLLMs), yesNo(r.LLMs.Present)},
		field{"llmsFullTxt", yesNo(r.LLMsFull)},
		field{string(CheckMarkdown), yesNo(r.Markdown)},
		field{string(CheckOrganization), yesNo(slices.ContainsFunc(r.StructuredData, isOrganization))},
		field{string(CheckProduct), yesNo(slices.ContainsFunc(r.StructuredData, isProduct))},
		field{string(CheckLanguages), list(r.Languages)},
		field{"cms", orNone(r.Stack.CMS)},
		field{"seoPlugin", orNone(r.Stack.SEOPlugin)},
		field{"botProtection", orNone(r.Stack.BotProtection)},
	)
}

func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

func list(items []string) string { return orNone(strings.Join(items, ", ")) }

func orNone(s string) string { return cmp.Or(s, "none") }
