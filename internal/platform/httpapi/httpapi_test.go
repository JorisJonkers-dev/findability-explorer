package httpapi_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	analysisapp "github.com/JorisJonkers-dev/findability-explorer/internal/analysis/app"
	"github.com/JorisJonkers-dev/findability-explorer/internal/analysis/domain"
	"github.com/JorisJonkers-dev/findability-explorer/internal/companies/adapters/research"
	companiesapp "github.com/JorisJonkers-dev/findability-explorer/internal/companies/app"
	companiesdomain "github.com/JorisJonkers-dev/findability-explorer/internal/companies/domain"
	identityweb "github.com/JorisJonkers-dev/findability-explorer/internal/identity/adapters/web"
	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/httpapi"
	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/httpx"
	"github.com/JorisJonkers-dev/findability-explorer/internal/terminology/adapters/seed"
	terminologyapp "github.com/JorisJonkers-dev/findability-explorer/internal/terminology/app"
	terminologydomain "github.com/JorisJonkers-dev/findability-explorer/internal/terminology/domain"
)

// analyses is a stand-in for the analysis use cases, so these tests exercise the contract.
type analyses struct {
	startErr    error
	listErr     error
	user        string
	companyName string
	stored      domain.Analysis
	rescanErr   error
	profileErr  error
	compareErr  error
	profile     domain.Profile
}

// companyRepo is an in-memory store of added companies.
type companyRepo struct {
	added     []companiesdomain.Company
	overrides []companiesdomain.Override
}

func (r *companyRepo) List(context.Context) ([]companiesdomain.Company, error) { return r.added, nil }

func (r *companyRepo) Add(_ context.Context, c companiesdomain.Company) error {
	r.added = append(r.added, c)
	return nil
}

// companyScans reports one scan of Twentebelt, or fails with err.
type companyScans struct{ err error }

func (s companyScans) Stats(context.Context) (map[string]companiesdomain.Stats, error) {
	last := companiesdomain.Scan{ID: uuid.New(), URL: "https://www.twentebelt.com/", Status: "done", MaxPages: 5, PagesCrawled: 5, CreatedAt: time.Now(), FinishedAt: time.Now()}
	return map[string]companiesdomain.Stats{"twentebelt": {Count: 1, Last: &last}}, s.err
}

func (s companyScans) ByCompany(context.Context, string, int) ([]companiesdomain.Scan, error) {
	return []companiesdomain.Scan{{ID: uuid.New(), URL: "https://www.twentebelt.com/", Status: "failed", Failure: "No page could be fetched.", MaxPages: 5, CreatedAt: time.Now()}}, s.err
}

func (a *analyses) Start(ctx context.Context, user, rawURL string, maxPages int, companyName string) (domain.Analysis, error) {
	a.companyName = companyName
	a.user = httpx.Subject(ctx)
	if a.startErr != nil {
		return domain.Analysis{}, a.startErr
	}
	return domain.Analysis{
		ID: uuid.New(), URL: rawURL, MaxPages: maxPages, Status: domain.StatusQueued,
		CreatedBy: user, CreatedAt: time.Now().UTC(), Matching: domain.MatchingPending,
	}, nil
}

func (a *analyses) Get(_ context.Context, id uuid.UUID) (domain.Analysis, error) {
	if id != a.stored.ID {
		return domain.Analysis{}, domain.ErrNotFound
	}
	return a.stored, nil
}

func (a *analyses) List(context.Context, int) ([]domain.Analysis, error) {
	if a.listErr != nil {
		return nil, a.listErr
	}
	return []domain.Analysis{a.stored}, nil
}

func done() domain.Analysis {
	return domain.Analysis{
		ID: uuid.New(), URL: "https://www.twentebelt.com/", CompetitorID: "twentebelt", MaxPages: 5, Status: domain.StatusDone,
		PagesCrawled: 1, CreatedAt: time.Now().UTC(), FinishedAt: time.Now().UTC(), Matching: domain.MatchingDone,
		Pages: []domain.PageSummary{{URL: "https://www.twentebelt.com/", Title: "Twentebelt", Language: "en", WordCount: 100}},
		Keywords: []domain.Keyword{
			{Phrase: "metal conveyor belts", Language: "en", Score: 12.5, Pages: 1, Classification: domain.Classification{Tier: "broad", Family: "all", Closest: "metal conveyor belt(s)", InMap: true}},
			{Phrase: "belt options", Language: "en", Score: 2, Pages: 1, Classification: domain.Classification{Tier: "broad"}},
		},
		KnownTerms: []domain.KnownTerm{{
			GlossaryTerm: domain.GlossaryTerm{Text: "balanced weave belt", Language: "en", Family: "open_spiral", Tier: "product"},
			Occurrences:  4,
		}},
		Equivalents: []domain.Equivalent{{
			Product: "TwenteFlex", Family: "tri_flex_turn_curve", Reason: "spiral",
			SearchTerms: []domain.SearchTerm{{Term: "spiral belt", Language: "en"}},
		}},
	}
}

