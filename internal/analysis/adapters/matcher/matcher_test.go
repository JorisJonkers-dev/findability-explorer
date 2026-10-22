package matcher_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/anthropics/anthropic-sdk-go/option"

	"github.com/JorisJonkers-dev/findability-explorer/internal/analysis/adapters/matcher"
	"github.com/JorisJonkers-dev/findability-explorer/internal/analysis/domain"
)

// fakeAPI answers POST /v1/messages with status and a message whose text is text and whose stop
// reason is stop, and records the request body.
func fakeAPI(t *testing.T, status int, stop, text string, got *map[string]any) *matcher.Matcher {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" || r.Header.Get("X-Api-Key") != "test-key" {
			t.Errorf("request %s with key %q", r.URL.Path, r.Header.Get("X-Api-Key"))
		}
		body, _ := io.ReadAll(r.Body)
		if got != nil {
			_ = json.Unmarshal(body, got)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if status != http.StatusOK {
			_, _ = w.Write([]byte(`{"type":"error","error":{"type":"api_error","message":"no"}}`))
			return
		}
		msg := map[string]any{
			"id": "msg_1", "type": "message", "role": "assistant", "model": matcher.Model,
			"content":     []any{map[string]any{"type": "text", "text": text}},
			"stop_reason": stop,
			"usage":       map[string]any{"input_tokens": 1500, "output_tokens": 120},
		}
		_ = json.NewEncoder(w).Encode(msg)
	}))
	t.Cleanup(srv.Close)
	return matcher.New("test-key", option.WithBaseURL(srv.URL), option.WithMaxRetries(0))
}

const answer = `{"equivalents":[{"product":"TwenteFlex spiral belts","family":"tri_flex_turn_curve",
"reason":"A curve-going spiral belt.","searchTerms":[{"term":"spiral belt","language":"en"}]}]}`

func TestMatchSendsAStructuredRequestAndReadsTheAnswer(t *testing.T) {
	var req map[string]any
	m := fakeAPI(t, http.StatusOK, "end_turn", answer, &req)
	eqs, usage, err := m.Match(t.Context(), "PAGE https://x.test/\nSpiral belts", true, domain.TribeltFamilies())
	if err != nil {
		t.Fatal(err)
	}
	if len(eqs) != 1 || eqs[0].Product != "TwenteFlex spiral belts" || eqs[0].Family != "tri_flex_turn_curve" ||
		len(eqs[0].SearchTerms) != 1 || eqs[0].SearchTerms[0].Language != "en" {
		t.Fatalf("equivalents = %+v", eqs)
	}
	if usage.InputTokens != 1500 || usage.OutputTokens != 120 {
		t.Fatalf("usage = %+v", usage)
	}
	if req["model"] != matcher.Model {
		t.Errorf("model = %v", req["model"])
	}
	config, _ := req["output_config"].(map[string]any)
	format, _ := config["format"].(map[string]any)
	if config["effort"] != "low" || format["type"] != "json_schema" || format["schema"] == nil {
		t.Errorf("output_config = %v", config)
	}
	raw, _ := json.Marshal(req["messages"])
	if !strings.Contains(string(raw), "website_text") || !strings.Contains(string(raw), "cut off") ||
		!strings.Contains(string(raw), "tri_flex_turn_curve") {
		t.Errorf("prompt = %s", raw)
	}
}

func TestMatchFailures(t *testing.T) {
	cases := map[string]struct {
		status int
		stop   string
		text   string
		want   string
	}{
		"refusal":      {http.StatusOK, "refusal", "", "declined"},
		"cut off":      {http.StatusOK, "max_tokens", `{"equi`, "cut off"},
		"not JSON":     {http.StatusOK, "end_turn", "Sure! Here are", "read the model's answer"},
		"bad key":      {http.StatusUnauthorized, "", "", "refused the API key"},
		"rate limited": {http.StatusTooManyRequests, "", "", "rate limited"},
		"server error": {http.StatusInternalServerError, "", "", "answered 500"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			m := fakeAPI(t, c.status, c.stop, c.text, nil)
			if _, _, err := m.Match(t.Context(), "text", false, domain.TribeltFamilies()); err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("Match error = %v, want %q", err, c.want)
			}
		})
	}
}

func TestMatchWhenTheAPIIsUnreachable(t *testing.T) {
	m := matcher.New("test-key", option.WithBaseURL("http://127.0.0.1:1"), option.WithMaxRetries(0))
	if _, _, err := m.Match(t.Context(), "text", false, domain.TribeltFamilies()); err == nil || !strings.Contains(err.Error(), "could not be reached") {
		t.Fatalf("Match error = %v", err)
	}
}
