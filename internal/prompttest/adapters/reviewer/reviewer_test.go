package reviewer_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	llm "github.com/JorisJonkers-dev/findability-explorer/internal/llm/domain"
	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/adapters/reviewer"
	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain"
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
	return llm.Completion{Text: c.text}, c.err
}

func candidates() []domain.Candidate {
	return []domain.Candidate{
		{
			ID: "00000000000000aa", Name: "Taiwan Metiz Alliance", Answers: 8, Sites: []string{"metiz.tw"},
			Members: []domain.Member{{ID: "00000000000000aa", Name: "Taiwan Metiz Alliance"}, {ID: "00000000000000a2", Name: "Metiz"}},
		},
		{ID: "00000000000000bb", Name: "Metiz Alliance Co", Answers: 1, Members: []domain.Member{{ID: "00000000000000bb", Name: "Metiz Alliance Co"}}},
	}
}

func TestTheModelMergesNamesOfOneCompany(t *testing.T) {
	c := &client{text: "```json\n" + `{"clusters":[{"ids":["00000000000000aa","00000000000000bb"],"competitorId":""},{"ids":["00000000000000cc"],"competitorId":"ashworth"}]}` + "\n```"}
	dir := []domain.DirectoryCompany{{ID: "ashworth", Name: "Ashworth Bros", Host: "ashworth.com"}, {ID: "esfo", Name: "ESFO"}}
	got, err := reviewer.New(c, "claude-haiku-5-5").Merge(t.Context(), candidates(), dir)
	if err != nil || len(got) != 2 || len(got[0].IDs) != 2 || got[1].CompetitorID != "ashworth" {
		t.Fatalf("Merge = %+v, %v", got, err)
	}
	for _, want := range []string{
		`- id ashworth: "Ashworth Bros", ashworth.com`,
		`- id esfo: "ESFO"` + "\n",
		`- id 00000000000000aa: "Taiwan Metiz Alliance", also "Metiz"; websites metiz.tw; named in 8 answers`,
		`- id 00000000000000bb: "Metiz Alliance Co"; named in 1 answers`,
	} {
		if !strings.Contains(c.prompt.User, want) {
			t.Errorf("the prompt lacks %q:\n%s", want, c.prompt.User)
		}
	}
	if c.prompt.Model != "claude-haiku-5-5" || c.prompt.SchemaName != "candidate_clusters" || !strings.Contains(c.prompt.System, "metal conveyor belts") {
		t.Errorf("prompt = %+v", c.prompt)
	}
}

func TestTheModelChecksEachCompany(t *testing.T) {
	c := &client{text: `{"verdicts":[{"id":"00000000000000aa","competitor":true,"sure":true,"reason":"  Makes wire mesh belts. "},{"id":"00000000000000bb","competitor":false,"sure":false,"reason":"` + strings.Repeat("x", 400) + `"}]}`}
	got, err := reviewer.New(c, "m").Check(t.Context(), candidates())
	if err != nil || len(got) != 2 || !got[0].Competitor || !got[0].Sure || got[0].Reason != "Makes wire mesh belts." || len([]rune(got[1].Reason)) != 300 {
		t.Fatalf("Check = %+v, %v", got, err)
	}
	if c.prompt.SchemaName != "candidate_checks" || !strings.Contains(c.prompt.User, "Give one verdict for every company") {
		t.Errorf("prompt = %+v", c.prompt)
	}
}

func TestTheModelFailing(t *testing.T) {
	for _, c := range []*client{{err: errors.New("provider down")}, {text: "Sure! Here goes."}} {
		if _, err := reviewer.New(c, "m").Merge(t.Context(), candidates(), nil); err == nil {
			t.Errorf("Merge with %+v succeeded", c)
		}
		if _, err := reviewer.New(c, "m").Check(t.Context(), candidates()); err == nil {
			t.Errorf("Check with %+v succeeded", c)
		}
	}
}
