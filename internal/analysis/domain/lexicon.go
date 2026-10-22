package domain

import (
	"math"
	"slices"
	"strings"

	"github.com/JorisJonkers-dev/findability-explorer/internal/shared/fold"
)

// Lexicon is the vocabulary of metal conveyor belts the analysis judges a site's phrases by: every
// word of the terminology map, weighted by how specific it is, plus the words for the processes
// the belts run in. A phrase with none of these words is not a search a belt buyer makes.
type Lexicon struct {
	weight map[string]float64
	terms  []lexiconTerm
	// specific is the least weight of a word that tells belt types apart: a word in at most one
	// term in ten.
	specific float64
}

type lexiconTerm struct {
	term  GlossaryTerm
	words []string   // the term's distinct words, folded
	alts  [][]string // each spelling the term stands for, folded
}

// applicationWords are the processes and places metal belts run in, per language, folded. The
// map holds few application terms, so these keep a phrase like "spiral freezer" relevant.
func applicationWords() []string {
	return []string{
		// English
		"oven", "ovens", "freezer", "freezers", "freezing", "bakery", "baking", "cooling", "cooler",
		"drying", "dryer", "furnace", "furnaces", "proofing", "proofer", "hygienic", "sintering",
		"brazing", "annealing", "frying", "fryer", "pasteurising", "sterilisation", "tunnel", "tunnels",
		// German
		"ofen", "backofen", "durchlaufofen", "gefriertunnel", "spiralfroster", "kuhltunnel",
		"trocknung", "trockner", "backerei", "hartung", "hartofen", "frittieranlage", "lebensmittel",
		// Dutch
		"vriestunnel", "vriezer", "bakkerij", "koeltunnel", "droger", "frituur", "levensmiddelen",
		// French, Polish, Italian, Spanish
		"four", "fours", "surgelation", "boulangerie", "piec", "piekarnia", "forno", "forni", "horno", "hornos",
	}
}

// NewLexicon builds the lexicon of glossary.
func NewLexicon(glossary []GlossaryTerm) Lexicon {
	docs := map[string]int{}
	terms := make([]lexiconTerm, 0, len(glossary))
	for _, g := range glossary {
		t := newLexiconTerm(g)
		counted := map[string]bool{}
		for _, w := range t.words {
			if s := singular(w); !counted[s] {
				docs[s], counted[s] = docs[s]+1, true
			}
		}
		terms = append(terms, t)
	}
	// A word in at most one term in ten is specific; a map of fewer than ten terms is too small to
	// tell, so every word of it counts.
	specific := 0.0
	if len(glossary) >= specificShare {
		specific = math.Log(1 + specificShare)
	}
	return Lexicon{weight: wordWeights(docs, len(glossary)), terms: terms, specific: specific}
}

// newLexiconTerm folds a glossary term's spellings, and collects its distinct content words.
func newLexiconTerm(g GlossaryTerm) lexiconTerm {
	var alts [][]string
	var words []string
	for _, alt := range Alternatives(g.Text) {
		ws := foldedWords(alt)
		if len(ws) == 0 {
			continue
		}
		alts = append(alts, ws)
		for _, w := range ws {
			if !slices.Contains(words, w) && !isStopword(w) && len([]rune(w)) >= 3 {
				words = append(words, w)
			}
		}
	}
	return lexiconTerm{term: g, words: words, alts: alts}
}

// wordWeights weighs each word by how few of the terms it is in, out of termCount terms.
func wordWeights(docs map[string]int, termCount int) map[string]float64 {
	weight := make(map[string]float64, len(docs))
	for w, n := range docs {
		// A word in every term ("belt") weighs little; one in a single term ("eyelink") weighs most.
		weight[w] = math.Log(1 + float64(termCount)/float64(n))
	}
	// Words that number a maker's own types say nothing about the belt.
	for _, w := range []string{"typ", "type", "group", "gruppe", "serie", "series"} {
		if _, ok := weight[w]; ok {
			weight[w] = math.Log(2)
		}
	}
	for _, w := range applicationWords() {
		if _, ok := weight[singular(w)]; !ok {
			weight[singular(w)] = math.Log(1 + float64(termCount))
		}
	}
	return weight
}

func foldedWords(s string) []string {
	return strings.Fields(fold.Fold(strings.Join(Tokens(s), " ")))
}

// singular drops a plural s, so "belts" weighs as "belt".
func singular(w string) string {
	if len(w) > 3 && strings.HasSuffix(w, "s") && !strings.HasSuffix(w, "ss") {
		return w[:len(w)-1]
	}
	return w
}

