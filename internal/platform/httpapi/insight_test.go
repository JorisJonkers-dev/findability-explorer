package httpapi_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/JorisJonkers-dev/findability-explorer/internal/insight/app"
	"github.com/JorisJonkers-dev/findability-explorer/internal/insight/domain"
)

// insightCases is a stand-in for the insight use cases, so these tests exercise the contract.
type insightCases struct {
	err error
	run *domain.Run
}

func (a *analyses) insights() *insightCases {
	if a.insight == nil {
		a.insight = &insightCases{}
	}
	return a.insight
}

func (c *insightCases) Standings(context.Context) (app.Standings, error) {
	share := 0.24
	return app.Standings{
		Standings: []domain.Standing{
			{Measure: domain.MeasureShareOfVoice, Tribelt: &share, Median: new(0.17), Leader: &domain.Leader{CompanyID: "ashworth", Name: "Ashworth", Value: 0.57}, Competitors: 38},
			{Measure: domain.MeasureSitemapPages},
		},
		Trailing: 1,
		Run:      c.run,
	}, c.err
}

func (c *insightCases) AiSearch(context.Context) (app.AiSearch, error) {
	at := time.Date(2026, 11, 2, 6, 0, 0, 0, time.UTC)
	return app.AiSearch{
		Goal:     0.4,
		Followed: []domain.Company{{ID: "tribelt", Name: "Tribelt B.V.", Role: domain.RoleTribelt}, {ID: "ashworth", Name: "Ashworth", Role: domain.RoleCompetitor}},
		Trend:    []domain.TrendPoint{{StartedAt: at, Shares: []domain.CompanyShare{{CompanyID: "tribelt", Share: 0.24}, {CompanyID: "ashworth", Share: 0.57}}}},
		Crawlers: []app.CrawlerRow{
			{Crawler: domain.Crawler{Bot: "GPTBot", Lab: "OpenAI", Gets: "What future ChatGPT models know", Feeds: domain.FeedsFuture}, Access: domain.BotBlocked},
			{Crawler: domain.Crawler{Bot: "Googlebot", Lab: "Google", Gets: "Google Search", Feeds: domain.FeedsToday}},
		},
		KeepingOut: []domain.KeepingOut{{CompanyID: "wire-belt", Name: "Wire Belt", Blocked: []string{"GPTBot"}}},
		Ranking:    []domain.Ranked{{CompanyID: "martens", Name: "Märtens", Role: domain.RoleCompetitor, Score: 3.5}},
		Sites:      41, Questions: 20, Providers: 5,
	}, c.err
}

func TestTheAiSearchSummaryIsForEveryoneSignedIn(t *testing.T) {
	a := &analyses{}
	srv := serve(t, a)
	r := callAs(t, srv, http.MethodGet, "/api/v1/ai-search", "", "tester")
	if r.code != http.StatusOK || r.body["goal"].(float64) != 0.4 || r.body["sites"].(float64) != 41 || r.body["providers"].(float64) != 5 {
		t.Fatalf("summary = %d %v", r.code, r.body)
	}
	trend := r.body["trend"].([]any)[0].(map[string]any)
	if trend["runStartedAt"] != "2026-11-02T06:00:00Z" || trend["shares"].([]any)[1].(float64) != 0.57 {
		t.Errorf("trend = %v", trend)
	}
	crawlers := r.body["crawlers"].([]any)
	if c := crawlers[0].(map[string]any); c["access"] != "blocked" || c["feeds"] != "future" || c["lab"] != "OpenAI" {
		t.Errorf("crawler = %v", c)
	}
	if c := crawlers[1].(map[string]any); c["access"] != nil {
		t.Errorf("a crawler without access = %v", c)
	}
	if k := r.body["keepingOut"].([]any)[0].(map[string]any); k["name"] != "Wire Belt" || k["blocked"].([]any)[0] != "GPTBot" {
		t.Errorf("keeping out = %v", k)
	}
	if rk := r.body["ranking"].([]any)[0].(map[string]any); rk["score"].(float64) != 3.5 || rk["role"] != "competitor" {
		t.Errorf("ranking = %v", rk)
	}
	if f := r.body["followed"].([]any)[0].(map[string]any); f["role"] != "tribelt" {
		t.Errorf("followed = %v", f)
	}
	a.insights().err = errors.New("down")
	if r := callAs(t, srv, http.MethodGet, "/api/v1/ai-search", "", "tester"); r.code != http.StatusInternalServerError {
		t.Errorf("a failure = %d", r.code)
	}
}

