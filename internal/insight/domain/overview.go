package domain

import (
	"cmp"
	"slices"
	"strings"
)

// Overview is what the home page shows, in figures: the terminology map, the competitors on the
// list now, and the newest Prompt run.
type Overview struct {
	Terms       TermsOverview
	Competitors CompetitorsOverview
	AiSearch    AiSearchOverview
}

// TermsOverview is the terminology map in figures: its terms, how many of them tribelt.nl uses
// (the count TermMap.TermsUsed gives Tribelt), and both per language and per kind of search.
type TermsOverview struct {
	Total   int
	Tribelt int
	// From says what Tribelt's count comes from: tribelt.nl's newest scan, or the research before one.
	From TermsSource
	// Languages are every language with a term, the most terms first; American English counts as English.
	Languages []LanguageTerms
	// Tiers are every kind of search with a term, in the map's order.
	Tiers []TierTerms
}

// LanguageTerms are the terms of one language, and how many of them tribelt.nl uses.
type LanguageTerms struct {
	Language string
	Terms    int
	Tribelt  int
}

// TierTerms are the terms of one kind of search, and how many of them tribelt.nl uses.
type TierTerms struct {
	Tier    string
	Terms   int
	Tribelt int
}

// Tiers are the kinds of search, from the most specific to the broadest, as the map orders them.
func Tiers() []string {
	return []string{"spec", "product", "application", "problem", "broad"}
}

// CompetitorsOverview is how many competitors the list holds now, and how many of them offer each
// of Tribelt's belt families.
type CompetitorsOverview struct {
	// Makers are the competitors on the list that are not archived, researched or added since.
	Makers int
	// Families are Tribelt's belt families, in order.
	Families []FamilyMakers
}

// FamilyMakers is how many of the competitors on the list offer one belt family.
type FamilyMakers struct {
	Family string
	Makers int
}

// AiSearchOverview is the newest Prompt run in figures: its answers, and how many of them name
// Tribelt and the competitors named most.
type AiSearchOverview struct {
	Answers int
	// Companies are Tribelt and the competitors named most, the most first; none before any answer.
	Companies []Mentioned
}

// Mentioned is how many answers name one company, and their share of the run's answers.
type Mentioned struct {
	CompanyID string
	Name      string
	Role      Role
	Answers   int
	// Share is ShareOfVoice: the same share the standings and the plan show.
	Share float64
}

// VoicesShown is how many competitors the home page lists beside Tribelt.
const VoicesShown = 4

// OverviewOf counts the home page's figures from the companies, the terminology map and the newest
// Prompt run with answers that count, nil before any.
func OverviewOf(companies []Company, terms TermMap, run *Run) Overview {
	return Overview{Terms: TermsOf(terms), Competitors: CompetitorsOf(companies), AiSearch: AiSearchOf(run, companies)}
}

// TermsOf counts the map's terms, in total, per language and per kind of search, and those
// tribelt.nl uses: the same count as TermMap.TermsUsed gives Tribelt.
func TermsOf(m TermMap) TermsOverview {
	out := TermsOverview{Total: len(m.Terms), Tribelt: m.TermsUsed()[TribeltID], From: m.TribeltFrom, Languages: []LanguageTerms{}, Tiers: []TierTerms{}}
	for _, t := range m.Terms {
		language := t.Language
		if language == "en-US" {
			language = "en"
		}
		i := slices.IndexFunc(out.Languages, func(l LanguageTerms) bool { return l.Language == language })
		if i < 0 {
			out.Languages = append(out.Languages, LanguageTerms{Language: language})
			i = len(out.Languages) - 1
		}
		out.Languages[i].Terms++
		j := slices.IndexFunc(out.Tiers, func(k TierTerms) bool { return k.Tier == t.Tier })
		if j < 0 {
			out.Tiers = append(out.Tiers, TierTerms{Tier: t.Tier})
			j = len(out.Tiers) - 1
		}
		out.Tiers[j].Terms++
		if t.UsedByTribelt {
			out.Languages[i].Tribelt++
			out.Tiers[j].Tribelt++
		}
	}
	slices.SortFunc(out.Languages, func(a, b LanguageTerms) int {
		return cmp.Or(cmp.Compare(b.Terms, a.Terms), strings.Compare(a.Language, b.Language))
	})
	slices.SortStableFunc(out.Tiers, func(a, b TierTerms) int { return cmp.Compare(tierOrder(a.Tier), tierOrder(b.Tier)) })
	return out
}

