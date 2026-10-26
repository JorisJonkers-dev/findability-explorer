package openrouter_test

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/adapters/openrouter"
	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/app"
	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain"
)

// server answers like OpenRouter, recording each request body; reply writes the chat answer.
func server(t *testing.T, reply func(w http.ResponseWriter, body map[string]any)) (*httptest.Server, *[]map[string]any) {
	t.Helper()
	var bodies []map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer sk-or-test" || r.Header.Get("X-Title") != openrouter.Title {
			t.Errorf("headers %v", r.Header)
		}
		switch r.URL.Path {
		case "/models":
			_, _ = io.WriteString(w, `{"data":[
				{"id":"openai/gpt-oss-20b","name":"gpt-oss","architecture":{"input_modalities":["text"],"output_modalities":["text"]},"pricing":{"prompt":"0.00000003","completion":"0.00000014"}},
				{"id":"openrouter/auto","architecture":{},"pricing":{"prompt":"-1","completion":"-1"}},
				{"id":"broken","pricing":{"prompt":"x","completion":"0"}},
				{"id":"","pricing":{"prompt":"0","completion":"0"}}]}`)
		case "/chat/completions":
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			bodies = append(bodies, body)
			reply(w, body)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv, &bodies
}

func TestModelsReadsThePriceList(t *testing.T) {
	srv, _ := server(t, nil)
	models, err := openrouter.New(srv.URL, "sk-or-test", nil).Models(t.Context())
	if err != nil || len(models) != 2 {
		t.Fatalf("models = %+v, %v", models, err)
	}
	if m := models[0]; m.ID != "openai/gpt-oss-20b" || m.PromptPrice != 0.00000003 || !slices.Equal(m.OutputModalities, []string{"text"}) {
		t.Errorf("model = %+v", m)
	}
	if models[1].PromptPrice != -1 {
		t.Errorf("router = %+v", models[1])
	}
}

func TestAskReadsTheAnswerItsCitationsAndCost(t *testing.T) {
	srv, bodies := server(t, func(w http.ResponseWriter, _ map[string]any) {
		_, _ = io.WriteString(w, `{"choices":[{"message":{"content":"Twentebelt and Tribelt.","annotations":[
			{"type":"url_citation","url_citation":{"url":"https://www.tribelt.nl/"}},{"type":"other"}]}}],
			"usage":{"prompt_tokens":12,"completion_tokens":40,"cost":0.0123456}}`)
	})
	c := openrouter.New(srv.URL, "sk-or-test", nil)
	reply, err := c.Ask(t.Context(), "openai/gpt-oss-20b", "Who makes eyelink belts?", true)
	if err != nil {
		t.Fatal(err)
	}
	if reply.Text != "Twentebelt and Tribelt." || !slices.Equal(reply.Citations, []string{"https://www.tribelt.nl/"}) ||
		reply.InputTokens != 12 || reply.OutputTokens != 40 || reply.Cost != 12346 {
		t.Errorf("reply = %+v", reply)
	}
	sent := (*bodies)[0]
	if sent["model"] != "openai/gpt-oss-20b" || sent["plugins"].([]any)[0].(map[string]any)["id"] != "web" || sent["response_format"] != nil {
		t.Errorf("request = %v", sent)
	}
	if _, err := c.Ask(t.Context(), "m", "q", false); err != nil || (*bodies)[1]["plugins"] != nil {
		t.Errorf("a plain request carries plugins: %v", (*bodies)[1])
	}
}

func TestAskTellsARateLimitFromOtherFailures(t *testing.T) {
	cases := map[string]struct {
		status  int
		body    string
		limited bool
		want    string
	}{
		"429":                {429, `{"error":{"message":"Rate limit exceeded: free-models-per-min"}}`, true, "free-models-per-min"},
		"429 in the body":    {200, `{"error":{"code":429,"message":"slow down"}}`, true, "slow down"},
		"429 in a choice":    {200, `{"choices":[{"error":{"code":"429","message":"upstream limit"}}]}`, true, "upstream limit"},
		"an error in a body": {200, `{"error":{"code":502,"message":"bad gateway"}}`, false, "bad gateway"},
		"no choice":          {200, `{"choices":[]}`, false, "without a choice"},
		"a 500":              {500, `not json`, false, "answered 500: not json"},
		"not json":           {200, `not json`, false, "read OpenRouter's answer"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			srv, _ := server(t, func(w http.ResponseWriter, _ map[string]any) {
				w.WriteHeader(c.status)
				_, _ = io.WriteString(w, c.body)
			})
			_, err := openrouter.New(srv.URL, "sk-or-test", nil).Ask(t.Context(), "m", "q", false)
			if errors.Is(err, domain.ErrRateLimited) != c.limited || err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("err = %v", err)
			}
		})
	}
	if _, err := openrouter.New("http://127.0.0.1:1", "k", nil).Ask(t.Context(), "m", "q", false); err == nil || !strings.Contains(err.Error(), "could not be reached") {
		t.Errorf("unreachable = %v", err)
	}
	if _, err := openrouter.New("://bad", "k", nil).Models(t.Context()); err == nil {
		t.Error("a bad address built a request")
	}
}

