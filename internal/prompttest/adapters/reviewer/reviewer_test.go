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
	return llm.Completion{Text: c.text, InputTokens: 120, OutputTokens: 30}, c.err
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
	got, used, err := reviewer.New(c, "claude-haiku-5-5").Merge(t.Context(), candidates(), dir)
	if err != nil || len(got) != 2 || len(got[0].IDs) != 2 || got[1].CompetitorID != "ashworth" || used.InputTokens != 120 || used.OutputTokens != 30 {
		t.Fatalf("Merge = %+v, %+v, %v", got, used, err)
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

func dossiers() []domain.Dossier {
	cs := candidates()
	cs[0].Website = "https://metiz.tw/"
	return []domain.Dossier{
		{
			Candidate: cs[0], Snippets: []string{"…Taiwan Metiz Alliance makes wire mesh belts for ovens…"},
			Home: &domain.HomePage{URL: "https://metiz.tw/", Title: "Metiz", Description: "Wire mesh conveyor belts", Headings: []string{"Belts", "Ovens"}, Text: "Since 1983."},
		},
		{Candidate: cs[1]},
		{Candidate: domain.Candidate{ID: "00000000000000cc", Name: "Gone Co", Website: "https://gone.example/"}, Home: &domain.HomePage{Problem: "the website did not answer"}},
	}
}

func TestTheModelChecksEachCompany(t *testing.T) {
	c := &client{text: `{"verdicts":[{"id":"00000000000000aa","competitor":true,"sure":true,"reason":"  Makes wire mesh belts. "},{"id":"00000000000000bb","competitor":false,"sure":false,"reason":"` + strings.Repeat("x", 400) + `"}]}`}
	got, used, err := reviewer.New(c, "m").Check(t.Context(), dossiers())
	if err != nil || len(got) != 2 || !got[0].Competitor || !got[0].Sure || got[0].Reason != "Makes wire mesh belts." || len([]rune(got[1].Reason)) != 300 || used.InputTokens != 120 {
		t.Fatalf("Check = %+v, %+v, %v", got, used, err)
	}
	if c.prompt.SchemaName != "candidate_checks" || !strings.Contains(c.prompt.User, "Give one verdict for every company") ||
		!strings.Contains(c.prompt.System, domain.TargetMarket) {
		t.Errorf("prompt = %+v", c.prompt)
	}
	for _, want := range []string{
		`<company id="00000000000000aa">` + "\nNames: \"Taiwan Metiz Alliance\", \"Metiz\"\nWebsite: https://metiz.tw/\nNamed in 8 answers.\n",
		"What the answers say:\n- …Taiwan Metiz Alliance makes wire mesh belts for ovens…\n",
		"Home page:\n  Title: Metiz\n  Description: Wire mesh conveyor belts\n  Headings: Belts | Ovens\n  Text: Since 1983.\n</company>",
		`<company id="00000000000000bb">` + "\nNames: \"Metiz Alliance Co\"\nNo website.\nNamed in 1 answers.\n</company>",
		"Names: \"Gone Co\"\nWebsite: https://gone.example/\nNot named in the answers.\nHome page: could not be read (the website did not answer).\n",
	} {
		if !strings.Contains(c.prompt.User, want) {
			t.Errorf("the prompt lacks %q:\n%s", want, c.prompt.User)
		}
	}
}

func TestTheModelFailing(t *testing.T) {
	for _, c := range []*client{{err: errors.New("provider down")}, {text: "Sure! Here goes."}} {
		if _, used, err := reviewer.New(c, "m").Merge(t.Context(), candidates(), nil); err == nil || used.InputTokens != 120 {
			t.Errorf("Merge with %+v = %+v, %v", c, used, err)
		}
		if _, used, err := reviewer.New(c, "m").Check(t.Context(), dossiers()); err == nil || used.InputTokens != 120 {
			t.Errorf("Check with %+v = %+v, %v", c, used, err)
		}
	}
}
