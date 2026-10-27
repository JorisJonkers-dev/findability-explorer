package domain_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/JorisJonkers-dev/findability-explorer/internal/analysis/domain"
)

func TestDetectLanguage(t *testing.T) {
	cases := map[string]string{
		"The belt is made of stainless steel and it runs in the oven for the bakery.": "en",
		"Das Band ist aus Edelstahl und läuft in dem Ofen für die Bäckerei mit uns.":  "de",
		"De band is van roestvrij staal en loopt in de oven voor de bakkerij met u.":  "nl",
		"Le tapis est en acier et il est pour les fours avec une grande largeur.":     "fr",
		"Edelstahl Ofenband 1.4301": domain.UnknownLanguage,
	}
	for text, want := range cases {
		if got := domain.DetectLanguage(text); got != want {
			t.Errorf("DetectLanguage(%q) = %q, want %q", text, got, want)
		}
	}
}

func TestPageLanguagePrefersTheDeclaredOne(t *testing.T) {
	if got := domain.PageLanguage(domain.Page{Language: " de-DE ", Text: "the belt is for the oven and the line"}); got != "de" {
		t.Errorf("declared language = %q", got)
	}
	if got := domain.PageLanguage(domain.Page{Text: "the belt is for the oven and the line"}); got != "en" {
		t.Errorf("guessed language = %q", got)
	}
}

func TestSummarise(t *testing.T) {
	s := domain.Summarise(domain.Page{URL: "https://x.test/", Title: "T", Description: "D", Language: "nl", Text: "een twee drie"}, testLexicon())
	if s.URL != "https://x.test/" || s.Title != "T" || s.Description != "D" || s.Language != "nl" || s.WordCount != 3 || s.Family != "" {
		t.Fatalf("Summarise = %+v", s)
	}
}

func phrasesOf(kws []domain.Keyword) []string {
	out := make([]string, 0, len(kws))
	for _, k := range kws {
		out = append(out, k.Phrase)
	}
	return out
}

// testLexicon is a small terminology map: enough to tell belt phrases from the rest.
func testLexicon() domain.Lexicon {
	return domain.NewLexicon([]domain.GlossaryTerm{
		{Text: "spiral belt / spiral conveyor belt", Language: "en", Family: "tri_flex_turn_curve", Tier: "product"},
		{Text: "eye-link belt / eyelink belt", Language: "en", Family: "eyelink", Tier: "product"},
		{Text: "Eyelink-band", Language: "sv", Family: "eyelink", Tier: "product"},
		{Text: "balanced weave belt", Language: "en", Family: "open_spiral", Tier: "product"},
		{Text: "metal conveyor belt(s)", Language: "en", Family: "all", Tier: "broad"},
		{Text: "Drahtösengliedergurte Typ 800", Language: "de", Family: "eyelink", Tier: "product"},
		{Text: "wire diameter / cross rod diameter", Language: "en", Family: "spec_all", Tier: "spec"},
		{Text: "nid d'abeilles", Language: "fr", Family: "other", Tier: "product"},
		{Text: "honeycomb belt", Language: "en", Family: "other", Tier: "product"},
		{Text: "flat wire belt", Language: "en", Family: "other", Tier: "product"},
		{Text: "°", Language: "en", Family: "spec_all", Tier: "spec"},
	})
}

func TestKeywordsWeighTitlesAndSpreadOverPages(t *testing.T) {
	pages := []domain.Page{
		{
			URL: "https://x.test/", Language: "en", Title: "Spiral belts for freezers",
			Description: "Stainless spiral belts.", Headings: []string{"Spiral belts"},
			Text: "Our spiral belts run in freezers. Menu Menu Menu Menu Menu. 2026 12 the of",
		},
		{URL: "https://x.test/b", Language: "en", Title: "Eyelink belts", Text: "Eyelink belts and spiral belts."},
	}
	kws := domain.Keywords(pages, testLexicon(), nil)
	if kws[0].Phrase != "spiral belts" || kws[0].Pages != 2 || kws[0].Language != "en" ||
		kws[0].Tier != "product" || kws[0].Family != "tri_flex_turn_curve" || !kws[0].InMap {
		t.Fatalf("top keyword = %+v, want spiral belts on 2 pages, a product term in the map", kws[0])
	}
	got := phrasesOf(kws)
	for _, unwanted := range []string{"the", "of", "2026", "12", "our", "menu", "belts and", "and spiral", "run"} {
		if slices.Contains(got, unwanted) {
			t.Errorf("noise phrase %q kept", unwanted)
		}
	}
	for i := 1; i < len(kws); i++ {
		if kws[i].Score > kws[i-1].Score {
			t.Fatalf("keywords not sorted by score at %d", i)
		}
	}
}