// tierOrder is a kind of search's place in Tiers; one the map does not know goes last.
func tierOrder(tier string) int {
	if i := slices.Index(Tiers(), tier); i >= 0 {
		return i
	}
	return len(Tiers())
}

// CompetitorsOf counts the competitors on the list now, researched or added since (not a group
// company or the mirror), and how many of them offer each of Tribelt's families: by the belt names
// the research or the review recorded, or the products their newest scan matched.
func CompetitorsOf(companies []Company) CompetitorsOverview {
	makers := slices.DeleteFunc(slices.Clone(companies), func(c Company) bool { return c.Role != RoleCompetitor })
	out := CompetitorsOverview{Makers: len(makers), Families: make([]FamilyMakers, 0, len(Families()))}
	for _, f := range Families() {
		n := 0
		for _, c := range makers {
			if slices.ContainsFunc(c.BeltNames, func(b BeltName) bool { return b.Family == f }) {
				n++
			}
		}
		out.Families = append(out.Families, FamilyMakers{Family: f, Makers: n})
	}
	return out
}

// AiSearchOf counts the run's answers, and how many name Tribelt and each of the competitors named
// in the most of them: a tie goes to the one named first more often, then to the id first
// alphabetically. Tribelt is listed with none when no answer names it; a company no longer on the
// list is left out.
func AiSearchOf(run *Run, companies []Company) AiSearchOverview {
	out := AiSearchOverview{Companies: []Mentioned{}}
	if run == nil {
		return out
	}
	out.Answers = len(run.Answers)
	if out.Answers == 0 {
		return out
	}
	answers, first := namings(run.Answers)
	shareOf := func(id string) float64 {
		v, _ := ShareOfVoice(run.Answers, id)
		return v
	}
	var competitors []Mentioned
	tribelt := Mentioned{CompanyID: TribeltID, Name: "Tribelt", Role: RoleTribelt, Answers: answers[TribeltID], Share: shareOf(TribeltID)}
	for _, c := range companies {
		if c.Role == RoleCompetitor && answers[c.ID] > 0 {
			competitors = append(competitors, Mentioned{CompanyID: c.ID, Name: c.Name, Role: c.Role, Answers: answers[c.ID], Share: shareOf(c.ID)})
		}
	}
	slices.SortFunc(competitors, func(a, b Mentioned) int {
		return cmp.Or(cmp.Compare(b.Answers, a.Answers), cmp.Compare(first[b.CompanyID], first[a.CompanyID]), strings.Compare(a.CompanyID, b.CompanyID))
	})
	out.Companies = append(out.Companies, competitors[:min(len(competitors), VoicesShown)]...)
	out.Companies = append(out.Companies, tribelt)
	// Tribelt goes after the competitors it ties with, as the home page has always shown it.
	slices.SortStableFunc(out.Companies, func(a, b Mentioned) int { return cmp.Compare(b.Answers, a.Answers) })
	return out
}

// namings counts, per company, the answers that name it (once each, as ShareOfVoice does) and
// those that name it first of all.
func namings(answers []Answer) (named, first map[string]int) {
	named, first = map[string]int{}, map[string]int{}
	for _, a := range answers {
		for _, id := range slices.Compact(slices.Sorted(slices.Values(a.Named))) {
			named[id]++
		}
		if a.First != "" {
			first[a.First]++
		}
	}
	return named, first
}