func serve(t *testing.T, a *analyses) *httptest.Server {
	t.Helper()
	return serveAll(t, a, &llmCases{})
}

func serveWith(t *testing.T, l *llmCases) *httptest.Server {
	t.Helper()
	return serveAll(t, &analyses{}, l)
}

func serveAll(t *testing.T, a *analyses, l *llmCases) *httptest.Server {
	t.Helper()
	catalogue, err := seed.Load()
	if err != nil {
		t.Fatal(err)
	}
	terminology := terminologyapp.New(catalogue, &reviewStore{}, func(context.Context) ([]terminologydomain.Observation, error) {
		return observations(), nil
	}, time.Now)
	companies := companiesapp.New(
		func() []companiesdomain.Company { return research.Companies(terminology.Companies()) },
		&companyRepo{}, companyScans{}, time.Now,
	)
	h, err := httpapi.New(slog.New(slog.DiscardHandler), httpapi.Contexts{
		Terminology: terminology, Analysis: a, Companies: companies, LLM: l, Auth: auth{},
	})
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return srv
}

type response struct {
	code int
	body map[string]any
}

// call sends a request as the account "tester" when identified, else without a session.
func call(t *testing.T, srv *httptest.Server, method, path, body string, identified bool) response {
	t.Helper()
	cookie := ""
	if identified {
		cookie = "tester"
	}
	return callAs(t, srv, method, path, body, cookie)
}

// callAs sends a request with the session cookie set to cookie, unless it is empty.
func callAs(t *testing.T, srv *httptest.Server, method, path, body, cookie string) response {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), method, srv.URL+path, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if cookie != "" {
		req.AddCookie(&http.Cookie{Name: identityweb.SessionCookie, Value: cookie}) //nolint:gosec // a request cookie
	}
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	out := response{code: resp.StatusCode}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &out.body); err != nil {
			t.Fatalf("%s %s: body %q is not JSON: %v", method, path, raw, err)
		}
	}
	return out
}

func TestSearchTerms(t *testing.T) {
	srv := serve(t, &analyses{})
	r := call(t, srv, http.MethodGet, "/api/v1/terms?q=twenteflex&limit=5", "", true)
	if r.code != http.StatusOK {
		t.Fatalf("status %d: %v", r.code, r.body)
	}
	items, _ := r.body["items"].([]any)
	if len(items) == 0 || r.body["total"].(float64) < 1 {
		t.Fatalf("no TwenteFlex match: %v", r.body)
	}
	facets, _ := r.body["facets"].(map[string]any)
	if langs, _ := facets["languages"].([]any); len(langs) != 10 {
		t.Fatalf("language facets = %v", facets["languages"])
	}
	filtered := call(t, srv, http.MethodGet, "/api/v1/terms?language=de&tier=spec&family=spec_all&user=pelger", "", true)
	if filtered.code != http.StatusOK {
		t.Fatalf("filtered status %d", filtered.code)
	}
	for _, item := range filtered.body["items"].([]any) {
		term := item.(map[string]any)
		if term["language"] != "de" || term["tier"] != "spec" {
			t.Fatalf("filter not applied: %v", term)
		}
	}
	if bad := call(t, srv, http.MethodGet, "/api/v1/terms?language=xx", "", true); bad.code != http.StatusBadRequest {
		t.Fatalf("unknown language status %d", bad.code)
	}
	if anon := call(t, srv, http.MethodGet, "/api/v1/terms", "", false); anon.code != http.StatusUnauthorized {
		t.Fatalf("anonymous status %d", anon.code)
	}
}