func (l Lexicon) wordWeight(w string) float64 {
	if v, ok := l.weight[singular(w)]; ok {
		return v
	}
	return l.compoundWeight(w)
}

// compoundWeight is half the weight of the longest lexicon word of at least five letters inside
// w, so a German compound the map does not spell out ("Flachdrahtgliedergurt") still counts.
func (l Lexicon) compoundWeight(w string) float64 {
	best, weight := 0, 0.0
	for word, v := range l.weight {
		if n := len(word); n >= minCompoundPart && n > best && n < len(w) && strings.Contains(w, word) {
			best, weight = n, v/2
		}
	}
	return weight
}

// specificShare is the share of terms (one in this many) a specific word may be in at most.
const specificShare = 10

// minCompoundPart is the shortest lexicon word compoundWeight looks for inside another word.
const minCompoundPart = 5

// Relevant reports whether a phrase is about belts: at least half of its content words (all but
// stopwords) are words of the lexicon. "metalen transportbanden" is; "huidige arsenaal metalen"
// is not.
func (l Lexicon) Relevant(words []string) bool {
	content, known := 0, 0
	for _, w := range words {
		if isStopword(w) {
			continue
		}
		content++
		if l.wordWeight(w) > 0 {
			known++
		}
	}
	return known > 0 && 2*known >= content
}

// Classification is what the lexicon makes of a phrase.
type Classification struct {
	Tier   string
	Family string
	// Closest is the map term that shares the most specific words with the phrase.
	Closest string
	// InMap is true when the phrase is one of the map's own spellings, plural or not.
	InMap bool
}

// Classify finds the map term that shares the most specific words with the phrase and takes its
// tier and family; on a tie a term in the phrase's language, then the term with fewer words of its
// own wins. Generic words (belt, conveyor, metal) decide nothing, so a phrase of only those is
// broad, with no family.
func (l Lexicon) Classify(words []string, language string) Classification {
	best, bestScore := -1, 0.0
	for i, t := range l.terms {
		score := 0.0
		for _, w := range words {
			if weight := l.wordWeight(w); weight >= l.specific/2 && t.shares(w) {
				score += weight
			}
		}
		if score == 0 {
			continue // the term shares no specific word with the phrase
		}
		// A tenth of a word's weight per word of the term breaks ties, and a fifth for a term in
		// the phrase's own language.
		score -= 0.1 * l.specific * float64(len(t.words))
		if sameLanguage(t.term.Language, language) {
			score += 0.2 * l.specific
		}
		if score > bestScore {
			best, bestScore = i, score
		}
	}
	if best < 0 {
		return Classification{Tier: "broad", InMap: l.inMap(words)}
	}
	t := l.terms[best]
	return Classification{Tier: t.term.Tier, Family: t.term.Family, Closest: t.term.Text, InMap: l.inMap(words)}
}

// shares reports whether the term has the word w, plural or not, or a word of at least
// minCompoundPart letters inside w, as "drahtgliedergurt" holds the term's "gliedergurt".
func (t lexiconTerm) shares(w string) bool {
	return slices.ContainsFunc(t.words, func(tw string) bool {
		return tw == w || tw == singular(w) || (len(tw) >= minCompoundPart && strings.Contains(w, tw))
	})
}

// sameLanguage compares primary subtags, so en-US terms count for an English page.
func sameLanguage(a, b string) bool {
	pa, _, _ := strings.Cut(a, "-")
	pb, _, _ := strings.Cut(b, "-")
	return pa != "" && pa == pb
}

func (l Lexicon) inMap(words []string) bool {
	plain := make([]string, 0, len(words))
	for _, w := range words {
		plain = append(plain, singular(w))
	}
	for _, t := range l.terms {
		for _, alt := range t.alts {
			if slices.Equal(alt, words) || slices.Equal(alt, plain) || slices.Equal(singularAll(alt), plain) {
				return true
			}
		}
	}
	return false
}

func singularAll(words []string) []string {
	out := make([]string, 0, len(words))
	for _, w := range words {
		out = append(out, singular(w))
	}
	return out
}

// BrandWords are the words that only name the site itself: the labels of its host before the
// public suffix, such as twentebelt for www.twentebelt.com. A phrase with one of them says who
// the site is, not what a buyer searches for.
func BrandWords(host string) []string {
	labels := strings.Split(strings.TrimPrefix(strings.ToLower(host), "www."), ".")
	var out []string
	for _, l := range labels[:max(len(labels)-1, 0)] {
		for _, w := range foldedWords(strings.ReplaceAll(l, "-", " ")) {
			if len(w) >= 4 {
				out = append(out, w)
			}
		}
	}
	return out
}
