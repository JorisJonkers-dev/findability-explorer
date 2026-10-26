package providers_test

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/JorisJonkers-dev/findability-explorer/internal/llm/adapters/providers"
	"github.com/JorisJonkers-dev/findability-explorer/internal/llm/domain"
)

func TestOpenRouterUsageIsTheKeysOwnSpend(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/key" || r.Header.Get("Authorization") != "Bearer sk-or-key" {
			t.Errorf("request %s %v", r.URL.Path, r.Header)
		}
		_, _ = io.WriteString(w, `{"data":{"usage":25.5,"usage_daily":1.2,"usage_weekly":5.6,"usage_monthly":24.5,"limit":100,
			"limit_remaining":75.5,"limit_reset":"monthly","free_model_daily_requests":{"used":10,"limit":50,"remaining":40}}}`)
	}))
	defer srv.Close()
	c := &providers.Connector{HTTP: srv.Client(), OpenRouterURL: srv.URL}
	u, err := c.OpenRouter(t.Context(), "sk-or-key")
	if err != nil {
		t.Fatal(err)
	}
	if u.Source != domain.UsageOpenRouter || *u.Total != 25.5 || *u.Daily != 1.2 || *u.Weekly != 5.6 || *u.Monthly != 24.5 ||
		*u.Limit != 100 || *u.LimitRemaining != 75.5 || u.LimitReset != "monthly" || *u.Free != (domain.FreeRequests{Used: 10, Limit: 50, Remaining: 40}) {
		t.Errorf("usage = %+v", u)
	}
	unlimited := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"data":{"usage":1,"limit":null,"limit_remaining":null}}`)
	}))
	defer unlimited.Close()
	u, err = (&providers.Connector{HTTP: unlimited.Client(), OpenRouterURL: unlimited.URL}).OpenRouter(t.Context(), "k")
	if err != nil || u.Limit != nil || u.Free != nil {
		t.Errorf("unlimited = %+v, %v", u, err)
	}
	if _, err := (&providers.Connector{HTTP: srv.Client(), OpenRouterURL: "http://127.0.0.1:1"}).OpenRouter(t.Context(), "k"); err == nil {
		t.Error("an unreachable OpenRouter reported usage")
	}
}

// anthropicAdmin answers like Anthropic's Admin API for the Admin key sk-ant-admin.
func anthropicAdmin(t *testing.T, seen *[]string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Api-Key") != "sk-ant-admin" || r.Header.Get("anthropic-version") == "" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = io.WriteString(w, `{"type":"error","error":{"type":"authentication_error","message":"invalid x-api-key"}}`)
			return
		}
		*seen = append(*seen, r.URL.Path+"?"+r.URL.RawQuery)
		switch r.URL.Path {
		case "/v1/organizations/api_keys":
			_, _ = io.WriteString(w, `{"data":[{"id":"apikey_other","partial_key_hint":"sk-ant-api03-ZZZ...zzzz"},
				{"id":"apikey_bad","partial_key_hint":"no hint"},{"id":"apikey_ours","partial_key_hint":"sk-ant-api03-R2D...igAA"}]}`)
		case "/v1/organizations/usage_report/messages":
			_, _ = io.WriteString(w, `{"data":[
				{"results":[{"model":"claude-haiku-5-5","uncached_input_tokens":1000,"cache_read_input_tokens":200,"output_tokens":300,
					"cache_creation":{"ephemeral_1h_input_tokens":5,"ephemeral_5m_input_tokens":10}}]},
				{"results":[{"model":"claude-haiku-5-5","uncached_input_tokens":500,"output_tokens":100},{"model":"claude-opus-5","uncached_input_tokens":7,"output_tokens":3}]}]}`)
		case "/v1/organizations/cost_report":
			_, _ = io.WriteString(w, `{"data":[{"results":[{"amount":"123.45"},{"amount":"x"}]},{"results":[{"amount":"76.55"}]}]}`)
		default:
			http.NotFound(w, r)
		}
	}))
}

func TestAnthropicUsageIsReadWithTheAdminKey(t *testing.T) {
	var seen []string
	srv := anthropicAdmin(t, &seen)
	defer srv.Close()
	c := &providers.Connector{HTTP: srv.Client(), AnthropicURL: srv.URL}
	since := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	u, err := c.Anthropic(t.Context(), "sk-ant-admin", "sk-ant-api03-R2D-secret-igAA", since)
	if err != nil {
		t.Fatal(err)
	}
	if u.Source != domain.UsageAnthropic || len(u.Models) != 2 || u.Models[0] != (domain.ModelTokens{Model: "claude-haiku-5-5", InputTokens: 1515, OutputTokens: 400, CacheRead: 200}) ||
		*u.OrganisationCost != 2 || !u.Since.Equal(since) || !strings.Contains(u.Message, "this key") {
		t.Errorf("usage = %+v", u)
	}
	if !strings.Contains(seen[1], "api_key_ids%5B%5D=apikey_ours") || !strings.Contains(seen[1], "starting_at=2026-10-01T00%3A00%3A00Z") {
		t.Errorf("report asked %v", seen[1])
	}
	org, err := c.Anthropic(t.Context(), "sk-ant-admin", "", since)
	if err != nil || len(org.Models) != 2 || !strings.Contains(org.Message, "whole organisation") {
		t.Errorf("organisation usage = %+v, %v", org, err)
	}
	stranger, err := c.Anthropic(t.Context(), "sk-ant-admin", "sk-ant-api03-elsewhere", since)
	if err != nil || len(stranger.Models) != 0 || stranger.OrganisationCost != nil || !strings.Contains(stranger.Message, "no key like this one") {
		t.Errorf("a key of another organisation = %+v, %v", stranger, err)
	}
	if _, err := c.Anthropic(t.Context(), "wrong", "k", since); err == nil || !strings.Contains(err.Error(), "invalid x-api-key") {
		t.Errorf("a wrong Admin key = %v", err)
	}
	if _, err := c.Anthropic(t.Context(), "wrong", "", since); err == nil {
		t.Error("a wrong Admin key read the report")
	}
}

func TestAnAdminKeyIsCheckedAndNeverAnswers(t *testing.T) {
	var seen []string
	srv := anthropicAdmin(t, &seen)
	defer srv.Close()
	c := &providers.Connector{HTTP: srv.Client(), AnthropicURL: srv.URL}
	models, err := c.Connect(domain.AnthropicAdmin, "sk-ant-admin").Models(t.Context())
	if err != nil || len(models) != 0 {
		t.Errorf("check = %v, %v", models, err)
	}
	if _, err := c.Connect(domain.AnthropicAdmin, "wrong").Models(t.Context()); err == nil {
		t.Error("a wrong Admin key passed the check")
	}
	if _, err := c.Connect(domain.AnthropicAdmin, "sk-ant-admin").Complete(t.Context(), prompt("m")); !errors.Is(err, domain.ErrReportsOnly) {
		t.Errorf("an Admin key answered: %v", err)
	}
	broken := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, "not json") }))
	defer broken.Close()
	if _, err := (&providers.Connector{HTTP: broken.Client(), AnthropicURL: broken.URL}).Connect(domain.AnthropicAdmin, "k").Models(t.Context()); err == nil {
		t.Error("a broken answer passed the check")
	}
	if _, err := (&providers.Connector{HTTP: broken.Client(), AnthropicURL: "http://127.0.0.1:1"}).Connect(domain.AnthropicAdmin, "k").Models(t.Context()); err == nil {
		t.Error("an unreachable Anthropic passed the check")
	}
	if _, err := (&providers.Connector{HTTP: broken.Client(), AnthropicURL: "://bad"}).Connect(domain.AnthropicAdmin, "k").Models(t.Context()); err == nil {
		t.Error("a bad address built a request")
	}
	if c := providers.New(); c.AnthropicURL != providers.AnthropicURL {
		t.Errorf("default Anthropic URL %q", c.AnthropicURL)
	}
}