func TestListCompetitors(t *testing.T) {
	r := call(t, serve(t, &analyses{}), http.MethodGet, "/api/v1/competitors", "", true)
	items, _ := r.body["items"].([]any)
	if r.code != http.StatusOK || len(items) != 28 {
		t.Fatalf("status %d, %d companies", r.code, len(items))
	}
	last := items[len(items)-1].(map[string]any)
	if last["groupCompany"] != true {
		t.Fatalf("group companies are not last: %v", last)
	}
	for _, item := range items {
		c := item.(map[string]any)
		if c["id"] != "twentebelt" {
			continue
		}
		if c["websiteUrl"] != "https://www.twentebelt.com/" || c["host"] != "twentebelt.com" || c["source"] != "research" ||
			c["scans"].(float64) != 1 || c["lastScan"].(map[string]any)["competitorId"] != "twentebelt" {
			t.Fatalf("Twentebelt = %v", c)
		}
		return
	}
	t.Fatal("no Twentebelt")
}

func TestGetAndAddCompetitor(t *testing.T) {
	srv := serve(t, &analyses{})
	r := call(t, srv, http.MethodGet, "/api/v1/competitors/twentebelt", "", true)
	scans, _ := r.body["scans"].([]any)
	if r.code != http.StatusOK || r.body["competitor"].(map[string]any)["name"] != "Twentebelt" || len(scans) != 1 ||
		scans[0].(map[string]any)["failure"] == nil {
		t.Fatalf("get = %d %v", r.code, r.body)
	}
	if r := call(t, srv, http.MethodGet, "/api/v1/competitors/nobody", "", true); r.code != http.StatusNotFound {
		t.Errorf("unknown company status %d", r.code)
	}
	if r := call(t, srv, http.MethodGet, "/api/v1/competitors/Not_An_Id", "", true); r.code != http.StatusBadRequest {
		t.Errorf("bad id status %d", r.code)
	}
	if r := call(t, srv, http.MethodPost, "/api/v1/competitors", `{"name":"Example Belts","url":"https://www.example-belts.com"}`, true); r.code != http.StatusForbidden {
		t.Fatalf("a non-admin adding a company: status %d", r.code)
	}
	added := callAs(t, srv, http.MethodPost, "/api/v1/competitors", `{"name":"Example Belts","url":"https://www.example-belts.com"}`, "admin")
	if added.code != http.StatusCreated || added.body["id"] != "example-belts" || added.body["source"] != "added" || added.body["addedAt"] == nil {
		t.Fatalf("add = %d %v", added.code, added.body)
	}
	for body, want := range map[string]int{
		`{"name":"Copy","url":"https://twentebelt.com"}`: http.StatusConflict,
		`{"name":"Local","url":"https://localhost/"}`:    http.StatusUnprocessableEntity,
		`{"name":"   ","url":"https://fine.example/"}`:   http.StatusUnprocessableEntity,
	} {
		r := callAs(t, srv, http.MethodPost, "/api/v1/competitors", body, "admin")
		if r.code != want || r.body["detail"] == nil {
			t.Errorf("add %s: status %d, want %d (%v)", body, r.code, want, r.body)
		}
	}
	list := call(t, srv, http.MethodGet, "/api/v1/competitors", "", true)
	if items := list.body["items"].([]any); len(items) != 29 {
		t.Errorf("after adding: %d companies", len(items))
	}
}

func TestCompetitorsFailures(t *testing.T) {
	catalogue, err := seed.Load()
	if err != nil {
		t.Fatal(err)
	}
	terminology := terminologyapp.New(catalogue, &reviewStore{}, func(context.Context) ([]terminologydomain.Observation, error) {
		return nil, errors.New("database down")
	}, time.Now)
	companies := companiesapp.New(
		func() []companiesdomain.Company { return research.Companies(terminology.Companies()) },
		&companyRepo{}, companyScans{err: errors.New("database down")}, time.Now,
	)
	h, err := httpapi.New(slog.New(slog.DiscardHandler), httpapi.Contexts{
		Terminology: terminology, Analysis: &analyses{}, Companies: companies, LLM: &llmCases{}, Auth: auth{},
	})
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(h)
	defer srv.Close()
	for _, path := range []string{"/api/v1/competitors", "/api/v1/competitors/twentebelt", "/api/v1/admin/term-candidates"} {
		if r := callAs(t, srv, http.MethodGet, path, "", "admin"); r.code != http.StatusInternalServerError {
			t.Errorf("%s with failing scans: status %d", path, r.code)
		}
	}
}

