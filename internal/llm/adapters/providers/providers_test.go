package providers_test

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/anthropics/anthropic-sdk-go/option"

	"github.com/JorisJonkers-dev/findability-explorer/internal/llm/adapters/providers"
	"github.com/JorisJonkers-dev/findability-explorer/internal/llm/domain"
)

func prompt(model string) domain.Prompt {
	return domain.Prompt{
		Model: model, System: "You compare.", User: "<website_text>x</website_text>", SchemaName: "tribelt_equivalents",
		Schema: map[string]any{"type": "object", "properties": map[string]any{"equivalents": map[string]any{"type": "array"}}}, MaxTokens: 8000,
	}
}

// recorder keeps every request a fake provider saw.
type recorder struct {
	mu   sync.Mutex
	reqs []seen
}

type seen struct {
	path   string
	header http.Header
	body   map[string]any
}

func (r *recorder) add(req *http.Request) {
	raw, _ := io.ReadAll(req.Body)
	var body map[string]any
	_ = json.Unmarshal(raw, &body)
	r.mu.Lock()
	r.reqs = append(r.reqs, seen{path: req.URL.Path, header: req.Header.Clone(), body: body})
	r.mu.Unlock()
}

func (r *recorder) all() []seen {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]seen(nil), r.reqs...)
}

func serve(t *testing.T, rec *recorder, h func(w http.ResponseWriter, r *http.Request)) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec.add(r)
		w.Header().Set("Content-Type", "application/json")
		h(w, r)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func connector(srv *httptest.Server) *providers.Connector {
	c := providers.New()
	c.OpenRouterURL, c.MistralURL = srv.URL, srv.URL
	c.AnthropicOptions = []option.RequestOption{option.WithBaseURL(srv.URL), option.WithMaxRetries(0)}
	return c
}

func TestNewPointsAtTheRealAPIs(t *testing.T) {
	c := providers.New()
	if c.OpenRouterURL != "https://openrouter.ai/api/v1" || c.MistralURL != "https://api.mistral.ai/v1" || c.HTTP.Timeout == 0 {
		t.Fatalf("connector = %+v", c)
	}
}

