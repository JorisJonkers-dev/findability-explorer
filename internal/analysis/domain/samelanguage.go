package domain

import (
	"slices"

	"github.com/JorisJonkers-dev/findability-explorer/internal/shared/lang"
	"github.com/JorisJonkers-dev/findability-explorer/internal/shared/spelling"
)

// Scans made before a phrase was paired only with terms in its own language (ADR-0008) were
// stored with keywords in the map, or closest to a term, through a term in another language, and
// with known terms counted on pages in another language. They are not rewritten; they are read
// by the rule instead.

// ByTheMap is the analysis as the map lex was built from judges it now: each keyword stored in no
// language takes the one its own words tell (lang.Vocabulary.Phrase; ADR-0011), and is left out
// when the scan has the phrase in that language already; each is classified again from its phrase
// and language (Lexicon.Classify), so it is in the map, and closest to a term, only in its own
// language; and only the known terms in a language of a page the scan read. Which page a known term was counted on is not stored, so one in a language the scan read
// a page in stays, even when it was counted on a page in another language as well.
func (a Analysis) ByTheMap(lex Lexicon) Analysis {
	if a.Keywords != nil {
		keywords := make([]Keyword, 0, len(a.Keywords))
		told := map[[2]string]bool{}
		for _, k := range a.Keywords {
			told[[2]string{k.Phrase, k.Language}] = lang.Told(k.Language)
		}
		for _, k := range a.Keywords {
			if !lang.Told(k.Language) {
				k.Language = lex.vocabulary.Phrase(k.Phrase)
				if told[[2]string{k.Phrase, k.Language}] {
					continue
				}
			}
			k.Classification = lex.Classify(spelling.Words(k.Phrase), k.Language)
			keywords = append(keywords, k)
		}
		a.Keywords = keywords
	}
	languages := make([]string, 0, len(a.Pages))
	for _, p := range a.Pages {
		languages = append(languages, p.Language)
	}
	if a.KnownTerms != nil {
		a.KnownTerms = slices.DeleteFunc(slices.Clone(a.KnownTerms), func(k KnownTerm) bool { return !inAny(k.Language, languages) })
		// Whether tribelt.nl uses a term is the map's word now, not the scan's: it follows
		// tribelt.nl's newest scan. A term the map no longer holds keeps what the scan stored.
		for i, k := range a.KnownTerms {
			if g, ok := lex.term(k.Text, k.Language); ok {
				a.KnownTerms[i].UsedByTribelt = g.UsedByTribelt
			}
		}
	}
	return a
}

// SightedInTheirLanguage keeps the sightings of a term in a language of a page the scan that found
// it read. A sighting stored before the same-language rule may have counted a German term on an
// English page; when the scan read no German page, that is certain, and the sighting goes.
func SightedInTheirLanguage(seen []Sighting) []Sighting {
	return slices.DeleteFunc(slices.Clone(seen), func(s Sighting) bool { return !inAny(s.Language, s.PageLanguages) })
}

// inAny reports whether language is the same as one of languages (lang.Same).
func inAny(language string, languages []string) bool {
	return slices.ContainsFunc(languages, func(l string) bool { return lang.Same(language, l) })
}