func TestExtractReadsTheStrictFormat(t *testing.T) {
	content := `{"companies":[{"name":" Twentebelt ","url":"https://twentebelt.com"},{"name":"","url":""},{"name":"Tribelt","url":""}],
		"addresses":["https://twentebelt.com"],"namesTribelt":true,"citesTribelt":false,"correctness":"partly",
		"wrongClaims":["Tribelt is in Hengelo."]}`
	srv, bodies := server(t, func(w http.ResponseWriter, _ map[string]any) {
		raw, _ := json.Marshal(map[string]any{
			"choices": []any{map[string]any{"message": map[string]any{"content": content}}},
			"usage":   map[string]any{"prompt_tokens": 300, "completion_tokens": 50, "cost": 0.00005},
		})
		_, _ = w.Write(raw)
	})
	c := openrouter.New(srv.URL, "sk-or-test", nil)
	x, usage, err := c.Extract(t.Context(), domain.ExtractorModel, app.ExtractInput{
		Prompt: "Who makes eyelink belts?", Answer: "Twentebelt and Tribelt.", Citations: []string{"https://twentebelt.com"}, FactSheet: "Tribelt is in Haaksbergen.",
	})
	if err != nil {
		t.Fatal(err)
	}
	want := domain.Extracted{
		Companies:    []domain.Named{{Name: "Twentebelt", URL: "https://twentebelt.com"}, {Name: "Tribelt"}},
		Addresses:    []string{"https://twentebelt.com"},
		NamesTribelt: true, Correctness: domain.CorrectnessPartly, WrongClaims: []string{"Tribelt is in Hengelo."},
	}
	if !slices.Equal(x.Companies, want.Companies) || !slices.Equal(x.Addresses, want.Addresses) || x.NamesTribelt != want.NamesTribelt ||
		x.Correctness != want.Correctness || !slices.Equal(x.WrongClaims, want.WrongClaims) {
		t.Errorf("extracted = %+v", x)
	}
	if usage.InputTokens != 300 || usage.Cost != 50 {
		t.Errorf("usage = %+v", usage)
	}
	sent := (*bodies)[0]
	format := sent["response_format"].(map[string]any)["json_schema"].(map[string]any)
	user := sent["messages"].([]any)[1].(map[string]any)["content"].(string)
	if format["strict"] != true || !strings.Contains(user, "Haaksbergen") || !strings.Contains(user, "cited sources") {
		t.Errorf("request = %v", sent)
	}
	content = `{"companies":[],"addresses":[],"namesTribelt":false,"citesTribelt":false,"correctness":"none","wrongClaims":[]}`
	if x, _, err := c.Extract(t.Context(), "m", app.ExtractInput{}); err != nil || x.Correctness != domain.CorrectnessNone {
		t.Errorf("none = %+v, %v", x, err)
	}
	content = `not json`
	if _, usage, err := c.Extract(t.Context(), "m", app.ExtractInput{}); err == nil || usage.InputTokens != 300 {
		t.Errorf("a broken format = %v, usage %+v", err, usage)
	}
}

func TestExtractReportsAFailedCall(t *testing.T) {
	srv, _ := server(t, func(w http.ResponseWriter, _ map[string]any) {
		w.WriteHeader(http.StatusTooManyRequests)
	})
	if _, _, err := openrouter.New(srv.URL, "sk-or-test", nil).Extract(t.Context(), "m", app.ExtractInput{}); !errors.Is(err, domain.ErrRateLimited) {
		t.Errorf("err = %v", err)
	}
}