func TestAnthropicModels(t *testing.T) {
	rec := &recorder{}
	srv := serve(t, rec, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"data":[
			{"id":"claude-haiku-5-5","display_name":"Claude Haiku 5.5","lifecycle":"active","type":"model"},
			{"id":"claude-old","display_name":"Old","lifecycle":"retired","type":"model"},
			{"id":"claude-x","display_name":"","lifecycle":"active","type":"model"}],"has_more":false}`)
	})
	models, err := connector(srv).Connect(domain.Anthropic, "sk-ant").Models(t.Context())
	if err != nil || len(models) != 2 || models[0].Name != "Claude Haiku 5.5" || models[1].Name != "claude-x" {
		t.Fatalf("models = %+v, %v", models, err)
	}
	if r := rec.all()[0]; r.path != "/v1/models" || r.header.Get("X-Api-Key") != "sk-ant" {
		t.Fatalf("request %s with key %q", r.path, r.header.Get("X-Api-Key"))
	}
}

func TestAnthropicComplete(t *testing.T) {
	rec := &recorder{}
	stop := "end_turn"
	srv := serve(t, rec, func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "msg_1", "type": "message", "role": "assistant", "model": "m",
			"content":     []any{map[string]any{"type": "text", "text": `{"equivalents":`}, map[string]any{"type": "text", "text": `[]}`}},
			"stop_reason": stop, "usage": map[string]any{"input_tokens": 1500, "output_tokens": 120},
		})
	})
	client := connector(srv).Connect(domain.Anthropic, "sk-ant")
	out, err := client.Complete(t.Context(), prompt(domain.DefaultModel))
	if err != nil || out.Text != `{"equivalents":[]}` || out.InputTokens != 1500 || out.OutputTokens != 120 {
		t.Fatalf("complete = %+v, %v", out, err)
	}
	body := rec.all()[0].body
	config, _ := body["output_config"].(map[string]any)
	format, _ := config["format"].(map[string]any)
	if body["model"] != domain.DefaultModel || config["effort"] != "low" || format["type"] != "json_schema" || format["schema"] == nil {
		t.Fatalf("request = %v", body)
	}
	if _, err := client.Complete(t.Context(), prompt("claude-opus-5")); err != nil {
		t.Fatal(err)
	}
	if config, _ := rec.all()[1].body["output_config"].(map[string]any); config["effort"] != nil {
		t.Fatalf("another model got an effort: %v", config)
	}
	for reason, want := range map[string]error{"refusal": providers.ErrDeclined, "max_tokens": providers.ErrCutOff} {
		stop = reason
		if out, err := client.Complete(t.Context(), prompt("m")); !errors.Is(err, want) || out.InputTokens != 1500 {
			t.Errorf("%s: %+v, %v", reason, out, err)
		}
	}
}

func TestAnthropicErrors(t *testing.T) {
	cases := map[int]string{
		http.StatusUnauthorized:        "Anthropic refused the API key (invalid x-api-key)",
		http.StatusTooManyRequests:     "Anthropic is rate limiting; try again later (invalid x-api-key)",
		http.StatusInternalServerError: "Anthropic answered 500 (invalid x-api-key)",
	}
	for status, want := range cases {
		srv := serve(t, &recorder{}, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(status)
			_, _ = io.WriteString(w, `{"type":"error","error":{"type":"authentication_error","message":"invalid x-api-key"}}`)
		})
		client := connector(srv).Connect(domain.Anthropic, "sk-ant")
		if _, err := client.Models(t.Context()); err == nil || err.Error() != want {
			t.Errorf("models %d: %v", status, err)
		}
		if _, err := client.Complete(t.Context(), prompt("m")); err == nil || err.Error() != want {
			t.Errorf("complete %d: %v", status, err)
		}
	}
	c := providers.New()
	c.AnthropicOptions = []option.RequestOption{option.WithBaseURL("http://127.0.0.1:1"), option.WithMaxRetries(0)}
	if _, err := c.Connect(domain.Anthropic, "k").Models(t.Context()); err == nil || !strings.Contains(err.Error(), "could not be reached") {
		t.Fatalf("unreachable: %v", err)
	}
}

func TestOpenRouterChecksTheKeyThenListsModels(t *testing.T) {
	rec := &recorder{}
	srv := serve(t, rec, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/key":
			if r.Header.Get("Authorization") != "Bearer sk-or-good" {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = io.WriteString(w, `{"error":{"code":401,"message":"No auth credentials found"}}`)
				return
			}
			_, _ = io.WriteString(w, `{"data":{"label":"x"}}`)
		case "/models":
			_, _ = io.WriteString(w, `{"data":[{"id":"anthropic/claude-haiku-5-5","name":"Anthropic: Claude Haiku 5.5"},{"id":""},{"id":"x/y"}]}`)
		}
	})
	c := connector(srv)
	models, err := c.Connect(domain.OpenRouter, "sk-or-good").Models(t.Context())
	if err != nil || len(models) != 2 || models[1].Name != "x/y" {
		t.Fatalf("models = %+v, %v", models, err)
	}
	reqs := rec.all()
	if reqs[0].path != "/key" || reqs[1].path != "/models" || reqs[0].header.Get("HTTP-Referer") != providers.Referer ||
		reqs[0].header.Get("X-Title") != providers.Title {
		t.Fatalf("requests = %+v", reqs)
	}
	_, err = c.Connect(domain.OpenRouter, "sk-or-bad").Models(t.Context())
	var pe *providers.ProviderError
	if !errors.As(err, &pe) || pe.Status != http.StatusUnauthorized || err.Error() != "OpenRouter refused the API key (No auth credentials found)" {
		t.Fatalf("bad key: %v", err)
	}
}

func TestMistralComplete(t *testing.T) {
	rec := &recorder{}
	answer := `{"choices":[{"finish_reason":"stop","message":{"role":"assistant","content":[{"type":"thinking","text":"hm"},{"type":"text","text":"{\"equivalents\":[]}"}]}}],
		"usage":{"prompt_tokens":900,"completion_tokens":100}}`
	srv := serve(t, rec, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/models" {
			_, _ = io.WriteString(w, `{"object":"list","data":[{"id":"mistral-small-latest","name":"Mistral Small"}]}`)
			return
		}
		_, _ = io.WriteString(w, answer)
	})
	client := connector(srv).Connect(domain.Mistral, "mistral-key")
	if models, err := client.Models(t.Context()); err != nil || len(models) != 1 || len(rec.all()) != 1 {
		t.Fatalf("models = %+v, %v after %d requests", models, err, len(rec.all()))
	}
	out, err := client.Complete(t.Context(), prompt("mistral-small-latest"))
	if err != nil || out.Text != `{"equivalents":[]}` || out.InputTokens != 900 || out.OutputTokens != 100 {
		t.Fatalf("complete = %+v, %v", out, err)
	}
	req := rec.all()[1]
	format, _ := req.body["response_format"].(map[string]any)
	schema, _ := format["json_schema"].(map[string]any)
	if req.path != "/chat/completions" || format["type"] != "json_schema" || schema["name"] != "tribelt_equivalents" || schema["strict"] != true ||
		schema["schema"] == nil || req.body["max_tokens"] != float64(8000) || req.header.Get("Authorization") != "Bearer mistral-key" ||
		req.header.Get("HTTP-Referer") != "" {
		t.Fatalf("request = %+v", req)
	}
	answer = `{"choices":[{"finish_reason":"stop","message":{"content":"plain"}}],"usage":{"prompt_tokens":1,"completion_tokens":1}}`
	if out, _ := client.Complete(t.Context(), prompt("m")); out.Text != "plain" {
		t.Fatalf("string content = %q", out.Text)
	}
	for body, want := range map[string]error{
		`{"choices":[{"finish_reason":"length","message":{"content":"{"}}]}`:             providers.ErrCutOff,
		`{"choices":[{"finish_reason":"model_length","message":{"content":"{"}}]}`:       providers.ErrCutOff,
		`{"choices":[{"finish_reason":"stop","message":{"content":"","refusal":"no"}}]}`: providers.ErrDeclined,
		`{"choices":[{"finish_reason":"content_filter","message":{"content":""}}]}`:      providers.ErrDeclined,
	} {
		answer = body
		if _, err := client.Complete(t.Context(), prompt("m")); !errors.Is(err, want) {
			t.Errorf("%s: %v", body, err)
		}
	}
	answer = `{"choices":[],"usage":{"prompt_tokens":5,"completion_tokens":0}}`
	if out, err := client.Complete(t.Context(), prompt("m")); err == nil || out.InputTokens != 5 {
		t.Fatalf("no choice: %+v, %v", out, err)
	}
	answer = `not json`
	if _, err := client.Complete(t.Context(), prompt("m")); err == nil || !strings.Contains(err.Error(), "read Mistral's answer") {
		t.Fatalf("not JSON: %v", err)
	}
}

func TestChatFallsBackToJSONMode(t *testing.T) {
	rec := &recorder{}
	srv := serve(t, rec, func(w http.ResponseWriter, _ *http.Request) {
		if len(rec.all()) == 1 {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = io.WriteString(w, `{"error":{"message":"This model does not support structured outputs"}}`)
			return
		}
		_, _ = io.WriteString(w, `{"choices":[{"finish_reason":"stop","message":{"content":"{}"}}],"usage":{"prompt_tokens":10,"completion_tokens":2}}`)
	})
	out, err := connector(srv).Connect(domain.OpenRouter, "k").Complete(t.Context(), prompt("x/y"))
	if err != nil || out.Text != "{}" {
		t.Fatalf("fallback = %+v, %v", out, err)
	}
	second := rec.all()[1].body
	format, _ := second["response_format"].(map[string]any)
	messages, _ := second["messages"].([]any)
	system, _ := messages[0].(map[string]any)
	if format["type"] != "json_object" || !strings.Contains(system["content"].(string), `"equivalents"`) {
		t.Fatalf("fallback request = %v", second)
	}
}

func TestChatErrors(t *testing.T) {
	cases := []struct {
		status int
		body   string
		want   string
	}{
		{http.StatusPaymentRequired, `{"error":{"code":402,"message":"Insufficient credits"}}`, "Mistral says the account has no credit left (Insufficient credits)"},
		{http.StatusTooManyRequests, `{"message":"Requests rate limit exceeded"}`, "Mistral is rate limiting; try again later (Requests rate limit exceeded)"},
		{http.StatusUnauthorized, `{"detail":"Unauthorized"}`, "Mistral refused the API key (Unauthorized)"},
		{http.StatusForbidden, `{"error":"forbidden key"}`, "Mistral refused the API key (forbidden key)"},
		{http.StatusInternalServerError, `oops`, "Mistral answered 500"},
		{http.StatusBadGateway, `{"detail":[{"msg":"x"}]}`, "Mistral answered 502"},
	}
	for _, c := range cases {
		srv := serve(t, &recorder{}, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(c.status)
			_, _ = io.WriteString(w, c.body)
		})
		if _, err := connector(srv).Connect(domain.Mistral, "k").Models(t.Context()); err == nil || err.Error() != c.want {
			t.Errorf("%d: %v, want %q", c.status, err, c.want)
		}
	}
	c := providers.New()
	c.MistralURL = "http://127.0.0.1:1"
	if _, err := c.Connect(domain.Mistral, "k").Models(t.Context()); err == nil || !strings.Contains(err.Error(), "Mistral could not be reached") {
		t.Fatalf("unreachable: %v", err)
	}
	c.MistralURL = "http://bad host"
	if _, err := c.Connect(domain.Mistral, "k").Models(t.Context()); err == nil {
		t.Fatal("a malformed base URL worked")
	}
	srv := serve(t, &recorder{}, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"message":"`+strings.Repeat("long ", 100)+`"}`)
	})
	_, err := connector(srv).Connect(domain.Mistral, "k").Complete(t.Context(), prompt("m"))
	if err == nil || len([]rune(err.Error())) > 260 || !strings.HasSuffix(err.Error(), "…)") {
		t.Fatalf("a long message after the fallback: %v", err)
	}
}
