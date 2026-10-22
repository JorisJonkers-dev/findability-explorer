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
	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/httpapi"
	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/httpx"
	"github.com/JorisJonkers-dev/findability-explorer/internal/terminology/adapters/seed"
	terminologyapp "github.com/JorisJonkers-dev/findability-explorer/internal/terminology/app"
)

// analyses is a stand-in for the analysis use cases, so these tests exercise the contract.
type analyses struct {
	startErr error
	listErr  error
	user     string
	stored   domain.Analysis
}

func (a *analyses) Start(ctx context.Context, user, rawURL string, maxPages int) (domain.Analysis, error) {
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
		ID: uuid.New(), URL: "https://www.twentebelt.com/", MaxPages: 5, Status: domain.StatusDone,
		PagesCrawled: 1, CreatedAt: time.Now().UTC(), FinishedAt: time.Now().UTC(), Matching: domain.MatchingDone,
		Pages:    []domain.PageSummary{{URL: "https://www.twentebelt.com/", Title: "Twentebelt", Language: "en", WordCount: 100}},
		Keywords: []domain.Keyword{{Phrase: "metal conveyor belts", Language: "en", Score: 12.5, Pages: 1}},
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
	catalogue, err := seed.Load()
	if err != nil {
		t.Fatal(err)
	}
	h, err := httpapi.New(slog.New(slog.DiscardHandler), httpapi.Contexts{
		Terminology: terminologyapp.New(catalogue), Analysis: a,
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

func call(t *testing.T, srv *httptest.Server, method, path, body string, identified bool) response {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), method, srv.URL+path, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if identified {
		req.Header.Set(httpx.IdentityHeader, "tester")
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
}

func TestStartAnalysis(t *testing.T) {
	a := &analyses{}
	srv := serve(t, a)
	r := call(t, srv, http.MethodPost, "/api/v1/analyses", `{"url":"https://www.twentebelt.com","maxPages":3}`, true)
	if r.code != http.StatusAccepted || a.user != "tester" {
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
		errors.New("database down"): http.StatusInternalServerError,
	}
	for err, want := range cases {
		a.startErr = err
		r := call(t, srv, http.MethodPost, "/api/v1/analyses", `{"url":"https://www.twentebelt.com"}`, true)
		if r.code != want {
			t.Errorf("%v: status %d, want %d", err, r.code, want)
		}
		if want != http.StatusInternalServerError && r.body["detail"] == nil {
			t.Errorf("%v: no detail in %v", err, r.body)
		}
	}
	if r := call(t, srv, http.MethodPost, "/api/v1/analyses", `{"url":"https://x.test","maxPages":99}`, true); r.code != http.StatusBadRequest {
		t.Errorf("maxPages 99: status %d", r.code)
	}
}

func TestGetAndListAnalyses(t *testing.T) {
	a := &analyses{stored: done()}
	srv := serve(t, a)
	r := call(t, srv, http.MethodGet, "/api/v1/analyses/"+a.stored.ID.String(), "", true)
	if r.code != http.StatusOK {
		t.Fatalf("status %d: %v", r.code, r.body)
	}
	for _, key := range []string{"pages", "keywords", "knownTerms", "equivalents"} {
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
	if r := call(t, srv, http.MethodPost, "/api/v1/analyses", `{"url":`, true); r.code != http.StatusBadRequest {
		t.Errorf("broken body status %d", r.code)
	}
}
