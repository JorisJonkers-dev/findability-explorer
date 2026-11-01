// Package matcher implements the analysis Matcher port with a language model: it reads a site's
// text and names the site's products that correspond to Tribelt's belt families, with the search
// terms the site uses for them in each language. It ports the find_keywords prompt of
// github.com/RNgo-Sch/digital-strategy-webscraper, with structured output instead of free text.
// Which provider and model answer is the language-model context's business (internal/llm).
package matcher

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/JorisJonkers-dev/findability-explorer/internal/analysis/domain"
	llm "github.com/JorisJonkers-dev/findability-explorer/internal/llm/domain"
	"github.com/JorisJonkers-dev/findability-explorer/internal/shared/lang"
)

const (
	maxTokens  = 8000
	schemaName = "tribelt_equivalents"
)

// Matcher asks one model through one provider's client, with the instructions of the Claude step
// matching products (internal/claudestep), which an administrator may change; the data it sends
// and the reply format it reads stay here.
type Matcher struct {
	client       llm.Client
	model        string
	instructions string
}

var _ domain.Matcher = (*Matcher)(nil)

// New returns a Matcher asking model through client, with instructions as the system message.
func New(client llm.Client, model, instructions string) *Matcher {
	return &Matcher{client: client, model: model, instructions: instructions}
}

type answer struct {
	Equivalents []struct {
		Product     string `json:"product"`
		Family      string `json:"family"`
		Reason      string `json:"reason"`
		SearchTerms []struct {
			Term     string `json:"term"`
			Language string `json:"language"`
		} `json:"searchTerms"`
	} `json:"equivalents"`
}

// Match sends one request and reads the structured answer.
func (m *Matcher) Match(ctx context.Context, siteText string, truncated bool, families []domain.TribeltFamily) ([]domain.Equivalent, domain.Usage, error) {
	resp, err := m.client.Complete(ctx, llm.Prompt{
		Model: m.model, System: m.instructions, User: prompt(siteText, truncated, families),
		SchemaName: schemaName, Schema: schema(families), MaxTokens: maxTokens,
	})
	usage := domain.Usage{InputTokens: resp.InputTokens, OutputTokens: resp.OutputTokens}
	if err != nil {
		return nil, usage, err
	}
	var a answer
	if err := json.Unmarshal([]byte(unfence(resp.Text)), &a); err != nil {
		return nil, usage, fmt.Errorf("read the model's answer: %w", err)
	}
	out := make([]domain.Equivalent, 0, len(a.Equivalents))
	for _, e := range a.Equivalents {
		terms := make([]domain.SearchTerm, 0, len(e.SearchTerms))
		for _, s := range e.SearchTerms {
			if lang.InFocus(s.Language) { // a model may still answer in another language
				terms = append(terms, domain.SearchTerm{Term: s.Term, Language: s.Language})
			}
		}
		out = append(out, domain.Equivalent{Product: e.Product, Family: e.Family, Reason: e.Reason, SearchTerms: terms})
	}
	return out, usage, nil
}

// unfence drops the Markdown code fence a model in JSON mode may wrap its answer in.
func unfence(text string) string {
	text = strings.TrimSpace(text)
	if !strings.HasPrefix(text, "```") {
		return text
	}
	text = strings.TrimPrefix(strings.TrimPrefix(text, "```json"), "```")
	return strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(text), "```"))
}

func prompt(siteText string, truncated bool, families []domain.TribeltFamily) string {
	var b strings.Builder
	b.WriteString("<tribelt_families>\n")
	for _, f := range families {
		fmt.Fprintf(&b, "- id %s: %s. %s\n", f.Family, f.Name, f.Description)
	}
	b.WriteString("</tribelt_families>\n\n<website_text>\n")
	b.WriteString(siteText)
	b.WriteString("\n</website_text>\n")
	if truncated {
		b.WriteString("\nThe website text was cut off at a length limit; judge only what is there.\n")
	}
	return b.String()
}

// schema is the JSON schema the answer must follow; family is limited to Tribelt's ids.
func schema(families []domain.TribeltFamily) map[string]any {
	ids := make([]any, 0, len(families))
	for _, f := range families {
		ids = append(ids, f.Family)
	}
	str := map[string]any{"type": "string"}
	return map[string]any{
		"type": "object", "additionalProperties": false, "required": []any{"equivalents"},
		"properties": map[string]any{
			"equivalents": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object", "additionalProperties": false,
					"required": []any{"product", "family", "reason", "searchTerms"},
					"properties": map[string]any{
						"product": str,
						"family":  map[string]any{"type": "string", "enum": ids},
						"reason":  str,
						"searchTerms": map[string]any{
							"type": "array",
							"items": map[string]any{
								"type": "object", "additionalProperties": false,
								"required":   []any{"term", "language"},
								"properties": map[string]any{"term": str, "language": str},
							},
						},
					},
				},
			},
		},
	}
}
