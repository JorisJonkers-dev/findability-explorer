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
	err     error
	run     *domain.Run
	tickErr error
	ticked  string
	done    bool
	actor   string
	targets map[domain.Target]float64
	planErr error
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
			{Measure: domain.MeasureSitemapCoverage},
		},
		Trailing:  1,
		Run:       c.run,
		Open:      plan().Items[:1],
		TermsFrom: domain.TermsFromScan,
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
		Checks: []domain.CheckSummary{
			{Check: domain.CheckCrawlersLetIn, Scored: true, Passing: 30, Tribelt: new(true)},
			{Check: domain.CheckProduct, Passing: 4},
		},
	}, c.err
}

func TestTheAiSearchSummaryIsForEveryoneSignedIn(t *testing.T) {
	a := &analyses{}
	srv := serve(t, a)
	r := callAs(t, srv, http.MethodGet, "/api/v1/ai-search", "", "tester")
	if r.code != http.StatusOK || r.body["goal"].(float64) != 0.4 || r.body["sites"].(float64) != 41 || r.body["providers"].(float64) != 5 {
		t.Fatalf("summary = %d %v", r.code, r.body)
	}
	checks := r.body["checks"].([]any)
	if c := checks[0].(map[string]any); r.body["readinessChecks"].(float64) != 5 || c["check"] != "crawlers" || c["scored"] != true || c["passing"].(float64) != 30 || c["tribelt"] != true {
		t.Errorf("checks = %v of %v", checks, r.body["readinessChecks"])
	}
	if c := checks[1].(map[string]any); c["scored"] != false || c["tribelt"] != nil {
		t.Errorf("a check beside = %v", c)
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
	// The products stub says nothing of where the terms come from, so the field is left out.
	if _, said := r.body["tribeltTermsFrom"]; r.code != http.StatusOK || len(items) != 1 || said || r.body["standouts"].([]any)[0].(map[string]any)["finding"] != "crowded" {
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

func (c *insightCases) Overview(context.Context) (domain.Overview, error) {
	return domain.Overview{
		Terms: domain.TermsOverview{
			Total: 387, Tribelt: 30, From: domain.TermsFromScan,
			Languages: []domain.LanguageTerms{{Language: "en", Terms: 200, Tribelt: 20}, {Language: "de", Terms: 187, Tribelt: 10}},
			Tiers:     []domain.TierTerms{{Tier: "product", Terms: 387, Tribelt: 30}},
		},
		Competitors: domain.CompetitorsOverview{Makers: 61, Families: []domain.FamilyMakers{{Family: "eyelink", Makers: 9}, {Family: "other"}}},
		AiSearch: domain.AiSearchOverview{Answers: 1260, Companies: []domain.Mentioned{
			{CompanyID: "twentebelt", Name: "Twentebelt", Role: domain.RoleCompetitor, Answers: 412, Share: 412.0 / 1260},
			{CompanyID: "tribelt", Name: "Tribelt", Role: domain.RoleTribelt},
		}},
	}, c.err
}

func TestTheOverviewIsForEveryoneSignedIn(t *testing.T) {
	a := &analyses{}
	srv := serve(t, a)
	r := callAs(t, srv, http.MethodGet, "/api/v1/overview", "", "tester")
	if r.code != http.StatusOK {
		t.Fatalf("overview = %d %v", r.code, r.body)
	}
	terms := r.body["terms"].(map[string]any)
	languages := terms["languages"].([]any)
	if terms["total"].(float64) != 387 || terms["tribelt"].(float64) != 30 || len(languages) != 2 ||
		languages[1].(map[string]any)["language"] != "de" || languages[1].(map[string]any)["tribelt"].(float64) != 10 ||
		terms["from"] != "scan" || terms["tiers"].([]any)[0].(map[string]any)["tribelt"].(float64) != 30 {
		t.Errorf("terms = %v", terms)
	}
	competitors := r.body["competitors"].(map[string]any)
	families := competitors["families"].([]any)
	if competitors["makers"].(float64) != 61 || competitors["listed"] != nil || len(families) != 2 || families[0].(map[string]any)["family"] != "eyelink" || families[1].(map[string]any)["makers"].(float64) != 0 {
		t.Errorf("competitors = %v", competitors)
	}
	ai := r.body["aiSearch"].(map[string]any)
	named := ai["companies"].([]any)
	if ai["answers"].(float64) != 1260 || len(named) != 2 || named[0].(map[string]any)["competitorId"] != "twentebelt" ||
		named[0].(map[string]any)["answers"].(float64) != 412 || named[0].(map[string]any)["share"].(float64) != 412.0/1260 || named[1].(map[string]any)["role"] != "tribelt" || named[1].(map[string]any)["name"] != "Tribelt" {
		t.Errorf("ai search = %v", ai)
	}
	a.insights().err = errors.New("down")
	if r := callAs(t, srv, http.MethodGet, "/api/v1/overview", "", "tester"); r.code != http.StatusInternalServerError {
		t.Errorf("a failure = %d", r.code)
	}
	if r := call(t, srv, http.MethodGet, "/api/v1/overview", "", false); r.code != http.StatusUnauthorized {
		t.Errorf("signed out = %d", r.code)
	}
}

func plan() app.Plan {
	at := time.Date(2026, 11, 2, 9, 0, 0, 0, time.UTC)
	share := 0.24
	return app.Plan{
		Items: []domain.PlanItem{
			{Improvement: domain.Improvements()[0], Done: true, Tick: &domain.Tick{By: "joris", At: at}, Leader: &domain.Leader{CompanyID: "ashworth", Name: "Ashworth", Value: 0.57}},
			{Improvement: domain.Improvements()[2], Auto: true},
		},
		Watched: []domain.Watched{
			{Target: domain.TargetShareOfVoice, Now: &share, Goal: 0.4},
			{Target: domain.TargetCitations, Goal: 20},
			{Target: domain.TargetTermsUsed, Now: new(45.0), Goal: 35, Met: true},
		},
		EyeOn:     []domain.EyeOn{{Why: domain.WatchTwente, Leader: domain.Leader{CompanyID: "twentebelt", Name: "Twentebelt", Value: 0.6}}},
		TermsFrom: domain.TermsFromScan,
	}
}

func (c *insightCases) Plan(context.Context) (app.Plan, error) { return plan(), c.planErr }

func (c *insightCases) Tick(_ context.Context, actor, id string, done bool) error {
	c.actor, c.ticked, c.done = actor, id, done
	return c.tickErr
}

func (c *insightCases) SetTargets(_ context.Context, actor string, targets map[domain.Target]float64) error {
	c.actor, c.targets = actor, targets
	return c.tickErr
}

func TestThePlanIsForEveryoneAndTickedByAdministrators(t *testing.T) {
	a := &analyses{}
	srv := serve(t, a)
	r := callAs(t, srv, http.MethodGet, "/api/v1/plan", "", "tester")
	items, _ := r.body["improvements"].([]any)
	if r.code != http.StatusOK || len(items) != 2 || r.body["catalogueVersion"].(float64) != 2 {
		t.Fatalf("plan = %d %v", r.code, r.body)
	}
	first := items[0].(map[string]any)
	if first["id"] != "submit-search-engines" || first["done"] != true || first["tickedBy"] != "joris" || first["leader"].(map[string]any)["name"] != "Ashworth" ||
		first["benefit"].(float64) != 4 || first["effort"].(float64) != 1 {
		t.Errorf("first = %v", first)
	}
	if second := items[1].(map[string]any); second["auto"] != true || second["benefit"].(float64) != 3 || second["effort"].(float64) != 1 || second["leader"] != nil || second["tickedAt"] != nil {
		t.Errorf("second = %v", second)
	}
	watched := r.body["watched"].([]any)
	if watched[0].(map[string]any)["now"].(float64) != 0.24 || watched[1].(map[string]any)["now"] != nil || watched[0].(map[string]any)["met"] != false ||
		watched[2].(map[string]any)["met"] != true || r.body["tribeltTermsFrom"] != "scan" ||
		watched[0].(map[string]any)["max"].(float64) != 1 || watched[2].(map[string]any)["max"] != nil || r.body["readinessChecks"].(float64) != 5 {
		t.Errorf("watched = %v", watched)
	}
	if eye := r.body["eyeOn"].([]any)[0].(map[string]any); eye["why"] != "twente" || eye["leader"].(map[string]any)["competitorId"] != "twentebelt" {
		t.Errorf("eye on = %v", eye)
	}

	path := "/api/v1/admin/plan/improvements/listings"
	if r := callAs(t, srv, http.MethodPut, path, `{"done":true}`, "tester"); r.code != http.StatusForbidden {
		t.Errorf("a viewer ticked: %d", r.code)
	}
	if r := callAs(t, srv, http.MethodPut, path, `{"done":false}`, "admin"); r.code != http.StatusOK || a.insights().ticked != "listings" || a.insights().done || a.insights().actor != "Admin" {
		t.Errorf("untick = %d, %+v", r.code, a.insights())
	}
	for err, want := range map[error]int{
		domain.ErrUnknownImprovement: http.StatusNotFound,
		domain.ErrCheckedByScan:      http.StatusUnprocessableEntity,
		errors.New("down"):           http.StatusInternalServerError,
	} {
		a.insights().tickErr = err
		if r := callAs(t, srv, http.MethodPut, path, `{"done":true}`, "admin"); r.code != want {
			t.Errorf("%v: %d", err, r.code)
		}
	}
	a.insights().tickErr = nil

	targets := "/api/v1/admin/plan/targets"
	if r := callAs(t, srv, http.MethodPut, targets, `{"targets":[{"target":"termsUsed","value":40}]}`, "tester"); r.code != http.StatusForbidden {
		t.Errorf("a viewer set targets: %d", r.code)
	}
	if r := callAs(t, srv, http.MethodPut, targets, `{"targets":[{"target":"termsUsed","value":40}]}`, "admin"); r.code != http.StatusOK || a.insights().targets[domain.TargetTermsUsed] != 40 {
		t.Errorf("set targets = %d, %v", r.code, a.insights().targets)
	}
	a.insights().tickErr = domain.ErrInvalid
	if r := callAs(t, srv, http.MethodPut, targets, `{"targets":[{"target":"termsUsed","value":40.5}]}`, "admin"); r.code != http.StatusUnprocessableEntity {
		t.Errorf("a bad target = %d", r.code)
	}
	a.insights().tickErr = errors.New("down")
	if r := callAs(t, srv, http.MethodPut, targets, `{"targets":[{"target":"termsUsed","value":40}]}`, "admin"); r.code != http.StatusInternalServerError {
		t.Errorf("a failing store = %d", r.code)
	}
	a.insights().tickErr, a.insights().planErr = nil, errors.New("down")
	for _, call := range [][3]string{{http.MethodGet, "/api/v1/plan", ""}, {http.MethodPut, path, `{"done":true}`}, {http.MethodPut, targets, `{"targets":[{"target":"citations","value":20}]}`}} {
		if r := callAs(t, srv, call[0], call[1], call[2], "admin"); r.code != http.StatusInternalServerError {
			t.Errorf("%s with a failing plan = %d", call[1], r.code)
		}
	}
}

func TestStandingsAreForEveryoneSignedIn(t *testing.T) {
	a := &analyses{}
	srv := serve(t, a)
	r := callAs(t, srv, http.MethodGet, "/api/v1/standings", "", "tester")
	measures, _ := r.body["measures"].([]any)
	if r.code != http.StatusOK || len(measures) != 2 || r.body["trailing"].(float64) != 1 || r.body["runStartedAt"] != nil || r.body["tribeltTermsFrom"] != "scan" ||
		r.body["readinessChecks"].(float64) != 5 {
		t.Fatalf("standings = %d %v", r.code, r.body)
	}
	voice := measures[0].(map[string]any)
	leader := voice["leader"].(map[string]any)
	if voice["measure"] != "aiShareOfVoice" || voice["tribelt"].(float64) != 0.24 || voice["median"].(float64) != 0.17 ||
		leader["name"] != "Ashworth" || leader["competitorId"] != "ashworth" || voice["trails"] != true || voice["leads"] != false {
		t.Errorf("share of voice = %v", voice)
	}
	if open := r.body["open"].([]any); len(open) != 1 {
		t.Errorf("open = %v", open)
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
