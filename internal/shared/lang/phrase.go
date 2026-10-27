package lang

import (
	"strings"
	"unicode/utf8"

	"github.com/JorisJonkers-dev/findability-explorer/internal/shared/fold"
	"github.com/JorisJonkers-dev/findability-explorer/internal/shared/spelling"
)

// A phrase's own language: what its words say about the language it is in, apart from the page it
// was found on. A scan reads a phrase's language from its pages, and asks the phrase only when
// those pages are in no language or in several (ADR-0011).

// set is a set of languages: the focus languages, and other for every language out of focus.
type set uint8

const (
	nl set = 1 << iota
	de
	en
	other
	focus = nl | de | en
)

// bit is the set of the one language code names, other for a code out of focus.
func bit(code string) set {
	switch Primary(code) {
	case "nl":
		return nl
	case "de":
		return de
	case "en":
		return en
	default:
		return other
	}
}

// codes are the focus languages of s, in Focus order.
func (s set) codes() []string {
	var out []string
	for _, code := range Focus() {
		if s&bit(code) != 0 {
			out = append(out, code)
		}
	}
	return out
}

// Text is a term of the map as a Vocabulary learns from it: its text and its language.
type Text struct {
	Text     string
	Language string
}

// Vocabulary tells a phrase's language from its words: the word list in words.go, and the words of
// the terms of the map, each in the languages of the terms it is in.
type Vocabulary struct {
	words map[string]set
}

// NewVocabulary learns the words of terms. A word in terms of several languages is a word of each.
func NewVocabulary(terms []Text) Vocabulary {
	words := map[string]set{}
	for _, t := range terms {
		for _, form := range spelling.Forms(t.Text) {
			for _, w := range form {
				if utf8.RuneCountInString(w) >= minWord {
					words[spelling.Singular(w)] |= bit(t.Language)
				}
			}
		}
	}
	return Vocabulary{words: words}
}

// minWord is the fewest letters a word needs to tell a language: "ra" or "st" before a product
// name, or "ø" before a wire size, tells none.
const minWord = 3

// minStem is the fewest letters before a known word that make a compound of it: "drahtgurt" is a
// "gurt", "unibelt" a "belt".
const minStem = 3

// Possible are the focus languages phrase could be in by its own words, in Focus order: one when
// they tell, several when they leave it open ("transportband" is Dutch or German), all three when
// nothing in it tells (a model number), and none when it is in another language ("cinta
// transportadora"). Each word says what it can:
//
//   - letters: ä, ö, ü and ß are German, ij, ë and ï Dutch, and letters such as é, ñ, ç, ł or å
//     another language;
//   - the word itself, in the word list or among the words of the map's terms (the list wins);
//   - the known word it ends in, as compounds do ("drahtfördergurte" is a "gurt", "draadogenbanden"
//     "banden");
//   - its ending ("teilung" is German, "toepassingen" Dutch).
//
// Each language counts the words that can be in it, the last word that tells twice: in Dutch,
// German and English the last word of a phrase is the thing it names, and a borrowed word comes
// before it ("Flat-Flex Stabgeflechtsgurte" is German). The phrase is in the language that counts
// most ("oven transportband" is Dutch); when several count as many, in those of them the last word
// can be in. Another language counts like a focus one, so a phrase most of whose words are Spanish
// is in none.
func (v Vocabulary) Possible(phrase string) []string {
	var told []set
	for _, w := range words(phrase) {
		if s := v.word(w); s != 0 && s&focus != focus { // a word every language has tells nothing
			told = append(told, s)
		}
	}
	if len(told) == 0 {
		return focus.codes()
	}
	last := told[len(told)-1]
	told = append(told, last)
	languages := []set{nl, de, en, other}
	counts := make([]int, len(languages))
	for _, s := range told {
		for i, l := range languages {
			if s&l != 0 {
				counts[i]++
			}
		}
	}
	var top set
	most := 1
	for i, l := range languages {
		switch {
		case counts[i] > most:
			top, most = l, counts[i]
		case counts[i] == most:
			top |= l
		}
	}
	if narrowed := top & last; narrowed != 0 {
		top = narrowed
	}
	return top.codes()
}

// Phrase is the focus language phrase is in by its own words (see Possible), or Unknown when they
// leave it open or it is in another language.
func (v Vocabulary) Phrase(phrase string) string {
	if p := v.Possible(phrase); len(p) == 1 {
		return p[0]
	}
	return Unknown
}

// words are the words of phrase in lower case, a hyphenated word as its parts: "eye-link belts" is
// "eye", "link" and "belts".
func words(phrase string) []string {
	var out []string
	for _, tok := range spelling.Tokens(phrase) {
		for part := range strings.SplitSeq(tok, "-") {
			if part != "" {
				out = append(out, part)
			}
		}
	}
	return out
}

// word is the set of languages w can be in: by its letters, else by itself, else by the known word
// it ends in, else by its ending; empty when nothing tells. A word shorter than minWord tells only
// when it is listed: the Dutch "en" between two belt words.
func (v Vocabulary) word(w string) set {
	f := fold.Fold(w)
	if strings.ContainsAny(w, "0123456789") {
		return 0 // a model number or a material number is in every language
	}
	if utf8.RuneCountInString(w) < minWord {
		s, _ := v.known(f)
		return s
	}
	if letters := lettersOf(w); letters != 0 {
		return letters
	}
	if s, ok := v.known(f); ok {
		return s
	}
	if s, ok := v.compound(f); ok {
		return s
	}
	return ending(f)
}

// known is the set of a folded word in the word list, else among the map's words, as written or
// singular.
func (v Vocabulary) known(f string) (set, bool) {
	for _, w := range []string{f, spelling.Singular(f)} {
		if s, ok := cueWords[w]; ok {
			return s, true
		}
	}
	for _, w := range []string{f, spelling.Singular(f)} {
		if s, ok := v.words[w]; ok {
			return s, true
		}
	}
	return 0, false
}

// compound is the set of the longest known word of at least four letters that f ends in, after a
// stem of at least minStem letters: Dutch or German when that word is also theirs, as English
// writes a compound apart ("ovenband" is no English word, "oven band" is), and narrowed by the
// stem when that is a known word too ("spiraalband" is Dutch, "ofenband" German).
func (v Vocabulary) compound(f string) (set, bool) {
	stem := 0
	for i := range f {
		if stem >= minStem && utf8.RuneCountInString(f[i:]) >= 4 {
			if s, ok := v.known(f[i:]); ok {
				if s&(nl|de) != 0 {
					s &^= en
				}
				if st, _ := v.known(f[:i]); st&s != 0 {
					s &= st
				}
				return s, true
			}
		}
		stem++
	}
	return 0, false
}

// ending is the set of the longest ending of cueEndings that f has after a stem of at least
// minStem letters, or empty.
func ending(f string) set {
	stem := 0
	for i := range f {
		if s, ok := cueEndings[f[i:]]; ok && stem >= minStem {
			return s
		}
		stem++
	}
	return 0
}

// lettersOf is the set of languages w's letters tell, or empty when they tell none.
func lettersOf(w string) set {
	switch {
	case strings.ContainsAny(w, "äöüß"):
		return de
	case strings.Contains(w, "ij") || strings.ContainsAny(w, "ëï"):
		return nl
	case strings.ContainsAny(w, "àáâãåæçèéêìíîñòóôõøùúûýÿœąćęłńśźżčřšžğış"):
		return other
	default:
		return 0
	}
}
