package judge_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	llm "github.com/JorisJonkers-dev/findability-explorer/internal/llm/domain"
	"github.com/JorisJonkers-dev/findability-explorer/internal/terminology/adapters/judge"
	"github.com/JorisJonkers-dev/findability-explorer/internal/terminology/domain"
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
	return llm.Completion{Text: c.text, InputTokens: 1200, OutputTokens: 300}, c.err
}

func candidates() []domain.Candidate {
	return []domain.Candidate{
		{
			ID: "00000000000000aa", Phrase: "spiral freezer belts", Languages: []string{"en"}, Closest: "spiral belt",
			Companies: []domain.CandidateCompany{{ID: "a", Name: "Ashworth"}, {ID: "t", Name: "Twentebelt"}},
		},
		{ID: "00000000000000bb", Phrase: "contact", Languages: []string{"en", "nl"}, Companies: []domain.CandidateCompany{{ID: "t", Name: "Twentebelt"}}},
	}
}

func TestTheModelGivesAVerdictPerPhrase(t *testing.T) {
	c := &client{text: "```json\n" + `{"verdicts":[
		{"id":"00000000000000aa","accept":true,"sure":true,"reason":"  A belt buyers search for. ","term":"spiral freezer belt","language":"en","family":"tri_flex_turn_curve","tier":"application"},
		{"id":"00000000000000bb","accept":false,"sure":true,"reason":"` + strings.Repeat("x", 400) + `","term":"","language":"","family":"","tier":""}
	]}` + "\n```"}
	examples := []domain.Term{{Text: "spiral belt", Language: "en", Family: "tri_flex_turn_curve", Tier: "product"}}
	got, usage, err := judge.New(c, "claude-haiku-5-5").Judge(t.Context(), candidates(), examples)
	if err != nil || len(got) != 2 {
		t.Fatalf("Judge = %+v, %v", got, err)
	}
	if usage.InputTokens != 1200 || usage.OutputTokens != 300 {
		t.Errorf("usage = %+v", usage)
	}
	a := got[0]
	if a.CandidateID != "00000000000000aa" || !a.Accept || !a.Sure || a.Reason != "A belt buyers search for." ||
		a.Term.Text != "spiral freezer belt" || a.Term.Family != "tri_flex_turn_curve" || a.Term.Tier != "application" || a.Term.Language != "en" {
		t.Errorf("the accepted one = %+v", a)
	}
	if b := got[1]; b.Accept || len([]rune(b.Reason)) != 300 {
		t.Errorf("the rejected one = %+v", b)
	}
	p := c.prompt
	if p.Model != "claude-haiku-5-5" || p.SchemaName == "" || p.Schema == nil || !strings.Contains(p.System, "metal conveyor belts") {
		t.Errorf("prompt = %+v", p)
	}
	for _, want := range []string{
		`- id 00000000000000aa: "spiral freezer belts", found at 2 companies in en; closest map term "spiral belt"`,
		`- id 00000000000000bb: "contact", found at 1 company in en, nl` + "\n",
		`- "spiral belt" (en, tri_flex_turn_curve, product)`,
		"- spec_all: ",
	} {
		if !strings.Contains(p.User, want) {
			t.Errorf("the prompt lacks %q:\n%s", want, p.User)
		}
	}
}

func TestTheModelFailing(t *testing.T) {
	down := &client{err: errors.New("provider down")}
	if _, usage, err := judge.New(down, "m").Judge(t.Context(), candidates(), nil); err == nil || usage.InputTokens != 1200 {
		t.Errorf("a failing provider = %+v, %v", usage, err)
	}
	garbled := &client{text: "Sure! Here are my thoughts."}
	if _, usage, err := judge.New(garbled, "m").Judge(t.Context(), candidates(), nil); err == nil || usage.OutputTokens != 300 {
		t.Errorf("an answer that is no JSON = %+v, %v", usage, err)
	}
}
