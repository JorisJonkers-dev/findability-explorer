// Package writer implements the insight Writer port with a Claude model: the Claude step writing
// recommendations, which places and words the gaps a Recommendation run's rules found, from
// tribelt.nl's pages. The instructions and the model are the step's; the data it gets and the
// reply it gives are set here.
package writer

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/JorisJonkers-dev/findability-explorer/internal/insight/app"
	"github.com/JorisJonkers-dev/findability-explorer/internal/insight/domain"
	llm "github.com/JorisJonkers-dev/findability-explorer/internal/llm/domain"
)

const (
	maxTokens = 12000
	// maxPageText is the most of a page's text the model reads.
	maxPageText = 2500
)

// Writer asks one model through one provider's client, with the step's instructions.
type Writer struct {
	client       llm.Client
	model        string
	instructions string
}

var _ app.Writer = (*Writer)(nil)

// New returns a Writer asking model through client with instructions.
func New(client llm.Client, model, instructions string) *Writer {
	return &Writer{client: client, model: model, instructions: instructions}
}

// Write has the model write a recommendation for each gap.
func (w *Writer) Write(ctx context.Context, gaps []domain.Gap, pages []domain.Page) ([]domain.Written, app.Usage, error) {
	var b strings.Builder
	b.WriteString("<gaps>\n")
	for _, g := range gaps {
		fmt.Fprintf(&b, "<gap key=%q>\nKind: %s\nDraft: %s\n", g.Key, g.Kind, g.Title)
		if g.Language != "" {
			fmt.Fprintf(&b, "Language: %s\n", g.Language)
		}
		if g.Family != "" {
			fmt.Fprintf(&b, "Belt family: %s\n", g.Family)
		}
		fmt.Fprintf(&b, "Benefit %d, effort %d\nFacts:\n", g.Benefit, g.Effort)
		for _, s := range g.Specifics {
			fmt.Fprintf(&b, "- %s\n", s)
		}
		b.WriteString("</gap>\n")
	}
	b.WriteString("</gaps>\n\n<pages>\n")
	for _, p := range pages {
		fmt.Fprintf(&b, "<page address=%q>\nTitle: %s\nLanguage: %s\nText: %s\n</page>\n", p.URL, p.Title, p.Language, clip(p.Text, maxPageText))
	}
	b.WriteString("</pages>\n\nWrite one recommendation for every gap, with its key.\n")
	resp, err := w.client.Complete(ctx, llm.Prompt{
		Model: w.model, System: w.instructions, User: b.String(), SchemaName: "recommendations", Schema: schema(), MaxTokens: maxTokens,
	})
	used := app.Usage{InputTokens: resp.InputTokens, OutputTokens: resp.OutputTokens}
	if err != nil {
		return nil, used, err
	}
	var a struct {
		Recommendations []struct {
			Key         string   `json:"key"`
			Title       string   `json:"title"`
			Where       string   `json:"where"`
			Why         string   `json:"why"`
			Steps       []string `json:"steps"`
			DoneWhen    string   `json:"doneWhen"`
			Benefit     int      `json:"benefit"`
			Effort      int      `json:"effort"`
			ScoreReason string   `json:"scoreReason"`
		} `json:"recommendations"`
	}
	if err := json.Unmarshal([]byte(unfence(resp.Text)), &a); err != nil {
		return nil, used, fmt.Errorf("read the model's answer: %w", err)
	}
	out := make([]domain.Written, 0, len(a.Recommendations))
	for _, r := range a.Recommendations {
		out = append(out, domain.Written{
			Key: r.Key, Title: r.Title, Where: r.Where, Why: r.Why, Steps: r.Steps, DoneWhen: r.DoneWhen, Benefit: r.Benefit, Effort: r.Effort, ScoreReason: r.ScoreReason,
		})
	}
	return out, used, nil
}

// schema is the reply the model must give: a recommendation per gap.
func schema() map[string]any {
	str, num := map[string]any{"type": "string"}, map[string]any{"type": "integer"}
	return map[string]any{
		"type": "object", "additionalProperties": false, "required": []any{"recommendations"},
		"properties": map[string]any{
			"recommendations": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object", "additionalProperties": false,
					"required":   []any{"key", "title", "where", "why", "steps", "doneWhen", "benefit", "effort", "scoreReason"},
					"properties": map[string]any{"key": str, "title": str, "where": str, "why": str, "steps": map[string]any{"type": "array", "items": str}, "doneWhen": str, "benefit": num, "effort": num, "scoreReason": str},
				},
			},
		},
	}
}

func clip(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n]) + "…"
	}
	return s
}

func unfence(text string) string {
	text = strings.TrimSpace(text)
	if !strings.HasPrefix(text, "```") {
		return text
	}
	text = strings.TrimPrefix(strings.TrimPrefix(text, "```json"), "```")
	return strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(text), "```"))
}
