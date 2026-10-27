package domain

import (
	"net/url"
	"slices"

	"github.com/JorisJonkers-dev/findability-explorer/internal/shared/spelling"
)

// Field weights for telling what a page is about: its title says most, then its address, then its
// first heading. Other headings and the body are left out; menus, footers and "our other belts"
// blocks in them name every family.
const (
	familyWeightTitle   = 3
	familyWeightURL     = 2
	familyWeightHeading = 1
	// familyLead is how many times the best family's score must be the runner-up's.
	familyLead = 1.5
	// familyShare is the share of the map terms holding a word that must be of one family for
	// the word to point at that family; "conveyor", in terms of every family, points nowhere.
	familyShare = 0.6
)

// tribeltFamilyIDs are the families a page can be about.
func tribeltFamilyIDs() []string {
	return []string{"eyelink", "open_spiral", "closed_spiral", "tri_flex_turn_curve", "other"}
}

// PageFamily is the Tribelt family a page is about: the family the specific words of the page's
// title, address and first heading point at, when it leads the next family clearly. A word points
// at a family when most map terms holding it are of that family. It is empty when no family leads.
func (l Lexicon) PageFamily(p Page) string {
	scores := map[string]float64{}
	l.addFamilyScores(scores, spelling.Words(p.Title), familyWeightTitle)
	if u, err := url.Parse(p.URL); err == nil {
		l.addFamilyScores(scores, linkWords(u, ""), familyWeightURL)
	}
	if len(p.Headings) > 0 {
		l.addFamilyScores(scores, spelling.Words(p.Headings[0]), familyWeightHeading)
	}
	return leader(scores)
}

// addFamilyScores adds, for each distinct specific word that points at a family, its weight
// times weight to that family's score.
func (l Lexicon) addFamilyScores(scores map[string]float64, words []string, weight float64) {
	for _, w := range distinct(words) {
		if v := l.wordWeight(w); v >= l.specific/2 {
			if family, ok := l.familyOf(w); ok {
				scores[family] += v * weight
			}
		}
	}
}

// leader is the family with the highest score when it leads the runner-up by familyLead, or empty.
func leader(scores map[string]float64) string {
	best, top, second := "", 0.0, 0.0
	for _, family := range tribeltFamilyIDs() {
		switch s := scores[family]; {
		case s > top:
			best, top, second = family, s, top
		case s > second:
			second = s
		}
	}
	if top == 0 || top < familyLead*second {
		return ""
	}
	return best
}

// familyOf is the Tribelt family a word points at: the family of the product terms among at least
// familyShare of all the map terms that hold w. Only product terms name which belt a word is
// about; a word that application, problem, specification or broad terms also hold ("baking" of
// "baking band", "tension", "open" of "open area", "conveyor") says what a belt does, how it is
// measured or that it is a belt, and points at no family.
func (l Lexicon) familyOf(w string) (string, bool) {
	counts := map[string]int{}
	total := 0
	for _, t := range l.terms {
		if t.shares(w) {
			total++
			if t.term.Tier == "product" {
				counts[t.term.Family]++
			}
		}
	}
	best, n := "", 0
	for _, family := range tribeltFamilyIDs() {
		if counts[family] > n {
			best, n = family, counts[family]
		}
	}
	return best, total > 0 && float64(n) >= familyShare*float64(total)
}

func distinct(words []string) []string {
	var out []string
	for _, w := range words {
		if !slices.Contains(out, w) {
			out = append(out, w)
		}
	}
	return out
}
