package domain

import (
	"math"
	"slices"
	"strings"

	"github.com/JorisJonkers-dev/findability-explorer/internal/shared/lang"
	"github.com/JorisJonkers-dev/findability-explorer/internal/shared/spelling"
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
	// vocabulary tells a phrase's own language from its words and the map's terms.
	vocabulary lang.Vocabulary
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
	texts := make([]lang.Text, 0, len(glossary))
	for _, g := range glossary {
		texts = append(texts, lang.Text{Text: g.Text, Language: g.Language})
		t := newLexiconTerm(g)
		counted := map[string]bool{}
		for _, w := range t.words {
			if s := spelling.Singular(w); !counted[s] {
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
	return Lexicon{weight: wordWeights(docs, len(glossary)), terms: terms, specific: specific, vocabulary: lang.NewVocabulary(texts)}
}

// newLexiconTerm folds a glossary term's spellings, and collects its distinct content words.
func newLexiconTerm(g GlossaryTerm) lexiconTerm {
	alts := spelling.Forms(g.Text)
	var words []string
	for _, ws := range alts {
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
		if _, ok := weight[spelling.Singular(w)]; !ok {
			weight[spelling.Singular(w)] = math.Log(1 + float64(termCount))
		}
	}
	return weight
}

func (l Lexicon) wordWeight(w string) float64 {
	if v, ok := l.weight[spelling.Singular(w)]; ok {
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
	// Closest is the map term in the phrase's language that shares the most specific words with
	// the phrase.
	Closest string
	// InMap is true when the phrase is one of the spellings of a map term in its language, plural
	// or not.
	InMap bool
}

// Classify finds the map term closest to the phrase and takes its tier and family.
//
// Only the terms in the phrase's language count (see lang.Same): a Dutch phrase spelled like a
// German term is not in the map, and is never closest to an English term. A phrase in a language
// not told matches no term and is broad. The words' weights and the families they point at are
// counted over the whole map, as they say how specific a word is, not which term it is.
//
// A phrase that is one of the map's own spellings takes that term's family and tier. Otherwise only the phrase's words
// that point at a family decide its family (see familyOf): a phrase of generic words, such as
// "conveyor belts", "belt width" or "tension", names no family even when the term that shares
// most of its words is of one. The closest term is the one of the family the phrase's words
// point at most clearly; or the closest of all when a word points at its family; or else the
// closest of no family (broad terms, and specification terms for a phrase with a specification
// word). Closest is the term sharing the most specific words; on a tie the term with fewer words
// of its own wins. A phrase whose only close terms are
// of a family it does not name is broad, keeping a specification, application or problem tier.
func (l Lexicon) Classify(words []string, language string) Classification {
	if i, ok := l.spelt(words, language); ok {
		// The spelling's family and tier; the closest term of them.
		spelt := l.terms[i].term
		if j := l.closest(words, language, func(t GlossaryTerm) bool { return t.Family == spelt.Family && t.Tier == spelt.Tier }); j >= 0 {
			i = j
		}
		return l.from(i, true)
	}
	if i, ok := l.ofFamily(words, language); ok {
		return l.from(i, false)
	}
	return l.withoutFamily(words, language)
}

// ofFamily is the index of the closest term of the family the phrase names, if it names one: the
// family its words point at most clearly; or the closest term's family when a word points at it,
// as "compound" does in "compound balanced weave"; or the family of a map spelling of two words
// or more inside the phrase, so "cb5 baking band" is a baking band.
func (l Lexicon) ofFamily(words []string, language string) (int, bool) {
	scores := map[string]float64{}
	l.addFamilyScores(scores, words, 1)
	if family := leader(scores); family != "" {
		if i := l.closest(words, language, func(t GlossaryTerm) bool { return t.Family == family }); i >= 0 {
			return i, true
		}
	}
	if i := l.closest(words, language, func(GlossaryTerm) bool { return true }); i >= 0 && scores[l.terms[i].term.Family] > 0 {
		return i, true
	}
	return l.containedSpelling(words, language)
}

// withoutFamily classifies a phrase that names no family by the closest term of no family. A
// phrase with a specification word ("maximum belt width") is closest to a specification term if
// one shares more; a phrase without one is closest to a broad term first, so "belt life" is
// "metal conveyor belt(s)", not "furnace belt", and never to a specification term, so "spiral"
// alone is not "spiral pitch". Failing those it is broad, closest to any term and keeping that
// term's tier unless it is a product of a family the phrase does not name.
func (l Lexicon) withoutFamily(words []string, language string) Classification {
	spec := slices.ContainsFunc(words, l.specWord)
	if !spec {
		if i := l.closest(words, language, func(t GlossaryTerm) bool { return !isTribeltFamily(t.Family) && t.Tier == "broad" }); i >= 0 {
			return l.from(i, false)
		}
	}
	if i := l.closest(words, language, func(t GlossaryTerm) bool {
		return !isTribeltFamily(t.Family) && (t.Tier != "spec" || spec)
	}); i >= 0 {
		return l.from(i, false)
	}
	i := l.closest(words, language, func(GlossaryTerm) bool { return true })
	if i < 0 {
		return Classification{Tier: "broad"}
	}
	t := l.terms[i].term
	tier := t.Tier
	if tier == "product" {
		tier = "broad"
	}
	return Classification{Tier: tier, Closest: t.Text}
}

// containedSpelling is the index of the term in language with the longest spelling of two words
// or more that the phrase holds word for word, plural or not.
func (l Lexicon) containedSpelling(words []string, language string) (int, bool) {
	plain := spelling.SingularAll(words)
	best, length := -1, 1
	for i, t := range l.terms {
		if !lang.Same(t.term.Language, language) {
			continue
		}
		for _, alt := range t.alts {
			if len(alt) > length && holds(plain, spelling.SingularAll(alt)) {
				best, length = i, len(alt)
			}
		}
	}
	return best, best >= 0
}

// holds reports whether part is a run of consecutive words of words.
func holds(words, part []string) bool {
	for i := 0; i+len(part) <= len(words); i++ {
		if slices.Equal(words[i:i+len(part)], part) {
			return true
		}
	}
	return false
}

// specWord reports whether w is a specification word: at least familyShare of the map terms that
// hold the word itself are specification terms, as for "pitch", "sprocket" and "t316". Only the
// whole word counts, so "posidrive" is not "drive".
func (l Lexicon) specWord(w string) bool {
	spec, total := 0, 0
	for _, t := range l.terms {
		if slices.Contains(t.words, w) || slices.Contains(t.words, spelling.Singular(w)) {
			total++
			if t.term.Tier == "spec" {
				spec++
			}
		}
	}
	return total > 0 && float64(spec) >= familyShare*float64(total)
}

func (l Lexicon) from(i int, inMap bool) Classification {
	t := l.terms[i].term
	return Classification{Tier: t.Tier, Family: t.Family, Closest: t.Text, InMap: inMap}
}

// closest is the index of the term in language keep accepts that shares the most specific words
// with the phrase, by the tie-breaks Classify names, or -1 when none shares a specific word.
func (l Lexicon) closest(words []string, language string, keep func(GlossaryTerm) bool) int {
	best, bestScore := -1, 0.0
	for i, t := range l.terms {
		if !lang.Same(t.term.Language, language) || !keep(t.term) {
			continue
		}
		score := 0.0
		for _, w := range words {
			if weight := l.wordWeight(w); weight >= l.specific/2 && t.shares(w) {
				score += weight
			}
		}
		if score == 0 {
			continue // the term shares no specific word with the phrase
		}
		// A tenth of a word's weight per word of the term breaks ties. Every term here is in the
		// phrase's language, which is worth a fifth, so a long term sharing one word still counts.
		score += 0.2*l.specific - 0.1*l.specific*float64(len(t.words))
		if score > bestScore {
			best, bestScore = i, score
		}
	}
	return best
}

func isTribeltFamily(family string) bool { return slices.Contains(tribeltFamilyIDs(), family) }

// shares reports whether the term has the word w, plural or not, or a word of at least
// minCompoundPart letters inside w, as "drahtgliedergurt" holds the term's "gliedergurt".
func (t lexiconTerm) shares(w string) bool {
	return slices.ContainsFunc(t.words, func(tw string) bool {
		return tw == w || tw == spelling.Singular(w) || (len(tw) >= minCompoundPart && strings.Contains(w, tw))
	})
}

// spelt is the index of the term in language the phrase is one of the spellings of, plural or
// not.
func (l Lexicon) spelt(words []string, language string) (int, bool) {
	key := spelling.Key(words)
	for i, t := range l.terms {
		if !lang.Same(t.term.Language, language) {
			continue
		}
		for _, alt := range t.alts {
			if spelling.Key(alt) == key {
				return i, true
			}
		}
	}
	return -1, false
}

// BrandWords are the words that only name the site itself: the labels of its host before the
// public suffix, such as twentebelt for www.twentebelt.com. A phrase with one of them says who
// the site is, not what a buyer searches for.
func BrandWords(host string) []string {
	labels := strings.Split(strings.TrimPrefix(strings.ToLower(host), "www."), ".")
	var out []string
	for _, l := range labels[:max(len(labels)-1, 0)] {
		for _, w := range spelling.Words(strings.ReplaceAll(l, "-", " ")) {
			if len(w) >= 4 {
				out = append(out, w)
			}
		}
	}
	return out
}
