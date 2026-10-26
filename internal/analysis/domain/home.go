package domain

import (
	"cmp"
	"encoding/json"
	"slices"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"github.com/JorisJonkers-dev/findability-explorer/internal/shared/lang"
)

// homeFacts is what a home page's markup tells of the site: its JSON-LD types, the language
// versions it declares, and its generator tags.
type homeFacts struct {
	types      []string
	languages  []string
	generators []string
	// markup is the page in lower case, for the traces tools leave in it.
	markup string
}

// readHome reads a home page's markup; it never fails, reading what it can.
func readHome(body string) homeFacts {
	r := homeReader{facts: homeFacts{markup: strings.ToLower(body)}, types: map[string]bool{}}
	z := html.NewTokenizer(strings.NewReader(body))
	for tt := z.Next(); tt != html.ErrorToken; tt = z.Next() {
		if tt == html.StartTagToken || tt == html.SelfClosingTagToken {
			r.element(z, z.Token())
		}
	}
	for t := range r.types {
		r.facts.types = append(r.facts.types, t)
	}
	slices.Sort(r.facts.types)
	r.facts.languages = declaredLanguages(r.hreflangs)
	return r.facts
}

// homeReader collects a home page's facts, element by element.
type homeReader struct {
	facts     homeFacts
	types     map[string]bool
	hreflangs []string
}

func (r *homeReader) element(z *html.Tokenizer, tok html.Token) {
	switch tok.DataAtom { //nolint:exhaustive // only these elements say anything of readiness
	case atom.Script:
		if strings.EqualFold(attrOf(tok, "type"), "application/ld+json") && z.Next() == html.TextToken {
			jsonLDTypes(z.Token().Data, r.types)
		}
	case atom.Link:
		if strings.EqualFold(attrOf(tok, "rel"), "alternate") && attrOf(tok, "hreflang") != "" {
			r.hreflangs = append(r.hreflangs, attrOf(tok, "hreflang"))
		}
	case atom.Meta:
		if strings.EqualFold(attrOf(tok, "name"), "generator") {
			r.facts.generators = append(r.facts.generators, strings.TrimSpace(attrOf(tok, "content")))
		}
	}
}

func attrOf(t html.Token, key string) string {
	for _, a := range t.Attr {
		if strings.EqualFold(a.Key, key) {
			return a.Val
		}
	}
	return ""
}

// jsonLDTypes adds every @type in a JSON-LD block to types, nested ones and @graph's included.
func jsonLDTypes(raw string, types map[string]bool) {
	var doc any
	if json.Unmarshal([]byte(raw), &doc) == nil {
		walkJSONLD(doc, types)
	}
}

func walkJSONLD(v any, types map[string]bool) {
	switch v := v.(type) {
	case []any:
		for _, item := range v {
			walkJSONLD(item, types)
		}
	case map[string]any:
		for _, t := range typeNames(v["@type"]) {
			types[t] = true
		}
		for _, item := range v {
			walkJSONLD(item, types)
		}
	}
}

