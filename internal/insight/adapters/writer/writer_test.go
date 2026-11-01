package writer_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/JorisJonkers-dev/findability-explorer/internal/insight/adapters/writer"
	"github.com/JorisJonkers-dev/findability-explorer/internal/insight/domain"
	llm "github.com/JorisJonkers-dev/findability-explorer/internal/llm/domain"
)

// client answers with text, or fails, and keeps the prompt it was sent.
type client struct {
	text   string
	err    error
	prompt llm.Prompt
}

func (c *client) Models(context.Context) ([]llm.Model, error) { return nil, nil }

func (c *client) Complete(_ context.Context, p llm.Prompt) (llm.Completion, error) {
	c.prompt = p
	return llm.Completion{Text: c.text, InputTokens: 900, OutputTokens: 90}, c.err
}

func gaps() []domain.Gap {
	return []domain.Gap{{
		Key: "term:nl:spiraalband", Kind: domain.GapTerm, Title: `Use "Spiraalband" on tribelt.nl`, Language: "nl", Family: "open_spiral",
		Specifics: []string{"2 competitors use it."}, Benefit: 4, Effort: 2,
	}}
}

func pages() []domain.Page {
	return []domain.Page{{URL: "https://www.tribelt.nl/spiraalbanden", Title: "Spiraalbanden", Language: "nl", Family: "open_spiral", Text: strings.Repeat("Spiraal. ", 400)}}
}

func TestTheModelWritesARecommendationPerGap(t *testing.T) {
	c := &client{text: "```json\n" + `{"recommendations":[{"key":"term:nl:spiraalband","title":"Use it","where":"https://www.tribelt.nl/spiraalbanden","why":"Ashworth does.","steps":["Add it."],"doneWhen":"Found.","benefit":3,"effort":2,"scoreReason":"Few search."}]}` + "\n```"}
	got, used, err := writer.New(c, "claude-haiku-5-5", "Write them.").Write(t.Context(), gaps(), pages())
	if err != nil || len(got) != 1 || used.InputTokens != 900 || used.OutputTokens != 90 {
		t.Fatalf("Write = %+v, %+v, %v", got, used, err)
	}
	if w := got[0]; w.Key != "term:nl:spiraalband" || w.Title != "Use it" || w.Where != "https://www.tribelt.nl/spiraalbanden" || w.Benefit != 3 || w.ScoreReason != "Few search." || w.Steps[0] != "Add it." {
		t.Errorf("written = %+v", w)
	}
	p := c.prompt
	if p.Model != "claude-haiku-5-5" || p.System != "Write them." || p.SchemaName != "recommendations" || p.Schema == nil || p.MaxTokens == 0 {
		t.Errorf("prompt = %+v", p)
	}
	for _, want := range []string{
		`<gap key="term:nl:spiraalband">`, "Kind: term", "Language: nl", "Belt family: open_spiral", "Benefit 4, effort 2", "- 2 competitors use it.",
		`<page address="https://www.tribelt.nl/spiraalbanden">`, "Title: Spiraalbanden",
	} {
		if !strings.Contains(p.User, want) {
			t.Errorf("the prompt lacks %q:\n%s", want, p.User)
		}
	}
	if strings.Count(p.User, "Spiraal. ") > 300 {
		t.Error("a page's text was not cut short")
	}
}

func TestTheModelFailing(t *testing.T) {
	for _, c := range []*client{{err: errors.New("provider down")}, {text: "Sure! Here goes."}} {
		if _, used, err := writer.New(c, "m", "x").Write(t.Context(), gaps(), pages()); err == nil || used.InputTokens != 900 {
			t.Errorf("Write with %+v = %+v, %v", c, used, err)
		}
	}
}
