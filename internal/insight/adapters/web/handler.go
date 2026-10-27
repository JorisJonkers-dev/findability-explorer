// Package web is the insight context's inbound HTTP adapter: it implements the standings,
// products and plan operations of the generated ogen Handler and maps between the contract's
// types and the domain's.
package web

import (
	"context"
	"net/url"

	"github.com/JorisJonkers-dev/findability-explorer/internal/insight/app"
	"github.com/JorisJonkers-dev/findability-explorer/internal/insight/domain"
	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/oas"
)

// UseCases is what the adapter needs from the application layer.
type UseCases interface {
	Standings(ctx context.Context) (app.Standings, error)
	AiSearch(ctx context.Context) (app.AiSearch, error)
	Products(ctx context.Context) (app.Products, error)
}

// Handler implements the insight operations. Business rules stay in the use cases.
type Handler struct {
	uc    UseCases
	actor func(context.Context) string
}

// New returns a Handler calling uc; actor names the caller in what it records.
func New(uc UseCases, actor func(context.Context) string) *Handler {
	return &Handler{uc: uc, actor: actor}
}

// GetStandings implements getStandings.
func (h *Handler) GetStandings(ctx context.Context) (oas.GetStandingsRes, error) {
	s, err := h.uc.Standings(ctx)
	if err != nil {
		return nil, err
	}
	out := &oas.Standings{Measures: toStandings(s.Standings), Trailing: int32(s.Trailing)} //nolint:gosec // at most five measures
	if s.Run != nil {
		out.RunStartedAt = oas.NewOptDateTime(s.Run.StartedAt)
	}
	return out, nil
}

func toStandings(in []domain.Standing) []oas.Standing {
	out := make([]oas.Standing, 0, len(in))
	for _, s := range in {
		o := oas.Standing{
			Measure: oas.StandingMeasure(s.Measure), Competitors: int32(s.Competitors), //nolint:gosec // a count of companies
			Trails: s.Trails(), Leads: s.Leads(),
		}
		if s.Tribelt != nil {
			o.Tribelt = oas.NewOptFloat64(*s.Tribelt)
		}
		if s.Median != nil {
			o.Median = oas.NewOptFloat64(*s.Median)
		}
		if s.Leader != nil {
			o.Leader = oas.NewOptMeasureLeader(oas.MeasureLeader{CompetitorId: oas.CompetitorId(s.Leader.CompanyID), Name: s.Leader.Name, Value: s.Leader.Value})
		}
		out = append(out, o)
	}
	return out
}

// GetAiSearchSummary implements getAiSearchSummary.
func (h *Handler) GetAiSearchSummary(ctx context.Context) (oas.GetAiSearchSummaryRes, error) {
	s, err := h.uc.AiSearch(ctx)
	if err != nil {
		return nil, err
	}
	out := &oas.AiSearchSummary{
		Goal: s.Goal, Followed: []oas.FollowedCompany{}, Trend: []oas.ShareTrendPoint{}, Crawlers: []oas.TribeltCrawler{},
		KeepingOut: []oas.KeepingOut{}, Ranking: []oas.ReadinessRank{},
		Sites: int32(s.Sites), Questions: int32(s.Questions), Providers: int32(s.Providers), //nolint:gosec // small counts
	}
	for _, c := range s.Followed {
		out.Followed = append(out.Followed, oas.FollowedCompany{CompetitorId: oas.CompetitorId(c.ID), Name: c.Name, Role: oas.CompanyRole(c.Role)})
	}
	for _, p := range s.Trend {
		shares := make([]float64, 0, len(p.Shares))
		for _, sh := range p.Shares {
			shares = append(shares, sh.Share)
		}
		out.Trend = append(out.Trend, oas.ShareTrendPoint{RunStartedAt: p.StartedAt, Shares: shares})
	}
	for _, c := range s.Crawlers {
		row := oas.TribeltCrawler{Bot: c.Bot, Lab: c.Lab, Gets: c.Gets, Feeds: oas.CrawlerFeeds(c.Feeds)}
		if c.Access != "" {
			row.Access = oas.NewOptCrawlerAccess(oas.CrawlerAccess(c.Access))
		}
		out.Crawlers = append(out.Crawlers, row)
	}
	for _, k := range s.KeepingOut {
		out.KeepingOut = append(out.KeepingOut, oas.KeepingOut{CompetitorId: oas.CompetitorId(k.CompanyID), Name: k.Name, Blocked: k.Blocked})
	}
	for _, r := range s.Ranking {
		out.Ranking = append(out.Ranking, oas.ReadinessRank{CompetitorId: oas.CompetitorId(r.CompanyID), Name: r.Name, Role: oas.CompanyRole(r.Role), Score: r.Score})
	}
	return out, nil
}

// GetProducts implements getProducts.
func (h *Handler) GetProducts(ctx context.Context) (oas.GetProductsRes, error) {
	p, err := h.uc.Products(ctx)
	if err != nil {
		return nil, err
	}
	companies := map[string]domain.Company{}
	for _, c := range p.Companies {
		companies[c.ID] = c
	}
	out := &oas.ProductsOverview{Items: make([]oas.Product, 0, len(p.Products)), Standouts: make([]oas.ProductStandout, 0, len(p.Standouts))}
	for _, pr := range p.Products {
		out.Items = append(out.Items, toProduct(pr, companies))
	}
	for _, s := range p.Standouts {
		out.Standouts = append(out.Standouts, oas.ProductStandout{Finding: oas.ProductFinding(s.Finding), Family: oas.PromptFamily(s.Family)})
	}
	return out, nil
}

//nolint:gosec // the counts are of a run's answers and the map's terms, far below int32
func toProduct(p domain.Product, companies map[string]domain.Company) oas.Product {
	o := oas.Product{
		Family: oas.PromptFamily(p.Family), TribeltNames: p.TribeltNames, Sellers: make([]oas.ProductSeller, 0, len(p.Sellers)),
		Terms: make([]oas.ProductTerm, 0, len(p.Terms)), TermsUsed: int32(p.TermsUsed), Answers: int32(p.Answers), Named: int32(p.Named),
		ByScope: make([]oas.ProductScope, 0, len(p.ByScope)), MostNamed: make([]oas.ProductMention, 0, len(p.MostNamed)),
	}
	for _, s := range p.Sellers {
		seller := oas.ProductSeller{CompetitorId: oas.CompetitorId(s.CompanyID), Name: s.Name, Names: s.Names}
		if u, err := url.Parse(s.Website); err == nil && s.Website != "" {
			seller.WebsiteUrl = oas.NewOptURI(*u)
		}
		o.Sellers = append(o.Sellers, seller)
	}
	for _, t := range p.Terms {
		o.Terms = append(o.Terms, oas.ProductTerm{Term: t.Text, Language: t.Language, Tier: oas.IntentTier(t.Tier), UsedBy: int32(t.UsedBy), UsedByTribelt: t.UsedByTribelt})
	}
	for _, s := range p.ByScope {
		o.ByScope = append(o.ByScope, oas.ProductScope{Scope: oas.PromptScope(s.Scope), Answers: int32(s.Answers), Named: int32(s.Named)})
	}
	for _, m := range p.MostNamed {
		c := companies[m.CompanyID]
		o.MostNamed = append(o.MostNamed, oas.ProductMention{CompetitorId: oas.CompetitorId(m.CompanyID), Name: c.Name, Role: oas.CompanyRole(c.Role), Share: m.Share})
	}
	return o
}
