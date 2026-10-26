package domain

import (
	"cmp"
	"errors"
	"math"
	"slices"
	"strings"

	"github.com/google/uuid"
)

// Group is how a company's search terms are grouped: one of Tribelt's belt families, other metal
// belts, broad terms that name no family, or material and specification terms.
type Group string

const (
	GroupEyelink      Group = "eyelink"
	GroupOpenSpiral   Group = "open_spiral"
	GroupClosedSpiral Group = "closed_spiral"
	GroupTriFlex      Group = "tri_flex_turn_curve"
	GroupOther        Group = "other"
	GroupBroad        Group = "broad"
	GroupSpec         Group = "spec"
)

// Groups lists every group in the order a profile shows them.
func Groups() []Group {
	return []Group{GroupEyelink, GroupOpenSpiral, GroupClosedSpiral, GroupTriFlex, GroupOther, GroupBroad, GroupSpec}
}

// GroupOf is the group a keyword belongs to: a specification term (tier spec, or a word for every
// family's material) is spec, a term of one family that family's group, and anything else broad.
func GroupOf(c Classification) Group {
	switch {
	case c.Tier == "spec" || c.Family == "spec_all":
		return GroupSpec
	case slices.Contains(tribeltFamilyIDs(), c.Family):
		return Group(c.Family)
	default:
		return GroupBroad
	}
}

// PhraseKey identifies a phrase in one language across scans.
type PhraseKey struct {
	Phrase   string
	Language string
}

func keyOf(k Keyword) PhraseKey { return PhraseKey{Phrase: k.Phrase, Language: k.Language} }

// ProfileTerm is a search term of a company's latest finished scan, and the number of the
// company's first finished scan that found it.
type ProfileTerm struct {
	Keyword
	FirstScan int
}

// GroupTerms are one group's terms, highest score first.
type GroupTerms struct {
	Group Group
	Terms []ProfileTerm
}

// Profile is a company's search terms from its latest finished scan.
type Profile struct {
	// Scan is the latest finished scan, without results; nil when no scan finished.
	Scan          *Analysis
	FinishedScans int
	// Languages are the terms' languages, most terms first.
	Languages []string
	// Groups holds every group in Groups order; empty without a finished scan.
	Groups []GroupTerms
}

// Finished returns the analyses that are done, in the order given.
func Finished(scans []Analysis) []Analysis {
	return slices.DeleteFunc(slices.Clone(scans), func(a Analysis) bool { return a.Status != StatusDone })
}

// BuildProfile groups latest's keywords and names, for each, the first scan that found it.
// A phrase firstScans does not know was first found by latest itself.
func BuildProfile(latest Analysis, finishedScans int, firstScans map[PhraseKey]int) Profile {
	byGroup := map[Group][]ProfileTerm{}
	perLanguage := map[string]int{}
	for _, k := range latest.Keywords {
		first, ok := firstScans[keyOf(k)]
		if !ok {
			first = latest.Number
		}
		g := GroupOf(k.Classification)
		byGroup[g] = append(byGroup[g], ProfileTerm{Keyword: k, FirstScan: first})
		perLanguage[k.Language]++
	}
	groups := make([]GroupTerms, 0, len(Groups()))
	for _, g := range Groups() {
		terms := byGroup[g]
		slices.SortStableFunc(terms, func(a, b ProfileTerm) int { return cmp.Compare(b.Score, a.Score) })
		groups = append(groups, GroupTerms{Group: g, Terms: append([]ProfileTerm{}, terms...)})
	}
	summary := latest
	summary.Pages, summary.Keywords, summary.KnownTerms, summary.Equivalents = nil, nil, nil, nil
	return Profile{Scan: &summary, FinishedScans: finishedScans, Languages: byCount(perLanguage), Groups: groups}
}

// byCount returns the keys of counts, highest count first, then alphabetically.
func byCount(counts map[string]int) []string {
	out := make([]string, 0, len(counts))
	for k := range counts {
		out = append(out, k)
	}
	slices.SortFunc(out, func(a, b string) int { return cmp.Or(cmp.Compare(counts[b], counts[a]), strings.Compare(a, b)) })
	return out
}

// Thresholds for a term that moved between two scans: its score changed by at least
// movedScoreShare of the older score and movedScorePoints points, or its pages by movedPages.
const (
	movedScoreShare  = 0.25
	movedScorePoints = 5
	movedPages       = 2
)