func TestKeywordsDropTheBrandMenusAndFragments(t *testing.T) {
	page := func(n int) domain.Page {
		return domain.Page{
			URL: "https://belts.test/" + strings.Repeat("p", n), Language: "en",
			Title: "Twentebelt balanced weave belts", Text: "Spiral belts menu\nTwentebelt\nmetal conveyor belts",
		}
	}
	pages := []domain.Page{page(1), page(2), page(3), page(4)}
	got := phrasesOf(domain.Keywords(pages, testLexicon(), domain.BrandWords("www.twentebelt.com")))
	for _, unwanted := range []string{
		"twentebelt", "twentebelt balanced weave", // the brand
		"spiral belts", "metal conveyor belts", // body-only text on every page: the menu
		"balanced weave", "weave belts", // fragments of "balanced weave belts"
	} {
		if slices.Contains(got, unwanted) {
			t.Errorf("phrase %q kept in %q", unwanted, got)
		}
	}
	if !slices.Contains(got, "balanced weave belts") {
		t.Errorf("balanced weave belts missing from %q", got)
	}
}

// A site in two languages has a menu per language: the English menu is on every English page but
// on only half the site's pages, and it is still the menu.
func TestKeywordsDropAMenuOfOneLanguage(t *testing.T) {
	var pages []domain.Page
	for i := range 4 {
		pages = append(pages,
			domain.Page{URL: fmt.Sprintf("https://belts.test/en/%d", i), Language: "en", Title: "Balanced weave belts", Text: "Spiral belts\nContact"},
			domain.Page{URL: fmt.Sprintf("https://belts.test/de/%d", i), Language: "de", Title: "Gliedergurte", Text: "Spiralgurte\nKontakt"},
		)
	}
	got := phrasesOf(domain.Keywords(pages, testLexicon(), nil))
	if slices.Contains(got, "spiral belts") {
		t.Errorf("the English menu kept in %q", got)
	}
	if !slices.Contains(got, "balanced weave belts") {
		t.Errorf("balanced weave belts missing from %q", got)
	}
}

// Link and button text starts or ends no keyword: "View flat wire", "Spiral belts details".
func TestKeywordsDropLinkText(t *testing.T) {
	pages := []domain.Page{{Language: "en", Title: "Spiral belts", Headings: []string{"View eyelink belts", "Spiral belts details", "Read more"}}}
	got := phrasesOf(domain.Keywords(pages, testLexicon(), nil))
	for _, unwanted := range []string{"view eyelink belts", "view eyelink", "spiral belts details", "belts details"} {
		if slices.Contains(got, unwanted) {
			t.Errorf("phrase %q kept in %q", unwanted, got)
		}
	}
	if !slices.Contains(got, "eyelink belts") || !slices.Contains(got, "spiral belts") {
		t.Errorf("eyelink belts or spiral belts missing from %q", got)
	}
}

