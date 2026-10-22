package domain_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/JorisJonkers-dev/findability-explorer/internal/analysis/domain"
)

func TestTokens(t *testing.T) {
	got := domain.Tokens("Tri-Flex belts in 1.4301, up to 1.150 °C! Edelstahl-Förderbänder -x. end-")
	want := []string{"tri-flex", "belts", "in", "1.4301", "up", "to", "1.150", "c", "edelstahl-förderbänder", "x", "end"}
	if !slices.Equal(got, want) {
		t.Fatalf("Tokens = %q, want %q", got, want)
	}
	if got := domain.Tokens("Ofenband"); !slices.Equal(got, []string{"ofenband"}) {
		t.Fatalf("Tokens(single word) = %q", got)
	}
}

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
	s := domain.Summarise(domain.Page{URL: "https://x.test/", Title: "T", Description: "D", Language: "nl", Text: "een twee drie"})
	if s.URL != "https://x.test/" || s.Title != "T" || s.Description != "D" || s.Language != "nl" || s.WordCount != 3 {
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

func TestKeywordsWeighTitlesAndSpreadOverPages(t *testing.T) {
	pages := []domain.Page{
		{
			URL: "https://x.test/", Language: "en", Title: "Spiral belts for freezers",
			Description: "Stainless spiral belts.", Headings: []string{"Spiral belts"},
			Text: "Our spiral belts run in freezers. Menu Menu Menu Menu Menu. 2026 12 the of",
		},
		{URL: "https://x.test/b", Language: "en", Title: "Eyelink belts", Text: "Eyelink belts and spiral belts."},
	}
	kws := domain.Keywords(pages)
	if kws[0].Phrase != "spiral belts" || kws[0].Pages != 2 || kws[0].Language != "en" {
		t.Fatalf("top keyword = %+v, want spiral belts on 2 pages", kws[0])
	}
	got := phrasesOf(kws)
	for _, unwanted := range []string{"the", "of", "2026", "12", "our", "belts and", "and spiral"} {
		if slices.Contains(got, unwanted) {
			t.Errorf("noise phrase %q kept", unwanted)
		}
	}
	var menu domain.Keyword
	for _, k := range kws {
		if k.Phrase == "menu" {
			menu = k
		}
	}
	if menu.Score > 3 {
		t.Errorf("a word repeated in the body scores %v, want at most 3", menu.Score)
	}
	for i := 1; i < len(kws); i++ {
		if kws[i].Score > kws[i-1].Score {
			t.Fatalf("keywords not sorted by score at %d", i)
		}
	}
}

func TestKeywordsKeepsAtMostMaxKeywords(t *testing.T) {
	var words []string
	for i := range 400 {
		words = append(words, "word"+strings.Repeat("x", i%26)+string(rune('a'+i%26))+string(rune('a'+i/26)))
	}
	kws := domain.Keywords([]domain.Page{{Language: "en", Text: strings.Join(words, " ")}})
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
		{Title: "Drahtosengliederband", Headings: []string{"Balanced-weave belt"}, Text: "A balanced weave belt. Spirals."},
		{Description: "drahtösengliederband und spiral", Text: "balanced weave belts"},
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

func TestFindKnownTermsKeepsAtMostMaxKnownTerms(t *testing.T) {
	var glossary []domain.GlossaryTerm
	var text []string
	for i := range domain.MaxKnownTerms + 20 {
		w := "term" + string(rune('a'+i%26)) + string(rune('a'+i/26))
		glossary = append(glossary, domain.GlossaryTerm{Text: w})
		text = append(text, w)
	}
	if got := domain.FindKnownTerms([]domain.Page{{Text: strings.Join(text, " ")}}, glossary); len(got) != domain.MaxKnownTerms {
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
	kws := domain.Keywords([]domain.Page{{Language: "en", Text: "Spiral belts\nEyelink belts | Rod belts"}})
	for _, k := range kws {
		if k.Phrase == "belts eyelink" || k.Phrase == "belts rod" || k.Phrase == "spiral belts eyelink" {
			t.Fatalf("phrase %q crosses a segment", k.Phrase)
		}
	}
}

func TestAlternatives(t *testing.T) {
	cases := map[string][]string{
		"eye-link belt / eyelink belt":         {"eye-link belt", "eyelink belt"},
		"metal conveyor belt(s)":               {"metal conveyor belt"},
		"Eye-Link OGBL (Ashworth)":             {"Eye-Link OGBL"},
		"spiraaltorens / vries- en koeltorens": {"spiraaltorens", "vries- en koeltorens"},
		"a / a / (only a remark)":              {"a"},
		"odd ) bracket":                        {"odd ) bracket"},
	}
	for in, want := range cases {
		if got := domain.Alternatives(in); !slices.Equal(got, want) {
			t.Errorf("Alternatives(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFindKnownTermsCountsAlternativesAndPlurals(t *testing.T) {
	glossary := []domain.GlossaryTerm{{Text: "eye-link belt / eyelink belt", Language: "en"}, {Text: "metal conveyor belt(s)", Language: "en"}}
	pages := []domain.Page{{Title: "Eyelink belts", Text: "An eye-link belt.\nMetal conveyor belts and one metal conveyor belt."}}
	got := domain.FindKnownTerms(pages, glossary)
	if len(got) != 2 || got[0].Occurrences != 2 || got[1].Occurrences != 2 {
		t.Fatalf("found %+v, want both terms twice", got)
	}
}
