package httpapi_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	llm "github.com/JorisJonkers-dev/findability-explorer/internal/llm/domain"
)

func (l *llmCases) Usage(_ context.Context, id string) (llm.Usage, error) {
	if l.usageErr != nil {
		return llm.Usage{}, l.usageErr
	}
	total, limit := 25.5, 100.0
	if id == "vault-anthropic" {
		cost := 2.0
		return llm.Usage{
			Source: llm.UsageAnthropic, Message: "Anthropic's report.", OrganisationCost: &cost, Since: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
			Models: []llm.ModelTokens{{Model: "claude-haiku-5-5", InputTokens: 100, OutputTokens: 20, CacheRead: 5}},
		}, nil
	}
	return llm.Usage{
		Source: llm.UsageOpenRouter, Message: "OpenRouter's figures.", Total: &total, Daily: &total, Weekly: &total, Monthly: &total,
		Limit: &limit, LimitRemaining: &limit, LimitReset: "monthly", Free: &llm.FreeRequests{Used: 1, Limit: 50, Remaining: 49},
	}, nil
}

func TestKeyUsage(t *testing.T) {
	l := &llmCases{}
	srv := serveWith(t, l)
	path := "/api/v1/admin/llm/credentials/3f2a9c4e-7d1b-4c6a-9e8f-1a2b3c4d5e6f/usage"
	if r := callAs(t, srv, http.MethodGet, path, "", "tester"); r.code != http.StatusForbidden {
		t.Errorf("a viewer read key usage: %d", r.code)
	}
	r := callAs(t, srv, http.MethodGet, path, "", "admin")
	if r.code != http.StatusOK || r.body["source"] != "openrouter" || r.body["totalUsd"].(float64) != 25.5 || r.body["limitReset"] != "monthly" ||
		r.body["freeRequests"].(map[string]any)["remaining"].(float64) != 49 || r.body["organisationCostUsd"] != nil {
		t.Fatalf("usage = %d %v", r.code, r.body)
	}
	r = callAs(t, srv, http.MethodGet, "/api/v1/admin/llm/credentials/vault-anthropic/usage", "", "admin")
	models, _ := r.body["models"].([]any)
	if r.code != http.StatusOK || r.body["organisationCostUsd"].(float64) != 2 || len(models) != 1 || r.body["since"] == nil || r.body["totalUsd"] != nil {
		t.Fatalf("anthropic usage = %d %v", r.code, r.body)
	}
	for err, want := range map[error]int{
		llm.ErrNotFound:    http.StatusNotFound,
		llm.ErrUnreadable:  http.StatusUnprocessableEntity,
		llm.ErrProvider:    http.StatusBadGateway,
		errors.New("down"): http.StatusInternalServerError,
	} {
		l.usageErr = err
		if r := callAs(t, srv, http.MethodGet, path, "", "admin"); r.code != want {
			t.Errorf("%v: %d", err, r.code)
		}
	}
}
