package domain

import (
	"cmp"
	"slices"
	"strings"
)

// Families are Tribelt's belt families, in the order the products show them.
func Families() []string {
	return []string{"eyelink", "open_spiral", "closed_spiral", "tri_flex_turn_curve", "other"}
}

// MapTerm is a term of the terminology map: its family and intent tier, how many competitors on
// the list use it (terminology's SiteUse.Makers), whether tribelt.nl uses it, and the companies
// whose newest finished scan found it.
type MapTerm struct {
	Text     string
	Language string
	Family   string
	Tier     string
	UsedBy   int
	// UsedByTribelt is whether tribelt.nl uses the term, as the terminology context decides it:
	// its newest scan found it, or, before any scan of tribelt.nl, the research credits Tribelt.
	UsedByTribelt bool
	FoundBy       []string
}

// TermsSource is what says which of the map's terms tribelt.nl uses.
type TermsSource string

const (
	// TermsFromScan is tribelt.nl's newest finished scan.
	TermsFromScan TermsSource = "scan"
	// TermsFromResearch is the desk research, before tribelt.nl has a finished scan.
	TermsFromResearch TermsSource = "research"
)

// TermMap is the terminology map as the insight context reads it: its terms, what says which of
// them tribelt.nl uses, and the companies a finished scan read.
type TermMap struct {
	Terms       []MapTerm
	TribeltFrom TermsSource
	Scanned     []string
}

// TermsUsed is how many of the map's terms each company's site uses: for every company a scan
// read, the terms its newest scan found, in their own language; for Tribelt always, the terms UsedByTribelt marks, so
// before a scan of tribelt.nl the ones the research credits it with. It is the one count of
// "search terms used" every page shows.
func (m TermMap) TermsUsed() map[string]int {
	out := map[string]int{TribeltID: 0}
	for _, id := range m.Scanned {
		out[id] = 0
	}
	for _, t := range m.Terms {
		if t.UsedByTribelt {
			out[TribeltID]++
		}
		// A company a scan found the term at was read by that scan, listed or not.
		for _, id := range slices.Compact(slices.Sorted(slices.Values(t.FoundBy))) {
			if id != TribeltID {
				out[id]++
			}
		}
	}
	return out
}

// Seller is a company that sells a family, with its own names for it.
type Seller struct {
	CompanyID string
	Name      string
	Website   string
	Role      Role
	Names     []string
}

// ScopeShare is how many of a family's answers in one scope name Tribelt.
type ScopeShare struct {
	Scope   string
	Answers int
	Named   int
}

// Product is one belt family against the field.
type Product struct {
	Family       string
	TribeltNames []string
	// Sellers are the competitors that sell it, by name.
	Sellers []Seller
	// Terms are the family's map terms, the most used first.
	Terms     []MapTerm
	TermsUsed int
	// Answers and Named count the newest run's answers about the family, and those naming Tribelt.
	Answers int
	Named   int
	ByScope []ScopeShare
	// MostNamed are the companies its answers name most, Tribelt among them when named.
	MostNamed []CompanyShare
}

// TopTerm is the family's most used term, or "" without terms.
func (p Product) TopTerm() string {
	if len(p.Terms) == 0 {
		return ""
	}
	return p.Terms[0].Text
}

// MostNamedShown is how many companies a product's answers list as named most.
const MostNamedShown = 5

// Products compares each belt family: who sells it under which names, the terms buyers search for
// it with and which Tribelt uses, and how the newest run's answers about it treat Tribelt.
func Products(companies []Company, terms []MapTerm, run *Run) []Product {
	out := make([]Product, 0, len(Families()))
	for _, f := range Families() {
		p := Product{Family: f, TribeltNames: []string{}, Sellers: []Seller{}, Terms: []MapTerm{}, ByScope: []ScopeShare{}, MostNamed: []CompanyShare{}}
		p.sellers(companies)
		p.terms(terms)
		if run != nil {
			p.answers(run, companies)
		}
		out = append(out, p)
	}
	return out
}