// typeNames reads @type, a name or a list of names.
func typeNames(v any) []string {
	switch v := v.(type) {
	case string:
		return []string{v}
	case []any:
		var out []string
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

// declaredLanguages are the primary languages of hreflang values, once each, Dutch, German and
// English first, then the others alphabetically; x-default is no language.
func declaredLanguages(hreflangs []string) []string {
	var out []string
	for _, h := range hreflangs {
		if p := lang.Primary(h); langTag(h) && !slices.Contains(out, p) {
			out = append(out, p)
		}
	}
	rank := func(l string) int {
		if i := slices.Index(lang.Focus(), l); i >= 0 {
			return i
		}
		return len(lang.Focus())
	}
	slices.SortFunc(out, func(a, b string) int { return cmp.Or(cmp.Compare(rank(a), rank(b)), strings.Compare(a, b)) })
	return out
}

// tool is a tool a site may run on, and how its traces show: in a generator tag, a header, or the
// markup.
type tool struct {
	name      string
	generator string
	headers   []string
	markup    []string
}

func cmsTools() []tool {
	return []tool{
		{name: "WordPress", generator: "wordpress", markup: []string{"/wp-content/", "/wp-includes/"}},
		{name: "TYPO3", generator: "typo3"},
		{name: "Drupal", generator: "drupal", headers: []string{"x-drupal-cache", "x-drupal-dynamic-cache"}},
		{name: "Joomla", generator: "joomla"},
		{name: "Webflow", generator: "webflow"},
		{name: "Wix", generator: "wix", headers: []string{"x-wix-request-id"}},
		{name: "Shopify", generator: "shopify", headers: []string{"x-shopify-stage"}},
		{name: "Squarespace", generator: "squarespace"},
		{name: "HubSpot", generator: "hubspot"},
		{name: "Magento", generator: "magento"},
	}
}

func seoTools() []tool {
	return []tool{
		{name: "Yoast SEO", generator: "yoast", markup: []string{"yoast seo", "yoast-schema-graph"}},
		{name: "Rank Math", generator: "rank math", markup: []string{"rank math", "rank-math"}},
		{name: "All in One SEO", generator: "all in one seo", markup: []string{"all in one seo", "aioseo"}},
		{name: "SEOPress", generator: "seopress", markup: []string{"seopress"}},
		{name: "The SEO Framework", generator: "the seo framework", markup: []string{"the seo framework"}},
	}
}

func protectionTools() []tool {
	return []tool{
		{name: "Cloudflare", headers: []string{"cf-ray", "cf-mitigated", "server=cloudflare"}},
		{name: "Akamai", headers: []string{"server=akamai", "x-akamai-transformed"}},
		{name: "Sucuri", headers: []string{"x-sucuri-id", "server=sucuri"}},
		{name: "Imperva", headers: []string{"x-iinfo", "x-cdn=imperva"}},
		{name: "DataDome", headers: []string{"x-datadome", "server=datadome"}},
	}
}

// stackOf tells the site's CMS, SEO plugin and bot protection from the home page's headers and
// markup. A generator tag no known CMS matches still names the CMS, unless an SEO plugin set it.
func stackOf(f File, h homeFacts) Stack {
	generators := slices.Clone(h.generators)
	if g := f.Header["x-generator"]; g != "" {
		generators = append(generators, g)
	}
	s := Stack{
		CMS:           detect(cmsTools(), f.Header, generators, h.markup),
		SEOPlugin:     detect(seoTools(), f.Header, generators, h.markup),
		BotProtection: detect(protectionTools(), f.Header, nil, ""),
	}
	if s.CMS == "" {
		for _, g := range generators {
			if detect(seoTools(), nil, []string{g}, "") == "" && g != "" {
				return Stack{CMS: clipText(g, 60), SEOPlugin: s.SEOPlugin, BotProtection: s.BotProtection}
			}
		}
	}
	return s
}

// detect returns the first tool whose traces show, or "". A header trace is a header's name, or
// name=part for a header whose value holds part.
func detect(tools []tool, header map[string]string, generators []string, markup string) string {
	for _, t := range tools {
		if t.generator != "" && slices.ContainsFunc(generators, func(g string) bool { return strings.Contains(strings.ToLower(g), t.generator) }) {
			return t.name
		}
		if slices.ContainsFunc(t.headers, func(trace string) bool { return hasHeader(header, trace) }) {
			return t.name
		}
		if slices.ContainsFunc(t.markup, func(m string) bool { return strings.Contains(markup, m) }) {
			return t.name
		}
	}
	return ""
}

func hasHeader(header map[string]string, trace string) bool {
	name, part, valued := strings.Cut(trace, "=")
	value, ok := header[name]
	return ok && (!valued || strings.Contains(strings.ToLower(value), part))
}

func clipText(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n])
	}
	return s
}
