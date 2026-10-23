package domain_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/JorisJonkers-dev/findability-explorer/internal/analysis/domain"
)

func mustURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func TestLinkScorerPutsBeltPagesFirstAndNewsLast(t *testing.T) {
	s := domain.NewLinkScorer(testLexicon())
	score := func(raw, anchor string) float64 { return s.Score(mustURL(t, raw), anchor) }
	product := score("https://x.test/products/eyelink-belts/", "")
	anchor := score("https://x.test/p/123", "Eyelink belts")
	german := score("https://x.test/de/produkte/drahtosengliedergurte", "")
	escaped := score("https://x.test/de/produkte/drahth%C3%B6sengliedergurte.html", "")
	plain := score("https://x.test/about-us", "About us")
	news := score("https://x.test/news/eyelink-belts-at-the-fair", "")
	contact := score("https://x.test/contact", "Contact")
	for name, v := range map[string]float64{"anchor": anchor, "german": german, "escaped": escaped} {
		if v <= plain {
			t.Errorf("%s scores %.2f, not above a plain page's %.2f", name, v, plain)
		}
	}
	if product <= plain || plain <= news || plain <= contact {
		t.Errorf("product %.2f, anchor %.2f, plain %.2f, news %.2f, contact %.2f", product, anchor, plain, news, contact)
	}
	// A bad escape is read as written.
	if v := s.Score(&url.URL{Scheme: "https", Host: "x.test", Path: "/%zz"}, ""); v > plain {
		t.Errorf("bad escape scores %.2f", v)
	}
}

func TestFrontierPopsBestFirstThenInOrder(t *testing.T) {
	f := domain.NewFrontier("www.x.test", domain.NewLinkScorer(testLexicon()))
	start := mustURL(t, "https://x.test/")
	f.Seen(start)
	link := func(raw, text string, alt bool) domain.Link {
		return domain.Link{URL: mustURL(t, raw), Text: text, Alternate: alt}
	}
	f.Push([]domain.Link{
		link("https://x.test/", "Home", false),        // seen
		link("https://x.test/a", "", false),           // plain
		link("https://www.x.test/b#top", "", false),   // plain, same site
		link("https://x.test/news", "", false),        // off topic
		link("https://x.test/eyelink?x=1", "", false), // belt word
		link("https://x.test/eyelink", "", false),     // the same page
		link("https://x.test/fr/", "", true),          // a language version
		link("https://other.test/eyelink", "", false), // another site
		link("mailto:x@x.test", "", false),            // not a page
		link("https://x.test/file.pdf", "", false),    // not a page
		link("https://u:p@x.test/secret", "", false),  // credentials
		{URL: nil}, // nothing
	}, 3)
	var got []string
	for next, _ := f.Pop(); next != nil; next, _ = f.Pop() {
		got = append(got, next.Path)
	}
	if strings.Join(got, " ") != "/fr/ /eyelink /a /b /news" {
		t.Fatalf("popped %q", got)
	}
	if next, score := f.Pop(); next != nil || score != 0 || f.Len() != 0 {
		t.Fatal("an empty frontier popped something")
	}
}

func TestFrontierFetchedKeepsOnePerPage(t *testing.T) {
	f := domain.NewFrontier("x.test", domain.NewLinkScorer(testLexicon()))
	if !f.Fetched("https://x.test/a/") || f.Fetched("https://www.x.test/a") {
		t.Fatal("a page fetched twice under two addresses counts twice")
	}
	if !f.Fetched("http://[::1") {
		t.Fatal("an address that does not parse is not refused")
	}
	f.Push([]domain.Link{{URL: mustURL(t, "https://x.test/a")}}, 0)
	if f.Len() != 0 {
		t.Fatal("a fetched page was queued")
	}
}

func TestFrontierHoldsAtMostTwoThousand(t *testing.T) {
	f := domain.NewFrontier("x.test", domain.NewLinkScorer(testLexicon()))
	links := make([]domain.Link, 0, 2100)
	for i := range 2100 {
		links = append(links, domain.Link{URL: mustURL(t, "https://x.test/p/"+strings.Repeat("a", i%50)+string(rune('a'+i%26))+strings.Repeat("b", i/26))})
	}
	f.Push(links, 0)
	if f.Len() != 2000 {
		t.Fatalf("frontier holds %d", f.Len())
	}
}

func TestPageFamily(t *testing.T) {
	lex := testLexicon()
	cases := map[string]struct {
		page domain.Page
		want string
	}{
		"title":          {domain.Page{URL: "https://x.test/p/1", Title: "Eyelink belts - Example | Metal conveyor belts"}, "eyelink"},
		"address":        {domain.Page{URL: "https://x.test/products/balanced-weave-belts/", Title: "Belts"}, "open_spiral"},
		"first heading":  {domain.Page{URL: "https://x.test/p", Title: "Products", Headings: []string{"Honeycomb belts", "Eyelink belts"}}, "other"},
		"no lead":        {domain.Page{URL: "https://x.test/eyelink-and-honeycomb", Title: "Eyelink and honeycomb belts"}, ""},
		"generic only":   {domain.Page{URL: "https://x.test/documents/", Title: "Documents | Metal conveyor belts"}, ""},
		"a broken URL":   {domain.Page{URL: "http://[::1", Title: "Spiral belts"}, "tri_flex_turn_curve"},
		"nothing at all": {domain.Page{}, ""},
		"a clear lead":   {domain.Page{URL: "https://x.test/p", Title: "Eyelink belts", Headings: []string{"Honeycomb"}}, "eyelink"},
	}
	for name, c := range cases {
		if got := lex.PageFamily(c.page); got != c.want {
			t.Errorf("%s: PageFamily = %q, want %q", name, got, c.want)
		}
	}
}

func TestCleanSiteOfAndKey(t *testing.T) {
	if domain.Clean(nil) != nil || domain.Clean(mustURL(t, "ftp://x.test/")) != nil {
		t.Fatal("Clean kept a link that is not a page")
	}
	if c := domain.Clean(mustURL(t, "https://x.test?q=1#f")); c.String() != "https://x.test/" {
		t.Fatalf("Clean = %s", c)
	}
	if domain.SiteOf("WWW.X.test") != "x.test" || domain.Key(mustURL(t, "https://www.x.test/a/")) != "x.test/a" {
		t.Fatal("SiteOf or Key")
	}
}
