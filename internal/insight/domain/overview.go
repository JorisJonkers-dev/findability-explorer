package domain

import (
	"cmp"
	"slices"
	"strings"
	"unicode"
)

// Overview is what the home page shows, in figures: the terminology map, the competitors the desk
// research found, and the newest Prompt run.
type Overview struct {
	Terms       TermsOverview
	Competitors CompetitorsOverview
	AiSearch    AiSearchOverview
}

// TermsOverview is the terminology map in figures: its terms, how many of them Tribelt's own site
// uses, and both per language.
type TermsOverview struct {
	Total   int
	Tribelt int
	// Languages are every language with a term, the most terms first; American English counts as English.
	Languages []LanguageTerms
}

// LanguageTerms are the terms of one language, and how many of them Tribelt's own site uses.
type LanguageTerms struct {
	Language string
	Terms    int
	Tribelt  int
}

// CompetitorsOverview is how many competitors the desk research found, and how many of them offer
// each of Tribelt's belt families.
type CompetitorsOverview struct {
	Makers int
	// Families are Tribelt's belt families, in order.
	Families []FamilyMakers
}

// FamilyMakers is how many of the researched competitors offer one belt family.
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

// Mentioned is how many answers name one company.
type Mentioned struct {
	CompanyID string
	Name      string
	Role      Role
	Answers   int
}

// VoicesShown is how many competitors the home page lists beside Tribelt.
const VoicesShown = 4

// OverviewOf counts the home page's figures from the companies, the terminology map and the newest
// Prompt run with answers that count, nil before any.
func OverviewOf(companies []Company, terms []MapTerm, run *Run) Overview {
	return Overview{Terms: TermsOf(terms), Competitors: CompetitorsOf(companies), AiSearch: AiSearchOf(run, companies)}
}

// TermsOf counts the map's terms, in total and per language.
func TermsOf(terms []MapTerm) TermsOverview {
	out := TermsOverview{Total: len(terms), Languages: []LanguageTerms{}}
	for _, t := range terms {
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
		if t.UsedByTribelt {
			out.Languages[i].Tribelt++
			out.Tribelt++
		}
	}
	slices.SortFunc(out.Languages, func(a, b LanguageTerms) int {
		return cmp.Or(cmp.Compare(b.Terms, a.Terms), strings.Compare(a.Language, b.Language))
	})
	return out
}

// CompetitorsOf counts the competitors the desk research found, not one added since, a group
// company or the mirror, and how many of them offer each of Tribelt's families.
func CompetitorsOf(companies []Company) CompetitorsOverview {
	makers := slices.DeleteFunc(slices.Clone(companies), func(c Company) bool { return !c.Researched || c.Role != RoleCompetitor })
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
	type count struct{ answers, first int }
	counts := map[string]*count{}
	for _, a := range run.Answers {
		for _, id := range a.Named {
			if counts[id] == nil {
				counts[id] = &count{}
			}
			counts[id].answers++
			if a.First == id {
				counts[id].first++
			}
		}
	}
	var competitors []Mentioned
	tribelt := Mentioned{CompanyID: TribeltID, Name: "Tribelt", Role: RoleTribelt}
	for _, c := range companies {
		n, ok := counts[c.ID]
		switch {
		case c.ID == TribeltID && ok:
			tribelt.Answers = n.answers
		case c.Role == RoleCompetitor && ok:
			competitors = append(competitors, Mentioned{CompanyID: c.ID, Name: c.Name, Role: c.Role, Answers: n.answers})
		}
	}
	slices.SortFunc(competitors, func(a, b Mentioned) int {
		return cmp.Or(cmp.Compare(b.Answers, a.Answers), cmp.Compare(counts[b.CompanyID].first, counts[a.CompanyID].first), strings.Compare(a.CompanyID, b.CompanyID))
	})
	out.Companies = append(competitors[:min(len(competitors), VoicesShown)], tribelt)
	// Tribelt goes after the competitors it ties with, as the home page has always shown it.
	slices.SortStableFunc(out.Companies, func(a, b Mentioned) int { return cmp.Compare(b.Answers, a.Answers) })
	return out
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