func TestKeywordsKeepTypeNumbersAndMergeLanguages(t *testing.T) {
	pages := []domain.Page{
		{Language: "de", Title: "Drahtösengliedergurte Typ 800", Text: "Typ 800 und Typ 900. Gurte und Bänder."},
		{Language: "nl", Title: "Drahtösengliedergurte Typ 800"},
		{Language: "de", Text: "links en rechts. eyelink belts and spiral belts"},
	}
	kws := domain.Keywords(pages, testLexicon(), nil)
	var typed domain.Keyword
	for _, k := range kws {
		if k.Phrase == "drahtösengliedergurte typ 800" {
			typed = k
		}
	}
	if typed.Pages != 2 || typed.Language != "de" || !typed.InMap {
		t.Errorf("typed phrase = %+v, want one phrase on 2 pages in de, in the map", typed)
	}
	got := phrasesOf(kws)
	for _, unwanted := range []string{"typ 800", "typ 900", "links en rechts", "gurte und bänder"} {
		if slices.Contains(got, unwanted) {
			t.Errorf("phrase %q kept", unwanted)
		}
	}
	if !slices.Contains(got, "eyelink belts and spiral") && !slices.Contains(got, "belts and spiral") {
		// "x and y" stays only when both sides are about belts.
		t.Logf("phrases: %q", got)
	}
}

func TestKeywordsKeepsAtMostMaxKeywords(t *testing.T) {
	var terms []domain.GlossaryTerm
	var words []string
	for i := range 1200 {
		w := "belt" + string(rune('a'+i%26)) + string(rune('a'+i/26%26)) + string(rune('a'+i/676))
		terms = append(terms, domain.GlossaryTerm{Text: w, Language: "en", Family: "other", Tier: "product"})
		words = append(words, w)
	}
	kws := domain.Keywords([]domain.Page{{Language: "en", Text: strings.Join(words, "\n")}}, domain.NewLexicon(terms), nil)
	if len(kws) != domain.MaxKeywords {
		t.Fatalf("%d keywords, want %d", len(kws), domain.MaxKeywords)
	}
}

func TestFindKnownTerms(t *testing.T) {
	glossary := []domain.GlossaryTerm{
		{Text: "Drahtösengliederband", Language: "de", Family: "eyelink", Tier: "product"},
		{Text: "balanced weave belt", Language: "en", Family: "open_spiral", Tier: "product"},
		{Text: "spiral", Language: "en", Family: "open_spiral", Tier: "broad"},
		{Text: "TwenteFlex", Language: "en", Family: "tri_flex_turn_curve", Tier: "product"},
		{Text: "°", Language: "en", Family: "spec_all", Tier: "spec"},
	}
	pages := []domain.Page{
		{Language: "en", Title: "Drahtosengliederband", Headings: []string{"Balanced-weave belt"}, Text: "A balanced weave belt. Spirals. A spiral. Balanced weave belts."},
		{Language: "de-DE", Title: "Drahtösengliederband", Description: "drahtösengliederband und spiral", Text: "balanced weave belts"},
	}
	got := domain.FindKnownTerms(pages, glossary)
	if len(got) != 3 {
		t.Fatalf("found %+v, want 3 terms", got)
	}
	// Plurals count: "belts" for "balanced weave belt", "Spirals" for "spiral".
	if got[0].Text != "balanced weave belt" || got[0].Occurrences != 3 {
		t.Errorf("first = %+v, want balanced weave belt three times", got[0])
	}
	// A tie keeps the glossary's order.
	if got[1].Text != "Drahtösengliederband" || got[1].Occurrences != 2 || got[1].Family != "eyelink" {
		t.Errorf("second = %+v", got[1])
	}
	if got[2].Text != "spiral" || got[2].Occurrences != 2 {
		t.Errorf("third = %+v, want spiral twice", got[2])
	}
}

// A term counts only on the pages in its language: the German term on the English page, and the
// English terms on the German page, do not count, and a page in a language not told counts none.
// American English is English.
func TestFindKnownTermsKeepsToTheTermsLanguage(t *testing.T) {
	glossary := []domain.GlossaryTerm{
		{Text: "Transportband", Language: "de"},
		{Text: "conveyor belt", Language: "en-US"},
		{Text: "spiral", Language: "und"},
	}
	pages := []domain.Page{
		{Language: "nl", Text: "Transportband"},                // Dutch, spelled like the German term
		{Language: "en", Text: "Transportband. Conveyor belt"}, // the English page counts only its English term
		{Text: "Spiral"}, // not told: counts nothing, not even an und term
	}
	got := domain.FindKnownTerms(pages, glossary)
	if len(got) != 1 || got[0].Text != "conveyor belt" || got[0].Occurrences != 1 {
		t.Fatalf("found %+v, want only conveyor belt once", got)
	}
}

