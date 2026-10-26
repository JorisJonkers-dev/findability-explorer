package matcher_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/JorisJonkers-dev/findability-explorer/internal/analysis/adapters/matcher"
	"github.com/JorisJonkers-dev/findability-explorer/internal/analysis/domain"
	llm "github.com/JorisJonkers-dev/findability-explorer/internal/llm/domain"
)

// client answers every prompt with text, or fails with err, and records the prompt.
type client struct {
	text string
	err  error
	got  llm.Prompt
}

func (c *client) Models(context.Context) ([]llm.Model, error) { return nil, nil }

func (c *client) Complete(_ context.Context, p llm.Prompt) (llm.Completion, error) {
	c.got = p
	return llm.Completion{Text: c.text, InputTokens: 1500, OutputTokens: 120}, c.err
}

const answer = `{"equivalents":[{"product":"TwenteFlex spiral belts","family":"tri_flex_turn_curve",
"reason":"A curve-going spiral belt.","searchTerms":[{"term":"spiral belt","language":"en"},{"term":"nastro a spirale","language":"it"}]}]}`

func TestMatchSendsAStructuredPromptAndReadsTheAnswer(t *testing.T) {
	for name, text := range map[string]string{
		"plain":  answer,
		"fenced": "```json\n" + answer + "\n```",
		"bare":   "```\n" + answer + "```",
	} {
		t.Run(name, func(t *testing.T) {
			c := &client{text: text}
			eqs, usage, err := matcher.New(c, "model-1").Match(t.Context(), "PAGE https://x.test/\nSpiral belts", true, domain.TribeltFamilies())
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
			p := c.got
			if p.Model != "model-1" || p.SchemaName == "" || p.Schema["type"] != "object" || p.MaxTokens == 0 ||
				!strings.Contains(p.System, "Tribelt") {
				t.Errorf("prompt = %+v", p)
			}
			if !strings.Contains(p.User, "website_text") || !strings.Contains(p.User, "cut off") || !strings.Contains(p.User, "tri_flex_turn_curve") {
				t.Errorf("user prompt = %s", p.User)
			}
		})
	}
}

func TestMatchFailures(t *testing.T) {
	boom := errors.New("Anthropic refused the API key")
	cases := map[string]struct {
		client *client
		want   string
	}{
		"provider fails": {&client{err: boom}, "refused the API key"},
		"not JSON":       {&client{text: "Sure! Here are"}, "read the model's answer"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			_, usage, err := matcher.New(c.client, "m").Match(t.Context(), "text", false, domain.TribeltFamilies())
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("Match error = %v, want %q", err, c.want)
			}
			if usage.Total() != 1620 {
				t.Fatalf("usage of a failed match = %+v", usage)
			}
		})
	}
}
