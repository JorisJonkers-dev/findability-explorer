package domain

import (
	"cmp"
	"math"
	"slices"
	"strings"
	"unicode"

	"github.com/JorisJonkers-dev/findability-explorer/internal/shared/fold"
)

// Page is one crawled page as the extraction reads it.
type Page struct {
	URL         string
	Title       string
	Description string
	// Language is the page's declared language (from <html lang>), or empty.
	Language string
	Headings []string
	// Text is the visible text, one line per block of the page.
	Text string
}

// PageSummary is what an analysis keeps of a page.
type PageSummary struct {
	URL         string
	Title       string
	Description string
	Language    string
	WordCount   int
}

// Keyword is a phrase a site emphasises, scored by where and how often it appears.
type Keyword struct {
	Phrase   string
	Language string
	Score    float64
	Pages    int
}

// GlossaryTerm is a term from the terminology map, as the analysis context receives it.
type GlossaryTerm struct {
	Text     string
	Language string
	Family   string
	Tier     string
	// UsedByTribelt is true when Tribelt's own website uses the term.
	UsedByTribelt bool
}

// KnownTerm is a glossary term found on the crawled pages.
type KnownTerm struct {
	GlossaryTerm
	Occurrences int
}

const (
	// MaxKeywords is how many keywords an analysis keeps.
	MaxKeywords = 200
	// MaxKnownTerms is how many known terms an analysis keeps.
	MaxKnownTerms = 200
	// UnknownLanguage marks a page whose language could not be told.
	UnknownLanguage = "und"
	maxNgram        = 3
	// A phrase counts at most this often per field of a page, so a menu repeated on every page
	// does not drown out what the page is about.
	maxCountPerField = 3
)

// Field weights: a word in the title says more about a page than the same word in its body.
const (
	weightTitle       = 5
	weightDescription = 3
	weightHeading     = 3
	weightBody        = 1
)

// Summarise returns what an analysis keeps of p, with its language settled.
func Summarise(p Page) PageSummary {
	return PageSummary{
		URL: p.URL, Title: p.Title, Description: p.Description,
		Language: PageLanguage(p), WordCount: len(Tokens(p.Text)),
	}
}

// PageLanguage is the declared language's primary subtag ("de-DE" is "de"), or else the language
// whose stopwords are most frequent in the page, or UnknownLanguage.
func PageLanguage(p Page) string {
	if lang, _, _ := strings.Cut(strings.ToLower(strings.TrimSpace(p.Language)), "-"); lang != "" {
		return lang
	}
	return DetectLanguage(p.Title + " " + p.Description + " " + p.Text)
}

// DetectLanguage returns the language with the most stopwords in text, if at least three are found.
func DetectLanguage(text string) string {
	counts := map[string]int{}
	for _, tok := range Tokens(text) {
		for _, lang := range stopwordLanguages {
			if _, ok := stopwords[lang][tok]; ok {
				counts[lang]++
			}
		}
	}
	best, bestCount := UnknownLanguage, 2
	for _, lang := range stopwordLanguages {
		if counts[lang] > bestCount {
			best, bestCount = lang, counts[lang]
		}
	}
	return best
}

// Tokens splits text into lower-case words. A word is a run of letters and digits; a hyphen or a
// dot between two such characters stays inside the word, so "tri-flex" and "1.4301" are one word.
func Tokens(text string) []string {
	runes := []rune(strings.ToLower(text))
	var out []string
	start := -1
	for i, r := range runes {
		inWord := unicode.IsLetter(r) || unicode.IsDigit(r)
		joiner := (r == '-' || r == '.') && start >= 0 && i+1 < len(runes) &&
			(unicode.IsLetter(runes[i+1]) || unicode.IsDigit(runes[i+1]))
		switch {
		case inWord || joiner:
			if start < 0 {
				start = i
			}
		case start >= 0:
			out = append(out, string(runes[start:i]))
			start = -1
		}
	}
	if start >= 0 {
		out = append(out, string(runes[start:]))
	}
	return out
}

// Segments splits text where a phrase cannot continue: at line breaks, at sentence and clause
// ends (".", "!", "?", ";", ":" before a space or at the end) and at the separators menus and titles
// use ("|", "•", "·", " - ", " – "). A dot inside "1.4301" or "b.v" does not split.
func Segments(text string) []string {
	runes := []rune(text)
	var out []string
	start := 0
	cut := func(i, skip int) {
		if s := strings.TrimSpace(string(runes[start:i])); s != "" {
			out = append(out, s)
		}
		start = i + skip
	}
	for i, r := range runes {
		next := ' '
		if i+1 < len(runes) {
			next = runes[i+1]
		}
		prev := 'x'
		if i > 0 {
			prev = runes[i-1]
		}
		switch {
		case r == '\n' || r == '|' || r == '•' || r == '·':
			cut(i, 1)
		case strings.ContainsRune(".!?;:", r) && unicode.IsSpace(next):
			cut(i, 1)
		case (r == '-' || r == '–') && unicode.IsSpace(prev) && unicode.IsSpace(next):
			cut(i, 1)
		}
	}
	cut(len(runes), 0)
	return out
}

// phrases returns every 1- to 3-word phrase of tokens that neither starts nor ends with a
// stopword or a bare number, and is not a single word shorter than three letters.
func phrases(tokens []string) []string {
	var out []string
	for i := range tokens {
		for n := 1; n <= maxNgram && i+n <= len(tokens); n++ {
			first, last := tokens[i], tokens[i+n-1]
			if isNoise(first) || isNoise(last) || (n == 1 && len([]rune(first)) < 3) {
				continue
			}
			out = append(out, strings.Join(tokens[i:i+n], " "))
		}
	}
	return out
}

