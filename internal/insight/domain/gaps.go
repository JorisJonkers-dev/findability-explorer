package domain

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/JorisJonkers-dev/findability-explorer/internal/shared/site"
)

// The Recommendation run's rules (the spec's section 10): they find where tribelt.nl falls behind
// its competitors, from the newest complete Prompt run, each company's newest scan, and the search
// term list, and score each gap's benefit and effort from ScoreMin to ScoreMax. A Claude model then
// places and words each gap (Place).

// GapKind is what kind of change a gap asks for.
type GapKind string

const (
	// GapTerm is a search term competitors use and tribelt.nl does not.
	GapTerm GapKind = "term"
	// GapPage is a belt family with no tribelt.nl page in a language where competitors have one.
	GapPage GapKind = "page"
	// GapListing is a site the web responses cite that names competitors but not Tribelt.
	GapListing GapKind = "listing"
	// GapFact is a claim about Tribelt the fact check marked wrong.
	GapFact GapKind = "fact"
	// GapReadiness is an AI readiness check tribelt.nl fails that no written recommendation covers.
	GapReadiness GapKind = "readiness"
)

// Gap is one place where tribelt.nl falls behind: its kind, a key that names it across runs, the
// rules' draft of what to do, the facts behind it, and its scores.
type Gap struct {
	Key       string
	Kind      GapKind
	Title     string
	Specifics []string
	Language  string
	Family    string
	Benefit   int
	Effort    int
}

// Page is a page of tribelt.nl as its newest scan read it.
type Page struct {
	URL      string
	Title    string
	Language string
	Family   string
	Text     string
}

// GapInputs is what the rules read: the companies, the newest complete Prompt run, the search term
// list, tribelt.nl's readiness, and its pages.
type GapInputs struct {
	Companies []Company
	Run       *Run
	Terms     TermMap
	Tribelt   *Readiness
	Pages     []Page
}

// MaxGaps is the most gaps a run writes recommendations for: the best value first.
const MaxGaps = 40

// FamilyNames are the belt families as recommendations name them.
var FamilyNames = map[string]string{ //nolint:gochecknoglobals // a constant table
	"eyelink": "eyelink belts", "open_spiral": "open spiral belts", "closed_spiral": "closed spiral belts",
	"tri_flex_turn_curve": "curve-going and spiral-tower belts", "other": "other metal belts",
}

// LanguageNames are the languages as recommendations name them.
var LanguageNames = map[string]string{"nl": "Dutch", "de": "German", "en": "English"} //nolint:gochecknoglobals // a constant table

// Gaps are every gap the rules find, scored, the best value first, at most MaxGaps.
func Gaps(in GapInputs) []Gap {
	competitors := map[string]bool{}
	for _, c := range in.Companies {
		if c.Role == RoleCompetitor {
			competitors[c.ID] = true
		}
	}
	all := slices.Concat(termGaps(in, competitors), pageGaps(in, competitors), listingGaps(in, competitors), factGaps(in), readinessGaps(in.Tribelt))
	slices.SortStableFunc(all, func(a, b Gap) int {
		if c := b.Benefit*a.Effort - a.Benefit*b.Effort; c != 0 {
			return c
		}
		if c := b.Benefit - a.Benefit; c != 0 {
			return c
		}
		return cmp.Compare(a.Key, b.Key)
	})
	return all[:min(len(all), MaxGaps)]
}

// clamp keeps a score within ScoreMin and ScoreMax.
func clamp(n int) int { return max(ScoreMin, min(ScoreMax, n)) }

// band scores a count: 1 is 1, 2 or 3 is 2, 4 to 7 is 3, 8 or more is 4.
func band(n int) int {
	switch {
	case n >= 8:
		return 4
	case n >= 4:
		return 3
	case n >= 2:
		return 2
	default:
		return 1
	}
}

// strongIntent is whether a term's buyer intent shows a buyer close to buying.
func strongIntent(tier string) bool {
	return tier == "spec" || tier == "product" || tier == "application"
}

