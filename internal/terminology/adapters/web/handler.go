// Package web is the terminology context's inbound HTTP adapter: it implements the terminology
// operations of the generated ogen Handler and maps between the contract's types and the domain's.
package web

import (
	"context"

	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/oas"
	"github.com/JorisJonkers-dev/findability-explorer/internal/terminology/domain"
)

// UseCases is what the adapter needs from the application layer.
type UseCases interface {
	Search(q domain.Query) domain.Result
	Companies() []domain.Company
}

// Handler implements the terminology operations. Business rules stay in the use cases.
type Handler struct {
	uc UseCases
}

// New returns a Handler calling uc.
func New(uc UseCases) *Handler {
	return &Handler{uc: uc}
}

// SearchTerms implements searchTerms. ogen has already checked every enum against the contract,
// whose values are the domain's.
func (h *Handler) SearchTerms(_ context.Context, params oas.SearchTermsParams) (oas.SearchTermsRes, error) {
	result := h.uc.Search(domain.Query{
		Text:     params.Q.Or(""),
		Language: domain.Language(params.Language.Or("")),
		Tier:     domain.Tier(params.Tier.Or("")),
		Family:   domain.Family(params.Family.Or("")),
		User:     params.User.Or(""),
		Limit:    int(params.Limit.Or(0)),
	})
	items := make([]oas.Term, 0, len(result.Items))
	for _, t := range result.Items {
		items = append(items, ToTerm(t))
	}
	return &oas.TermSearch{
		Items: items,
		Total: int32(result.Total), //nolint:gosec // the map holds a few hundred terms
		Facets: oas.Facets{
			Languages: toCounts(result.Facets.Languages),
			Tiers:     toCounts(result.Facets.Tiers),
			Families:  toCounts(result.Facets.Families),
		},
	}, nil
}

// ListCompetitors implements listCompetitors.
func (h *Handler) ListCompetitors(context.Context) (oas.ListCompetitorsRes, error) {
	companies := h.uc.Companies()
	items := make([]oas.Competitor, 0, len(companies))
	for _, c := range companies {
		names := make([]oas.BeltName, 0, len(c.BeltNames))
		for _, b := range c.BeltNames {
			names = append(names, oas.BeltName{Name: b.Name, Family: oas.BeltFamily(b.Family)})
		}
		items = append(items, oas.Competitor{
			Name: c.Name, Country: c.Country, Website: c.Website, Languages: nonNil(c.Languages),
			GroupCompany: c.GroupCompany, BeltNames: names, HomepageTitle: c.HomepageTitle,
		})
	}
	return &oas.CompetitorList{Items: items}, nil
}

// ToTerm maps a domain term to the contract's; the analysis context reuses it for known terms.
func ToTerm(t domain.Term) oas.Term {
	return oas.Term{
		Term: t.Text, Language: oas.Language(t.Language), Family: oas.BeltFamily(t.Family),
		Tier: oas.IntentTier(t.Tier), UsedBy: nonNil(t.UsedBy), Source: t.Source,
	}
}

func toCounts(counts []domain.FacetCount) []oas.FacetCount {
	out := make([]oas.FacetCount, 0, len(counts))
	for _, c := range counts {
		out = append(out, oas.FacetCount{Value: c.Value, Count: int32(c.Count)}) //nolint:gosec // small counts
	}
	return out
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
