package httpx_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/httpx"
)

func TestWriteProblem(t *testing.T) {
	rec := httptest.NewRecorder()
	httpx.WriteProblem(rec, http.StatusTeapot, "Longer")
	if rec.Code != http.StatusTeapot || rec.Header().Get("Content-Type") != "application/problem+json" {
		t.Fatalf("%d %q", rec.Code, rec.Header().Get("Content-Type"))
	}
	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["type"] != "about:blank" || got["title"] != "I'm a teapot" || got["status"] != float64(418) || got["detail"] != "Longer" {
		t.Fatalf("problem = %v", got)
	}
}

func TestProblemOmitsAnEmptyDetail(t *testing.T) {
	if p := httpx.Problem(http.StatusNotFound, ""); p.Detail.Set || p.Title != "Not Found" || p.Status != 404 {
		t.Fatalf("problem = %+v", p)
	}
}

func TestSecurityHeaders(t *testing.T) {
	rec := httptest.NewRecorder()
	httpx.SecurityHeaders(http.NotFoundHandler()).ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil))
	for _, h := range []string{"Content-Security-Policy", "X-Content-Type-Options", "Referrer-Policy", "X-Frame-Options", "Strict-Transport-Security"} {
		if rec.Header().Get(h) == "" {
			t.Errorf("missing %s", h)
		}
	}
}

func TestSameOriginRefusesCrossSiteUnsafeRequests(t *testing.T) {
	h := httpx.SameOrigin(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	cases := []struct {
		name, method string
		headers      map[string]string
		want         int
	}{
		{"same-origin fetch", http.MethodPost, map[string]string{"Sec-Fetch-Site": "same-origin"}, http.StatusNoContent},
		{"cross-site form", http.MethodPost, map[string]string{"Sec-Fetch-Site": "cross-site"}, http.StatusForbidden},
		{"cross-site delete", http.MethodDelete, map[string]string{"Sec-Fetch-Site": "same-site"}, http.StatusForbidden},
		{"matching Origin", http.MethodPut, map[string]string{"Origin": "https://app.test"}, http.StatusNoContent},
		{"foreign Origin", http.MethodPut, map[string]string{"Origin": "https://evil.test"}, http.StatusForbidden},
		{"no browser headers", http.MethodPost, nil, http.StatusNoContent},
		{"cross-site read", http.MethodGet, map[string]string{"Sec-Fetch-Site": "cross-site"}, http.StatusNoContent},
	}
	for _, c := range cases {
		req := httptest.NewRequestWithContext(t.Context(), c.method, "https://app.test/api/v1/x", nil)
		for k, v := range c.headers {
			req.Header.Set(k, v)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != c.want {
			t.Errorf("%s: status %d, want %d", c.name, rec.Code, c.want)
		}
		if c.want == http.StatusForbidden && rec.Header().Get("Content-Type") != "application/problem+json" {
			t.Errorf("%s: not a problem: %q", c.name, rec.Header().Get("Content-Type"))
		}
	}
}

func TestSubject(t *testing.T) {
	if got := httpx.Subject(httpx.WithSubject(t.Context(), "joris")); got != "joris" {
		t.Fatalf("Subject = %q", got)
	}
	if got := httpx.Subject(t.Context()); got != "" {
		t.Fatalf("Subject without one = %q", got)
	}
}