// sellers finds Tribelt's names for the product's family and the competitors that sell it.
func (p *Product) sellers(companies []Company) {
	for _, c := range companies {
		names := namesFor(c, p.Family)
		switch {
		case c.Role == RoleTribelt:
			p.TribeltNames = names
		case c.Role == RoleCompetitor && len(names) > 0:
			p.Sellers = append(p.Sellers, Seller{CompanyID: c.ID, Name: c.Name, Website: c.Website, Role: c.Role, Names: names})
		}
	}
	slices.SortStableFunc(p.Sellers, func(a, b Seller) int { return byName(a.Name, b.Name) })
}

// terms keeps the map's terms for the product's family, the most used first, and counts Tribelt's.
func (p *Product) terms(terms []MapTerm) {
	for _, t := range terms {
		if t.Family == p.Family {
			p.Terms = append(p.Terms, t)
			if t.UsedByTribelt {
				p.TermsUsed++
			}
		}
	}
	slices.SortStableFunc(p.Terms, func(a, b MapTerm) int {
		return cmp.Or(cmp.Compare(b.UsedBy, a.UsedBy), strings.Compare(a.Language, b.Language), strings.Compare(a.Text, b.Text))
	})
}

// namesFor are the company's own names for a family, each once.
func namesFor(c Company, family string) []string {
	out := []string{}
	for _, b := range c.BeltNames {
		if b.Family == family && !slices.Contains(out, b.Name) {
			out = append(out, b.Name)
		}
	}
	return out
}

// answers counts the run's answers about the product's family, per scope, and who they name.
func (p *Product) answers(run *Run, companies []Company) {
	mine := run.Keep(func(a Answer) bool { return a.Family == p.Family })
	p.Answers = len(mine)
	for _, a := range mine {
		named := slices.Contains(a.Named, TribeltID)
		if named {
			p.Named++
		}
		i := slices.IndexFunc(p.ByScope, func(s ScopeShare) bool { return s.Scope == a.Scope })
		if i < 0 {
			p.ByScope = append(p.ByScope, ScopeShare{Scope: a.Scope})
			i = len(p.ByScope) - 1
		}
		p.ByScope[i].Answers++
		if named {
			p.ByScope[i].Named++
		}
	}
	var shares []CompanyShare
	for _, c := range companies {
		if c.Role != RoleCompetitor && c.Role != RoleTribelt {
			continue
		}
		if v, _ := ShareOfVoice(mine, c.ID); v > 0 {
			shares = append(shares, CompanyShare{CompanyID: c.ID, Share: v})
		}
	}
	names := nameOf(companies)
	slices.SortStableFunc(shares, func(a, b CompanyShare) int {
		return cmp.Or(cmp.Compare(b.Share, a.Share), byName(names[a.CompanyID], names[b.CompanyID]))
	})
	p.MostNamed = shares[:min(MostNamedShown, len(shares))]
}

// Finding is something that stands out among the families.
type Finding string

const (
	// FindingCrowded is the family the most competitors sell.
	FindingCrowded Finding = "crowded"
	// FindingFewestTerms is the family whose terms Tribelt uses least, as a share.
	FindingFewestTerms Finding = "fewestTerms"
	// FindingBestInAI is the family whose answers name Tribelt most, as a share.
	FindingBestInAI Finding = "bestInAI"
)

// Standout is a finding and the family it is about.
type Standout struct {
	Finding Finding
	Family  string
}

// Standouts are what stands out: the most crowded family, the one whose terms Tribelt uses least,
// and the one whose answers name Tribelt most; a finding without data is left out, a tie goes to
// the family listed first.
func Standouts(products []Product) []Standout {
	var out []Standout
	best := func(f Finding, value func(Product) (float64, bool)) {
		family, top, found := "", 0.0, false
		for _, p := range products {
			if v, ok := value(p); ok && (!found || v > top) {
				family, top, found = p.Family, v, true
			}
		}
		if found {
			out = append(out, Standout{Finding: f, Family: family})
		}
	}
	best(FindingCrowded, func(p Product) (float64, bool) { return float64(len(p.Sellers)), len(p.Sellers) > 0 })
	best(FindingFewestTerms, func(p Product) (float64, bool) {
		if len(p.Terms) == 0 {
			return 0, false
		}
		return -float64(p.TermsUsed) / float64(len(p.Terms)), true
	})
	best(FindingBestInAI, func(p Product) (float64, bool) {
		if p.Answers == 0 || p.Named == 0 {
			return 0, false
		}
		return float64(p.Named) / float64(p.Answers), true
	})
	return out
}
