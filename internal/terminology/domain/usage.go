package domain

import (
	"slices"
	"strings"
	"unicode"
)

// TribeltID is Tribelt's own id among the companies, as the scans' sightings name it.
const TribeltID = "tribelt"

// UsageSource is what tells which of the map's terms tribelt.nl uses.
type UsageSource string

const (
	// UsageScan is tribelt.nl's newest finished scan: the terms it found.
	UsageScan UsageSource = "scan"
	// UsageResearch is the desk research's credits, until tribelt.nl has a finished scan.
	UsageResearch UsageSource = "research"
)

// SiteUse is what the map knows of which terms the companies' sites use: the companies whose newest
// finished scan found each term, whether tribelt.nl has a finished scan at all, and the competitors
// on the list now.
type SiteUse struct {
	FoundBy        map[TermKey][]string
	TribeltScanned bool
	// Competitors are the competitors on the list now, with the names the map credits them by.
	Competitors []Competitor
}

// Source is where TribeltUses takes its answer from: the newest scan once tribelt.nl has one, the
// desk research before.
func (u SiteUse) Source() UsageSource {
	if u.TribeltScanned {
		return UsageScan
	}
	return UsageResearch
}

// TribeltUses reports whether tribelt.nl uses the term (ADR-0012): its newest finished scan found
// it, in the term's own language (ADR-0008); before any finished scan, the desk research credits
// Tribelt with it.
func (u SiteUse) TribeltUses(t Term) bool {
	if u.TribeltScanned {
		return slices.Contains(u.FoundBy[t.Key()], TribeltID)
	}
	return CreditsTribelt(t.UsedBy)
}

// CreditsTribelt is whether one of the map's credits for a term names Tribelt: "Tribelt", or one
// of its product lines, "Tribelt (Tri-Flex)".
func CreditsTribelt(usedBy []string) bool {
	return slices.ContainsFunc(usedBy, func(u string) bool { return creditedName(u) == "Tribelt" })
}

// creditedName is the company a credit names, without the product line in brackets after it.
func creditedName(credit string) string {
	name := strings.TrimRightFunc(credit, unicode.IsSpace)
	if open := strings.LastIndex(name, "("); open >= 0 && strings.HasSuffix(name, ")") && !strings.Contains(name[open+1:len(name)-1], ")") {
		name = name[:open]
	}
	if name = strings.TrimSpace(name); name == "" {
		return strings.TrimSpace(credit)
	}
	return name
}

// Competitor is a company on the competitor list now (not archived, researched or added since),
// with the name the map credits its terms to: the longest credited name whose every word begins a
// word of its registered name, the most credited first on a tie (CreditNames); empty when no
// credit fits.
type Competitor struct {
	ID        string
	ShortName string
}

// Maker is a competitor on the list that uses a term, and whether only its newest scan says so.
type Maker struct {
	ID       string
	FromScan bool
}

// Listed is a competitor on the list now, by its id and registered name.
type Listed struct {
	ID   string
	Name string
}

// CreditNames gives each competitor on the list the name the map credits its terms to, as
// Competitor says, in the order given.
func CreditNames(terms []Term, companies []Listed) []Competitor {
	credits := creditedByCount(terms)
	out := make([]Competitor, 0, len(companies))
	for _, c := range companies {
		registered := nameWords(c.Name)
		best, length := "", 0
		for _, credit := range credits {
			own := nameWords(credit)
			fits := len(own) > 0 && !slices.ContainsFunc(own, func(w string) bool {
				return !slices.ContainsFunc(registered, func(r string) bool { return strings.HasPrefix(r, w) })
			})
			if fits && len(own) > length {
				best, length = credit, len(own)
			}
		}
		out = append(out, Competitor{ID: c.ID, ShortName: best})
	}
	return out
}

// creditedByCount are the names the map credits terms to, each once, the most credited first.
func creditedByCount(terms []Term) []string {
	count := map[string]int{}
	var names []string
	for _, t := range terms {
		for _, u := range t.UsedBy {
			name := creditedName(u)
			if count[name] == 0 {
				names = append(names, name)
			}
			count[name]++
		}
	}
	slices.SortStableFunc(names, func(a, b string) int { return count[b] - count[a] })
	return names
}

// nameWords are a name's folded words, split at everything but letters and digits.
func nameWords(name string) []string {
	return strings.FieldsFunc(Fold(name), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
}

// Makers are the competitors on the list now that use a term (ADR-0012): those its credits name,
// and those whose newest finished scan found it, in the list's order. A credit to a company no
// longer on the list, or to Tribelt, makes no maker.
func (u SiteUse) Makers(t Term) []Maker {
	found := u.FoundBy[t.Key()]
	var out []Maker
	for _, c := range u.Competitors {
		credited := c.ShortName != "" && slices.ContainsFunc(t.UsedBy, func(credit string) bool { return creditedName(credit) == c.ShortName })
		if scanned := slices.Contains(found, c.ID); credited || scanned {
			out = append(out, Maker{ID: c.ID, FromScan: !credited})
		}
	}
	return out
}
