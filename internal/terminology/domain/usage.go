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
// finished scan found each term, and whether tribelt.nl has a finished scan at all.
type SiteUse struct {
	FoundBy        map[TermKey][]string
	TribeltScanned bool
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