// Measure is a term's score and page count in one scan.
type Measure struct {
	Score float64
	Pages int
}

// TermChange is a term one scan found and the other did not, or whose measure moved. From is nil
// for an added term, To for a removed one.
type TermChange struct {
	Phrase   string
	Language string
	Classification
	From *Measure
	To   *Measure
}

// ComparisonGroup is what changed in one group.
type ComparisonGroup struct {
	Group   Group
	Added   []TermChange
	Removed []TermChange
	Moved   []TermChange
}

// Comparison is two finished scans of a company compared: their search terms per group, and
// their AI readiness.
type Comparison struct {
	From   Analysis
	To     Analysis
	Groups []ComparisonGroup
	// Readiness is what changed in AI readiness; nil when either scan recorded none.
	Readiness []ReadinessChange
}

// ErrNotComparable is returned when two scans cannot be compared.
var ErrNotComparable = errors.New("a comparison needs two different finished scans of the company")

// Compare returns, per group in Groups order, the terms only to found (added), only from found
// (removed), and those whose measure moved beyond the thresholds. An added or moved term counts
// in its group in to, a removed one in its group in from.
func Compare(from, to Analysis) []ComparisonGroup {
	old := make(map[PhraseKey]Keyword, len(from.Keywords))
	for _, k := range from.Keywords {
		old[keyOf(k)] = k
	}
	byGroup := map[Group]*ComparisonGroup{}
	for _, g := range Groups() {
		byGroup[g] = &ComparisonGroup{Group: g, Added: []TermChange{}, Removed: []TermChange{}, Moved: []TermChange{}}
	}
	seen := map[PhraseKey]bool{}
	for _, k := range to.Keywords {
		seen[keyOf(k)] = true
		g := byGroup[GroupOf(k.Classification)]
		now := &Measure{Score: k.Score, Pages: k.Pages}
		before, found := old[keyOf(k)]
		switch {
		case !found:
			g.Added = append(g.Added, change(k, nil, now))
		case moved(before, k):
			g.Moved = append(g.Moved, change(k, &Measure{Score: before.Score, Pages: before.Pages}, now))
		}
	}
	for _, k := range from.Keywords {
		if !seen[keyOf(k)] {
			g := byGroup[GroupOf(k.Classification)]
			g.Removed = append(g.Removed, change(k, &Measure{Score: k.Score, Pages: k.Pages}, nil))
		}
	}
	out := make([]ComparisonGroup, 0, len(byGroup))
	for _, g := range Groups() {
		c := byGroup[g]
		slices.SortStableFunc(c.Added, func(a, b TermChange) int { return cmp.Compare(b.To.Score, a.To.Score) })
		slices.SortStableFunc(c.Removed, func(a, b TermChange) int { return cmp.Compare(b.From.Score, a.From.Score) })
		slices.SortStableFunc(c.Moved, func(a, b TermChange) int { return cmp.Compare(b.shift(), a.shift()) })
		out = append(out, *c)
	}
	return out
}

func change(k Keyword, from, to *Measure) TermChange {
	return TermChange{Phrase: k.Phrase, Language: k.Language, Classification: k.Classification, From: from, To: to}
}

func moved(before, after Keyword) bool {
	delta := math.Abs(after.Score - before.Score)
	pages := after.Pages - before.Pages
	return (delta >= movedScorePoints && delta >= movedScoreShare*before.Score) || pages >= movedPages || -pages >= movedPages
}

// shift is how far a moved term's score moved.
func (c TermChange) shift() float64 { return math.Abs(c.To.Score - c.From.Score) }

// Pick returns the scans to compare: the finished scans named by from and to, by default the
// newest finished scan as to and the finished scan before to as from. finished is newest first.
// A nil id means the default for that side. The older of the two is always returned first.
func Pick(finished []Analysis, from, to uuid.UUID) (Analysis, Analysis, error) {
	index := func(id uuid.UUID) int {
		return slices.IndexFunc(finished, func(a Analysis) bool { return a.ID == id })
	}
	newer := 0
	if to != uuid.Nil {
		newer = index(to)
	}
	older := newer + 1
	if from != uuid.Nil {
		older = index(from)
	}
	if newer < 0 || older < 0 || older >= len(finished) || newer >= len(finished) || older == newer {
		return Analysis{}, Analysis{}, ErrNotComparable
	}
	a, b := finished[older], finished[newer]
	if a.Number > b.Number {
		a, b = b, a
	}
	return a, b, nil
}