// hasPage is whether tribelt.nl has a page about family in language.
func hasPage(pages []Page, family, language string) bool {
	return slices.ContainsFunc(pages, func(p Page) bool { return p.Family == family && p.Language == language })
}

// termGaps are the terms competitors use and tribelt.nl does not. Benefit grows with how many
// competitors use it and with a buyer close to buying; effort is small on an existing page of its
// family and language, larger without one.
func termGaps(in GapInputs, competitors map[string]bool) []Gap {
	var out []Gap
	for _, t := range in.Terms.Terms {
		if t.UsedByTribelt {
			continue
		}
		users := 0
		for _, id := range slices.Compact(slices.Sorted(slices.Values(t.FoundBy))) {
			if competitors[id] {
				users++
			}
		}
		users = max(users, t.UsedBy)
		if users == 0 {
			continue
		}
		benefit := 1 + band(users)
		if strongIntent(t.Tier) {
			benefit++
		}
		effort := 3
		if hasPage(in.Pages, t.Family, t.Language) {
			effort = 2
		}
		out = append(out, Gap{
			Key: fmt.Sprintf("term:%s:%s", t.Language, strings.ToLower(t.Text)), Kind: GapTerm, Language: t.Language, Family: t.Family,
			Title: fmt.Sprintf("Use %q on tribelt.nl", t.Text), Benefit: clamp(benefit), Effort: effort,
			Specifics: []string{fmt.Sprintf("%d competitors use the %s search term %q; tribelt.nl does not.", users, LanguageNames[t.Language], t.Text), "Buyer intent: " + t.Tier + "."},
		})
	}
	return out
}

// pageGaps are the belt families with no tribelt.nl page in a language where competitors use the
// family's terms. Benefit grows with those competitors, and when the family's AI responses name a
// competitor more often than Tribelt; a new page is a larger piece of work.
func pageGaps(in GapInputs, competitors map[string]bool) []Gap {
	users, tribelt := familyUse(in.Terms, competitors)
	var out []Gap
	for k, who := range users {
		if tribelt[k] || hasPage(in.Pages, k.family, k.language) || FamilyNames[k.family] == "" || LanguageNames[k.language] == "" {
			continue
		}
		benefit := 1 + band(len(who))
		specifics := []string{fmt.Sprintf("%d competitors have %s pages about %s; tribelt.nl has none.", len(who), LanguageNames[k.language], FamilyNames[k.family])}
		if behind, leader := familyBehind(in, k.family); behind {
			benefit++
			specifics = append(specifics, fmt.Sprintf("AI responses about %s name %s more often than Tribelt.", FamilyNames[k.family], leader))
		}
		out = append(out, Gap{
			Key: fmt.Sprintf("page:%s:%s", k.language, k.family), Kind: GapPage, Language: k.language, Family: k.family,
			Title: fmt.Sprintf("A %s page about %s", LanguageNames[k.language], FamilyNames[k.family]), Benefit: clamp(benefit), Effort: 4, Specifics: specifics,
		})
	}
	return out
}

// familyKey is a belt family in a language.
type familyKey struct{ family, language string }

// familyUse is, per family and language, the competitors whose newest scan found a term of it, and
// whether tribelt.nl uses one.
func familyUse(terms TermMap, competitors map[string]bool) (map[familyKey]map[string]bool, map[familyKey]bool) {
	users := map[familyKey]map[string]bool{}
	tribelt := map[familyKey]bool{}
	for _, t := range terms.Terms {
		k := familyKey{t.Family, t.Language}
		tribelt[k] = tribelt[k] || t.UsedByTribelt
		for _, id := range t.FoundBy {
			if !competitors[id] {
				continue
			}
			if users[k] == nil {
				users[k] = map[string]bool{}
			}
			users[k][id] = true
		}
	}
	return users, tribelt
}

// familyBehind is whether the run's responses about family name a competitor more often than
// Tribelt, and which one names it most.
func familyBehind(in GapInputs, family string) (bool, string) {
	if in.Run == nil {
		return false, ""
	}
	count := map[string]int{}
	for _, a := range in.Run.Answers {
		if a.Family != family {
			continue
		}
		for _, id := range slices.Compact(slices.Sorted(slices.Values(a.Named))) {
			count[id]++
		}
	}
	best, bestN := "", 0
	for _, c := range in.Companies {
		if c.Role == RoleCompetitor && count[c.ID] > bestN {
			best, bestN = c.Name, count[c.ID]
		}
	}
	return bestN > count[TribeltID], best
}