func TestStartAnalysis(t *testing.T) {
	a := &analyses{}
	srv := serve(t, a)
	if r := call(t, srv, http.MethodPost, "/api/v1/analyses", `{"url":"https://www.twentebelt.com"}`, true); r.code != http.StatusForbidden {
		t.Fatalf("a non-admin starting a scan: status %d", r.code)
	}
	r := callAs(t, srv, http.MethodPost, "/api/v1/analyses", `{"url":"https://www.twentebelt.com","maxPages":3,"companyName":"Twente"}`, "admin")
	if r.code != http.StatusAccepted || a.user != "admin-1" || a.companyName != "Twente" {
		t.Fatalf("status %d, user %q: %v", r.code, a.user, r.body)
	}
	summary := r.body["summary"].(map[string]any)
	if summary["status"] != "queued" || summary["maxPages"].(float64) != 3 {
		t.Fatalf("summary = %v", summary)
	}
	cases := map[error]int{
		domain.ErrInvalidTarget:     http.StatusUnprocessableEntity,
		domain.ErrInvalidMaxPages:   http.StatusUnprocessableEntity,
		analysisapp.ErrRateLimited:  http.StatusTooManyRequests,
		analysisapp.ErrBusy:         http.StatusTooManyRequests,
		analysisapp.ErrScanActive:   http.StatusConflict,
		errors.New("database down"): http.StatusInternalServerError,
	}
	for err, want := range cases {
		a.startErr = err
		r := callAs(t, srv, http.MethodPost, "/api/v1/analyses", `{"url":"https://www.twentebelt.com"}`, "admin")
		if r.code != want {
			t.Errorf("%v: status %d, want %d", err, r.code, want)
		}
		if want != http.StatusInternalServerError && r.body["detail"] == nil {
			t.Errorf("%v: no detail in %v", err, r.body)
		}
	}
	if r := callAs(t, srv, http.MethodPost, "/api/v1/analyses", `{"url":"https://x.test","maxPages":101}`, "admin"); r.code != http.StatusBadRequest {
		t.Errorf("maxPages 101: status %d", r.code)
	}
}

func TestGetAndListAnalyses(t *testing.T) {
	a := &analyses{stored: done()}
	srv := serve(t, a)
	r := call(t, srv, http.MethodGet, "/api/v1/analyses/"+a.stored.ID.String(), "", true)
	if r.code != http.StatusOK {
		t.Fatalf("status %d: %v", r.code, r.body)
	}
	if kws := r.body["keywords"].([]any); len(kws) != 2 || kws[0].(map[string]any)["closestTerm"] != "metal conveyor belt(s)" ||
		kws[0].(map[string]any)["inMap"] != true || r.body["summary"].(map[string]any)["competitorId"] != "twentebelt" {
		t.Errorf("keywords = %v", r.body["keywords"])
	}
	for _, key := range []string{"pages", "knownTerms", "equivalents"} {
		if list, _ := r.body[key].([]any); len(list) != 1 {
			t.Errorf("%s = %v", key, r.body[key])
		}
	}
	if r.body["summary"].(map[string]any)["finishedAt"] == nil || r.body["matching"] != "done" {
		t.Errorf("analysis = %v", r.body)
	}
	if r := call(t, srv, http.MethodGet, "/api/v1/analyses/"+uuid.NewString(), "", true); r.code != http.StatusNotFound {
		t.Errorf("unknown id status %d", r.code)
	}
	if r := call(t, srv, http.MethodGet, "/api/v1/analyses/not-a-uuid", "", true); r.code != http.StatusBadRequest {
		t.Errorf("bad id status %d", r.code)
	}

	failed := a.stored
	failed.Status, failed.Failure = domain.StatusFailed, "No page could be fetched."
	a.stored = failed
	list := call(t, srv, http.MethodGet, "/api/v1/analyses?limit=5", "", true)
	items, _ := list.body["items"].([]any)
	if list.code != http.StatusOK || len(items) != 1 || items[0].(map[string]any)["failure"] != "No page could be fetched." {
		t.Fatalf("list = %d %v", list.code, list.body)
	}
	a.listErr = errors.New("database down")
	if r := call(t, srv, http.MethodGet, "/api/v1/analyses", "", true); r.code != http.StatusInternalServerError {
		t.Errorf("list failure status %d", r.code)
	}
}

func TestUnknownPathsAndMethods(t *testing.T) {
	srv := serve(t, &analyses{})
	if r := call(t, srv, http.MethodGet, "/api/v1/nothing", "", true); r.code != http.StatusNotFound {
		t.Errorf("unknown path status %d", r.code)
	}
	if r := call(t, srv, http.MethodDelete, "/api/v1/terms", "", true); r.code != http.StatusMethodNotAllowed {
		t.Errorf("wrong method status %d", r.code)
	}
	if r := callAs(t, srv, http.MethodPost, "/api/v1/analyses", `{"url":`, "admin"); r.code != http.StatusBadRequest {
		t.Errorf("broken body status %d", r.code)
	}
}