func (c *insightCases) Products(context.Context) (app.Products, error) {
	eyelink := domain.Product{
		Family: "eyelink", TribeltNames: []string{"Draadogenbanden"},
		Sellers: []domain.Seller{{CompanyID: "twentebelt", Name: "Twentebelt", Website: "https://twentebelt.com/", Names: []string{"Oogbanden"}}, {CompanyID: "x", Name: "X", Names: []string{"Y"}}},
		Terms:   []domain.MapTerm{{Text: "draadogenband", Language: "nl", Tier: "product", UsedBy: 9}},
		Answers: 10, Named: 3, ByScope: []domain.ScopeShare{{Scope: "twente", Answers: 5, Named: 3}},
		MostNamed: []domain.CompanyShare{{CompanyID: "tribelt", Share: 0.3}},
	}
	return app.Products{
		Products:  []domain.Product{eyelink},
		Standouts: []domain.Standout{{Finding: domain.FindingCrowded, Family: "eyelink"}},
		Companies: []domain.Company{{ID: "tribelt", Name: "Tribelt B.V.", Role: domain.RoleTribelt}},
	}, c.err
}

func TestProductsAreForEveryoneSignedIn(t *testing.T) {
	a := &analyses{}
	srv := serve(t, a)
	r := callAs(t, srv, http.MethodGet, "/api/v1/products", "", "tester")
	items, _ := r.body["items"].([]any)
	if r.code != http.StatusOK || len(items) != 1 || r.body["standouts"].([]any)[0].(map[string]any)["finding"] != "crowded" {
		t.Fatalf("products = %d %v", r.code, r.body)
	}
	eye := items[0].(map[string]any)
	sellers := eye["sellers"].([]any)
	if eye["family"] != "eyelink" || sellers[0].(map[string]any)["websiteUrl"] != "https://twentebelt.com/" || sellers[1].(map[string]any)["websiteUrl"] != nil ||
		eye["terms"].([]any)[0].(map[string]any)["tier"] != "product" || eye["byScope"].([]any)[0].(map[string]any)["named"].(float64) != 3 ||
		eye["mostNamed"].([]any)[0].(map[string]any)["name"] != "Tribelt B.V." {
		t.Errorf("eyelink = %v", eye)
	}
	a.insights().err = errors.New("down")
	if r := callAs(t, srv, http.MethodGet, "/api/v1/products", "", "tester"); r.code != http.StatusInternalServerError {
		t.Errorf("a failure = %d", r.code)
	}
}

func TestStandingsAreForEveryoneSignedIn(t *testing.T) {
	a := &analyses{}
	srv := serve(t, a)
	r := callAs(t, srv, http.MethodGet, "/api/v1/standings", "", "tester")
	measures, _ := r.body["measures"].([]any)
	if r.code != http.StatusOK || len(measures) != 2 || r.body["trailing"].(float64) != 1 || r.body["runStartedAt"] != nil {
		t.Fatalf("standings = %d %v", r.code, r.body)
	}
	voice := measures[0].(map[string]any)
	leader := voice["leader"].(map[string]any)
	if voice["measure"] != "aiShareOfVoice" || voice["tribelt"].(float64) != 0.24 || voice["median"].(float64) != 0.17 ||
		leader["name"] != "Ashworth" || leader["competitorId"] != "ashworth" || voice["trails"] != true || voice["leads"] != false {
		t.Errorf("share of voice = %v", voice)
	}
	empty := measures[1].(map[string]any)
	if _, ok := empty["tribelt"]; ok || empty["leader"] != nil || empty["trails"] != false {
		t.Errorf("a measure without data = %v", empty)
	}

	a.insights().run = &domain.Run{StartedAt: time.Date(2026, 11, 2, 6, 0, 0, 0, time.UTC)}
	if r := callAs(t, srv, http.MethodGet, "/api/v1/standings", "", "tester"); r.body["runStartedAt"] != "2026-11-02T06:00:00Z" {
		t.Errorf("runStartedAt = %v", r.body["runStartedAt"])
	}
	a.insights().err = errors.New("down")
	if r := callAs(t, srv, http.MethodGet, "/api/v1/standings", "", "tester"); r.code != http.StatusInternalServerError {
		t.Errorf("a failure = %d", r.code)
	}
	if r := call(t, srv, http.MethodGet, "/api/v1/standings", "", false); r.code != http.StatusUnauthorized {
		t.Errorf("signed out = %d", r.code)
	}
}