// listingGaps are the sites the web responses cite that name competitors but not Tribelt: not a
// company's own site. Benefit grows with how many such responses cite it.
func listingGaps(in GapInputs, competitors map[string]bool) []Gap {
	if in.Run == nil {
		return nil
	}
	own := map[string]bool{"tribelt.nl": true}
	for _, c := range in.Companies {
		if d := site.Domain(c.Website); d != "" {
			own[d] = true
		}
	}
	cited := map[string]int{}
	for _, a := range in.Run.Answers {
		if a.Mode != ModeWeb || slices.Contains(a.Named, TribeltID) || !slices.ContainsFunc(a.Named, func(id string) bool { return competitors[id] }) {
			continue
		}
		seen := map[string]bool{}
		for _, src := range a.Sources {
			if d := site.Domain(src); d != "" && !own[d] && !seen[d] {
				seen[d] = true
				cited[d]++
			}
		}
	}
	var out []Gap
	for d, n := range cited {
		out = append(out, Gap{
			Key: "listing:" + d, Kind: GapListing, Title: "Get Tribelt listed on " + d, Benefit: clamp(1 + band(n)), Effort: 3,
			Specifics: []string{fmt.Sprintf("%d web responses cite %s and name competitors, but not Tribelt.", n, d)},
		})
	}
	return out
}

// factGaps are the claims about Tribelt the fact check marked wrong, each once. Benefit grows with
// how many responses make it; a correction is a small piece of work.
func factGaps(in GapInputs) []Gap {
	if in.Run == nil {
		return nil
	}
	count := map[string]int{}
	text := map[string]string{}
	for _, a := range in.Run.Answers {
		for _, claim := range a.WrongClaims {
			k := strings.ToLower(strings.TrimSpace(claim))
			if k == "" {
				continue
			}
			if count[k] == 0 {
				text[k] = strings.TrimSpace(claim)
			}
			count[k]++
		}
	}
	var out []Gap
	for k, n := range count {
		out = append(out, Gap{
			Key: "fact:" + k, Kind: GapFact, Title: "Correct what AI responses get wrong about Tribelt", Benefit: clamp(2 + band(n)), Effort: 1,
			Specifics: []string{fmt.Sprintf("%d responses claim: %q", n, text[k])},
		})
	}
	return out
}

// readinessGaps are the AI readiness checks tribelt.nl fails that no written recommendation covers.
func readinessGaps(r *Readiness) []Gap {
	if r == nil {
		return nil
	}
	var out []Gap
	if len(r.Blocked) > 0 {
		blocked := slices.Sorted(slices.Values(r.Blocked))
		out = append(out, Gap{
			Key: "readiness:crawlers", Kind: GapReadiness, Title: "Let AI crawlers read tribelt.nl", Benefit: 5, Effort: 1,
			Specifics: []string{"robots.txt keeps out " + strings.Join(blocked, ", ") + "."},
		})
	}
	if !r.Organization {
		out = append(out, Gap{
			Key: "readiness:organization", Kind: GapReadiness, Title: "Describe Tribelt as an Organization in structured data", Benefit: 3, Effort: 2,
			Specifics: []string{"The home page has no Organization structured data."},
		})
	}
	if !r.LanguageVersions {
		out = append(out, Gap{
			Key: "readiness:languages", Kind: GapReadiness, Title: "Pages in Dutch, German, and English", Benefit: 3, Effort: 4,
			Specifics: []string{"tribelt.nl does not have its pages in each of Dutch, German, and English."},
		})
	}
	return out
}

// Stale reports whether tribelt.nl was last scanned before the newest Prompt run started: its page
// text is older than the responses.
func Stale(scannedAt time.Time, run *Run) bool {
	return run != nil && (scannedAt.IsZero() || scannedAt.Before(run.StartedAt))
}
