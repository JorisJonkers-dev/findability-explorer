package domain

import (
	"cmp"
	"slices"
	"strings"
)

// Query is a search of the map. An empty field does not filter.
type Query struct {
	Text     string
	Language Language
	Tier     Tier
	Family   Family
	User     string
	Limit    int
}

// FacetCount is how many matching terms have one value of a facet.
type FacetCount struct {
	Value string
	Count int
}

// Facets counts the matching terms per value. Each facet ignores its own filter, so the counts
// say how many terms a user would see after switching that filter to another value.
type Facets struct {
	Languages []FacetCount
	Tiers     []FacetCount
	Families  []FacetCount
}

// Result is one page of matching terms.
type Result struct {
	Items  []Term
	Total  int
	Facets Facets
}

// Relevance of a term to a query text, highest first. A term with relevance 0 does not match.
const (
	relevanceExact    = 100
	relevancePrefix   = 60
	relevanceWord     = 40
	relevanceContains = 25
	relevanceUser     = 10
)

// relevance scores how well t matches the folded query text q; q is never empty here.
func relevance(t Term, q string) int {
	text := Fold(t.Text)
	switch {
	case text == q:
		return relevanceExact
	case strings.HasPrefix(text, q):
		return relevancePrefix
	case strings.Contains(text, " "+q):
		return relevanceWord
	case strings.Contains(text, q):
		return relevanceContains
	}
	for _, u := range t.UsedBy {
		if strings.Contains(Fold(u), q) {
			return relevanceUser
		}
	}
	return 0
}

// usedBy reports whether a company whose folded name contains user uses t.
func usedBy(t Term, user string) bool {
	for _, u := range t.UsedBy {
		if strings.Contains(Fold(u), user) {
			return true
		}
	}
	return false
}

type scored struct {
	term  Term
	score int
}

// Search returns the terms that match q, best match first, then by language, family and text.
// Without query text every term matches with the same score. q.Limit below 1 returns no items
// but still the total and the facets.
func Search(terms []Term, q Query) Result {
	text, user := Fold(q.Text), Fold(q.User)
	var matched []scored
	var facetPool []scored
	for _, t := range terms {
		score := 1
		if text != "" {
			score = relevance(t, text)
		}
		if score == 0 || (user != "" && !usedBy(t, user)) {
			continue
		}
		facetPool = append(facetPool, scored{term: t, score: score})
		if (q.Language == "" || t.Language == q.Language) &&
			(q.Tier == "" || t.Tier == q.Tier) &&
			(q.Family == "" || t.Family == q.Family) {
			matched = append(matched, scored{term: t, score: score})
		}
	}
	slices.SortStableFunc(matched, func(a, b scored) int {
		return cmp.Or(
			cmp.Compare(b.score, a.score),
			cmp.Compare(slices.Index(Languages(), a.term.Language), slices.Index(Languages(), b.term.Language)),
			cmp.Compare(slices.Index(Families(), a.term.Family), slices.Index(Families(), b.term.Family)),
			strings.Compare(Fold(a.term.Text), Fold(b.term.Text)),
		)
	})
	items := make([]Term, 0, min(max(q.Limit, 0), len(matched)))
	for i := 0; i < len(matched) && i < q.Limit; i++ {
		items = append(items, matched[i].term)
	}
	return Result{Items: items, Total: len(matched), Facets: facets(facetPool, q)}
}

// facets counts pool per value, applying every filter of q except the facet's own.
func facets(pool []scored, q Query) Facets {
	return Facets{
		Languages: count(Languages(), func(t Term) Language { return t.Language }, pool,
			func(t Term) bool { return keep(t.Tier, q.Tier) && keep(t.Family, q.Family) }),
		Tiers: count(Tiers(), func(t Term) Tier { return t.Tier }, pool,
			func(t Term) bool { return keep(t.Language, q.Language) && keep(t.Family, q.Family) }),
		Families: count(Families(), func(t Term) Family { return t.Family }, pool,
			func(t Term) bool { return keep(t.Language, q.Language) && keep(t.Tier, q.Tier) }),
	}
}

func keep[T ~string](value, filter T) bool { return filter == "" || value == filter }

// count returns one FacetCount per value in values, in that order, counting the terms in pool that
// pass other and have that value.
func count[T ~string](values []T, of func(Term) T, pool []scored, other func(Term) bool) []FacetCount {
	counts := make(map[T]int, len(values))
	for _, s := range pool {
		if other(s.term) {
			counts[of(s.term)]++
		}
	}
	out := make([]FacetCount, 0, len(values))
	for _, v := range values {
		out = append(out, FacetCount{Value: string(v), Count: counts[v]})
	}
	return out
}

// SortCompanies orders companies with competitors first, then by folded name.
func SortCompanies(companies []Company) []Company {
	out := slices.Clone(companies)
	slices.SortStableFunc(out, func(a, b Company) int {
		return cmp.Or(cmp.Compare(groupRank(a), groupRank(b)), strings.Compare(Fold(a.Name), Fold(b.Name)))
	})
	return out
}

func groupRank(c Company) int {
	if c.GroupCompany {
		return 1
	}
	return 0
}