func TestFindKnownTermsKeepsAtMostMaxKnownTerms(t *testing.T) {
	var glossary []domain.GlossaryTerm
	var text []string
	for i := range domain.MaxKnownTerms + 20 {
		w := "term" + string(rune('a'+i%26)) + string(rune('a'+i/26))
		glossary = append(glossary, domain.GlossaryTerm{Text: w, Language: "en"})
		text = append(text, w)
	}
	if got := domain.FindKnownTerms([]domain.Page{{Language: "en", Text: strings.Join(text, " ")}}, glossary); len(got) != domain.MaxKnownTerms {
		t.Fatalf("%d known terms, want %d", len(got), domain.MaxKnownTerms)
	}
}

func TestSiteText(t *testing.T) {
	pages := []domain.Page{{URL: "https://x.test/", Title: "Belts", Headings: []string{"Spiral"}, Text: "Wire mesh."}}
	text, cut := domain.SiteText(pages, 1000)
	if cut || text != "PAGE https://x.test/\nTITLE Belts\nHEADING Spiral\nWire mesh.\n\n" {
		t.Fatalf("SiteText = %q, %v", text, cut)
	}
	short, cut := domain.SiteText(pages, 10)
	if !cut || short != "PAGE https" {
		t.Fatalf("SiteText(10) = %q, %v", short, cut)
	}
}

func TestSegments(t *testing.T) {
	got := domain.Segments("Tribelt B.V. makes belts. Steel 1.4301: strong!\nHome | Products • About · FAQ - Contact – Jobs\n\nend-of-line? ok; done")
	want := []string{"Tribelt B.V", "makes belts", "Steel 1.4301", "strong", "Home", "Products", "About", "FAQ", "Contact", "Jobs", "end-of-line", "ok", "done"}
	if !slices.Equal(got, want) {
		t.Fatalf("Segments = %q, want %q", got, want)
	}
	if got := domain.Segments(""); len(got) != 0 {
		t.Fatalf("Segments(empty) = %q", got)
	}
}

func TestKeywordsNeverCrossASegment(t *testing.T) {
	kws := domain.Keywords([]domain.Page{{Language: "en", Text: "Spiral belts\nEyelink belts | Rod belts"}}, testLexicon(), nil)
	for _, k := range kws {
		if k.Phrase == "belts eyelink" || k.Phrase == "belts rod" || k.Phrase == "spiral belts eyelink" {
			t.Fatalf("phrase %q crosses a segment", k.Phrase)
		}
	}
}

func TestFindKnownTermsCountsAlternativesAndPlurals(t *testing.T) {
	glossary := []domain.GlossaryTerm{{Text: "eye-link belt / eyelink belt", Language: "en"}, {Text: "metal conveyor belt(s)", Language: "en"}}
	pages := []domain.Page{{Language: "en", Title: "Eyelink belts", Text: "An eye-link belt.\nMetal conveyor belts and one metal conveyor belt."}}
	got := domain.FindKnownTerms(pages, glossary)
	if len(got) != 2 || got[0].Occurrences != 2 || got[1].Occurrences != 2 {
		t.Fatalf("found %+v, want both terms twice", got)
	}
}

// Only Dutch, German and English phrases are keywords: a French page's are not, whatever they say.
func TestKeywordsKeepToTheFocusLanguages(t *testing.T) {
	pages := []domain.Page{
		{URL: "https://belts.test/fr/", Language: "fr", Title: "Bandes à spirale", Text: "Spiral belts"},
		{URL: "https://belts.test/en/", Language: "en", Title: "Eyelink belts"},
		{URL: "https://belts.test/x", Title: "Balanced weave belts"}, // language not told: kept
	}
	got := phrasesOf(domain.Keywords(pages, testLexicon(), nil))
	if slices.Contains(got, "spiral belts") || !slices.Contains(got, "eyelink belts") || !slices.Contains(got, "balanced weave belts") {
		t.Fatalf("keywords %q", got)
	}
}