func isNoise(tok string) bool {
	if isStopword(tok) {
		return true
	}
	return strings.IndexFunc(tok, unicode.IsLetter) < 0 && !strings.Contains(tok, ".")
}

func isStopword(tok string) bool {
	for _, lang := range stopwordLanguages {
		if _, ok := stopwords[lang][tok]; ok {
			return true
		}
	}
	return false
}

// pageScores sums, per phrase of one page, the weight of each field it appears in, counting a
// phrase at most maxCountPerField times per field.
func pageScores(p Page) map[string]float64 {
	scores := map[string]float64{}
	add := func(text string, weight float64) {
		counts := map[string]int{}
		for _, segment := range Segments(text) {
			for _, ph := range phrases(Tokens(segment)) {
				if counts[ph] < maxCountPerField {
					counts[ph]++
					scores[ph] += weight
				}
			}
		}
	}
	add(p.Title, weightTitle)
	add(p.Description, weightDescription)
	for _, h := range p.Headings {
		add(h, weightHeading)
	}
	add(p.Text, weightBody)
	return scores
}

type keywordKey struct{ phrase, language string }

type keywordTally struct {
	score float64
	pages int
}

// Keywords returns the most prominent phrases across pages, highest score first. A phrase scores
// its field weight for each time it appears in a field (up to maxCountPerField), longer phrases
// count more, and a phrase found on more pages counts more.
func Keywords(pages []Page) []Keyword {
	tally := map[keywordKey]*keywordTally{}
	for _, p := range pages {
		lang := PageLanguage(p)
		for ph, s := range pageScores(p) {
			k := keywordKey{phrase: ph, language: lang}
			t, ok := tally[k]
			if !ok {
				t = &keywordTally{}
				tally[k] = t
			}
			t.score += s
			t.pages++
		}
	}
	out := make([]Keyword, 0, len(tally))
	for k, t := range tally {
		words := float64(strings.Count(k.phrase, " ") + 1)
		score := t.score * (1 + 0.5*(words-1)) * (1 + math.Log(float64(t.pages)))
		out = append(out, Keyword{Phrase: k.phrase, Language: k.language, Score: math.Round(score*10) / 10, Pages: t.pages})
	}
	slices.SortFunc(out, func(a, b Keyword) int {
		return cmp.Or(cmp.Compare(b.Score, a.Score), strings.Compare(a.Phrase, b.Phrase), strings.Compare(a.Language, b.Language))
	})
	return out[:min(len(out), MaxKeywords)]
}

// FindKnownTerms counts how often each glossary term appears, as whole words after folding,
// across the titles, descriptions, headings and text of pages. A term written as alternatives
// ("eye-link belt / eyelink belt", "metal conveyor belt(s)") counts every alternative, and each
// alternative also matches with a plural s. Terms that do not appear are left out; the rest come
// most frequent first.
func FindKnownTerms(pages []Page, glossary []GlossaryTerm) []KnownTerm {
	var corpus strings.Builder
	for _, p := range pages {
		for _, part := range append([]string{p.Title, p.Description, p.Text}, p.Headings...) {
			for _, segment := range Segments(part) {
				corpus.WriteString(" " + fold.Fold(strings.Join(Tokens(segment), " ")) + " ")
			}
		}
	}
	text := corpus.String()
	var out []KnownTerm
	for _, g := range glossary {
		n := 0
		for _, alt := range Alternatives(g.Text) {
			needle := fold.Fold(strings.Join(Tokens(alt), " "))
			if needle != "" {
				n += strings.Count(text, " "+needle+" ") + strings.Count(text, " "+needle+"s ")
			}
		}
		if n > 0 {
			out = append(out, KnownTerm{GlossaryTerm: g, Occurrences: n})
		}
	}
	slices.SortStableFunc(out, func(a, b KnownTerm) int { return cmp.Compare(b.Occurrences, a.Occurrences) })
	return out[:min(len(out), MaxKnownTerms)]
}

// Alternatives splits a glossary entry into the spellings it stands for: "a / b" is a and b, and
// every parenthesised remark is dropped, so "belt(s)" is belt (FindKnownTerms adds the plural).
func Alternatives(term string) []string {
	var out []string
	add := func(s string) {
		if s = strings.Join(strings.Fields(s), " "); s != "" && !slices.Contains(out, s) {
			out = append(out, s)
		}
	}
	for _, part := range strings.Split(term, " / ") {
		add(dropRemarks(part))
	}
	return out
}

// dropRemarks removes every "(…)" from s.
func dropRemarks(s string) string {
	var b strings.Builder
	depth := 0
	for _, r := range s {
		switch {
		case r == '(':
			depth++
		case r == ')' && depth > 0:
			depth--
		case depth == 0:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// SiteText is the text the language model reads: each page's address, title, headings and text,
// cut off at limit characters. It also reports whether anything was cut.
func SiteText(pages []Page, limit int) (string, bool) {
	var b strings.Builder
	for _, p := range pages {
		b.WriteString("PAGE " + p.URL + "\nTITLE " + p.Title + "\n")
		for _, h := range p.Headings {
			b.WriteString("HEADING " + h + "\n")
		}
		b.WriteString(p.Text + "\n\n")
	}
	runes := []rune(b.String())
	if len(runes) <= limit {
		return string(runes), false
	}
	return string(runes[:limit]), true
}
