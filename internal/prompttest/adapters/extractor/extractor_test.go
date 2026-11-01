package extractor_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	llm "github.com/JorisJonkers-dev/findability-explorer/internal/llm/domain"
	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/adapters/extractor"
	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/app"
	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain"
)

// client answers with text and the tokens given, or fails, and keeps the prompt it was sent.
type client struct {
	text   string
	err    error
	prompt llm.Prompt
}

func (c *client) Models(context.Context) ([]llm.Model, error) { return nil, nil }

func (c *client) Complete(_ context.Context, p llm.Prompt) (llm.Completion, error) {
	c.prompt = p
	return llm.Completion{Text: c.text, InputTokens: 2000, OutputTokens: 400}, c.err
}

func TestHaikuReadsAnAnswerInTheStrictFormat(t *testing.T) {
	c := &client{text: ` {"companies":[{"name":" Twentebelt ","url":"https://twentebelt.com"},{"name":"","url":""},{"name":"Tribelt","url":""}],
		"addresses":["https://twentebelt.com"],"namesTribelt":true,"citesTribelt":true,"correctness":"wrong",
		"wrongClaims":["Tribelt is in Hengelo."],"leftOut":["Tribelt makes eyelink belts to measure."]} `}
	x, usage, err := extractor.New(c, "claude-haiku-5-5").Extract(t.Context(), app.ExtractInput{
		Prompt: "Who makes eyelink belts?", Answer: "Twentebelt and Tribelt.", Citations: []string{"https://twentebelt.com"}, FactSheet: "Tribelt is in Haaksbergen.",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(x.Companies, []domain.Named{{Name: "Twentebelt", URL: "https://twentebelt.com"}, {Name: "Tribelt"}}) ||
		!x.NamesTribelt || !x.CitesTribelt || x.Correctness != domain.CorrectnessWrong || len(x.WrongClaims) != 1 || len(x.LeftOut) != 1 || len(x.Addresses) != 1 {
		t.Errorf("extracted = %+v", x)
	}
	// 2000 × $0.10 and 400 × $0.50 per million tokens: $0.0004, 400 micro-dollars.
	if usage.InputTokens != 2000 || usage.OutputTokens != 400 || usage.Cost != 400 {
		t.Errorf("usage = %+v", usage)
	}
	p := c.prompt
	if p.Model != "claude-haiku-5-5" || p.System != extractor.System || p.SchemaName != "answer_reading" || p.Schema["additionalProperties"] != false ||
		p.MaxTokens < 2000 || !strings.Contains(p.User, "Haaksbergen") || !strings.Contains(p.User, "The response:\nTwentebelt and Tribelt.") || !strings.Contains(p.User, "cited sources") {
		t.Errorf("prompt = %+v", p)
	}
}

func TestAFailedReadingStillCountsWhatItUsed(t *testing.T) {
	c := &client{err: errors.New("Anthropic is rate limiting")}
	if _, usage, err := extractor.New(c, "claude-haiku-5-5").Extract(t.Context(), app.ExtractInput{}); err == nil || usage.InputTokens != 2000 || usage.Cost != 400 {
		t.Errorf("a failed call = %v, usage %+v", err, usage)
	}
	// Read by Claude Sonnet 5.5, the reading is priced at its list price: $2 and $10 per million.
	if _, usage, _ := extractor.New(c, "claude-sonnet-5-5").Extract(t.Context(), app.ExtractInput{}); usage.Cost != 8000 {
		t.Errorf("on Sonnet, cost %d", usage.Cost)
	}
	c = &client{text: "not json"}
	if _, usage, err := extractor.New(c, "claude-haiku-5-5").Extract(t.Context(), app.ExtractInput{}); err == nil || usage.OutputTokens != 400 {
		t.Errorf("a broken format = %v, usage %+v", err, usage)
	}
}

func TestParseKeepsWithinTheLimits(t *testing.T) {
	var b strings.Builder
	b.WriteString(`{"companies":[`)
	for i := range 40 {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString(`{"name":"` + strings.Repeat("n", 300) + `","url":""}`)
	}
	b.WriteString(`],"addresses":[],"namesTribelt":false,"citesTribelt":false,"correctness":"none","wrongClaims":[]}`)
	x, err := extractor.Parse(b.String())
	if err != nil || len(x.Companies) != 30 || len([]rune(x.Companies[0].Name)) != 200 || x.Correctness != domain.CorrectnessNone {
		t.Errorf("parsed %d companies, %v", len(x.Companies), err)
	}
}

func TestAReadingSeparatesClaimsWrongAboutTribeltFromFactsLeftOut(t *testing.T) {
	s := extractor.Schema()
	props := s["properties"].(map[string]any)
	if !slices.Contains(s["required"].([]string), "leftOut") || props["leftOut"] == nil || props["wrongClaims"] == nil {
		t.Errorf("the strict format asks no left-out list: %v", s)
	}
	if !strings.Contains(extractor.System, "leftOut:") || !strings.Contains(extractor.System, "wrongClaims:") ||
		!strings.Contains(extractor.System, "not a wrong claim") {
		t.Errorf("the instructions do not tell wrong claims from facts left out:\n%s", extractor.System)
	}
	var b strings.Builder
	b.WriteString(`{"companies":[{"name":"Tribelt","url":""}],"addresses":[],"namesTribelt":true,"citesTribelt":false,
		"correctness":"partly","wrongClaims":["Tribelt is in Hengelo."],"leftOut":[`)
	for i := range 12 {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString(`"` + strings.Repeat("f", 600) + `"`)
	}
	b.WriteString(`]}`)
	x, err := extractor.Parse(b.String())
	if err != nil || !slices.Equal(x.WrongClaims, []string{"Tribelt is in Hengelo."}) || len(x.LeftOut) != 10 || len([]rune(x.LeftOut[0])) != 500 {
		t.Errorf("parsed wrong %v, %d left out, %v", x.WrongClaims, len(x.LeftOut), err)
	}
}

func TestAReplyWithoutALeftOutListStillParses(t *testing.T) {
	x, err := extractor.Parse(`{"companies":[],"addresses":[],"namesTribelt":true,"citesTribelt":false,"correctness":"wrong","wrongClaims":["Tribelt is in Hengelo."]}`)
	if err != nil || x.LeftOut != nil || len(x.WrongClaims) != 1 {
		t.Errorf("an older reply = %+v, %v", x, err)
	}
}
