// Package extractor is how a Prompt run's answers are read: the instructions, the strict JSON
// format and the reading of the model's answer, shared by every model that reads them. Its
// Extractor reads them with Claude Haiku on an Anthropic key (ADR-0007); the openrouter package
// reads them on OpenRouter when no Anthropic key works.
package extractor

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	llm "github.com/JorisJonkers-dev/findability-explorer/internal/llm/domain"
	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/app"
	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain"
)

// System tells the extractor how to read an answer.
const System = `You read an AI assistant's answer to a buyer's question about metal conveyor belts and report, in the JSON schema given, exactly what it says:
- companies: every company the answer names as a maker or supplier, in the order it first names them, with the web address it gives for that company ("" when none).
- addresses: every web address the answer cites or links to.
- namesTribelt: whether it names Tribelt (Tribelt B.V.).
- citesTribelt: whether it cites or links a Tribelt web address.
- correctness: when it names Tribelt, whether what it says about Tribelt agrees with the fact sheet: "correct", "partly" (some of it wrong) or "wrong"; "none" when it does not name Tribelt.
- wrongClaims: each claim about Tribelt the fact sheet contradicts, quoted from the answer; empty when there is none.
Report only what the answer says. Do not add companies it does not name.`

// SchemaName names the strict format.
const SchemaName = "answer_reading"

// Schema is the strict format the extractor answers in.
func Schema() map[string]any {
	str := map[string]any{"type": "string"}
	return map[string]any{
		"type": "object", "additionalProperties": false,
		"required": []string{"companies", "addresses", "namesTribelt", "citesTribelt", "correctness", "wrongClaims"},
		"properties": map[string]any{
			"companies": map[string]any{"type": "array", "items": map[string]any{
				"type": "object", "additionalProperties": false, "required": []string{"name", "url"},
				"properties": map[string]any{"name": str, "url": str},
			}},
			"addresses":    map[string]any{"type": "array", "items": str},
			"namesTribelt": map[string]any{"type": "boolean"},
			"citesTribelt": map[string]any{"type": "boolean"},
			"correctness":  map[string]any{"type": "string", "enum": []string{"none", "correct", "partly", "wrong"}},
			"wrongClaims":  map[string]any{"type": "array", "items": str},
		},
	}
}

// User is what the extractor is given to read: the fact sheet, the question, the answer and its
// cited sources.
func User(in app.ExtractInput) string {
	user := "Fact sheet about Tribelt:\n" + in.FactSheet + "\n\nThe buyer's question:\n" + in.Prompt + "\n\nThe answer:\n" + in.Answer
	if len(in.Citations) > 0 {
		user += "\n\nThe answer's cited sources:\n" + strings.Join(in.Citations, "\n")
	}
	return user
}

// Limits on what one extraction keeps.
const (
	maxCompanies = 30
	maxAddresses = 50
	maxClaims    = 10
	maxName      = 200
	maxText      = 500
)

// Parse reads the extractor's answer in the strict format, keeping at most the limits above.
func Parse(text string) (domain.Extracted, error) {
	var raw struct {
		Companies []struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		} `json:"companies"`
		Addresses    []string `json:"addresses"`
		NamesTribelt bool     `json:"namesTribelt"`
		CitesTribelt bool     `json:"citesTribelt"`
		Correctness  string   `json:"correctness"`
		WrongClaims  []string `json:"wrongClaims"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(text)), &raw); err != nil {
		return domain.Extracted{}, fmt.Errorf("the extractor's answer is not the format asked: %w", err)
	}
	x := domain.Extracted{NamesTribelt: raw.NamesTribelt, CitesTribelt: raw.CitesTribelt}
	for _, co := range raw.Companies[:min(len(raw.Companies), maxCompanies)] {
		if name := strings.TrimSpace(co.Name); name != "" {
			x.Companies = append(x.Companies, domain.Named{Name: clip(name, maxName), URL: clip(strings.TrimSpace(co.URL), maxText)})
		}
	}
	for _, a := range raw.Addresses[:min(len(raw.Addresses), maxAddresses)] {
		x.Addresses = append(x.Addresses, clip(a, maxText))
	}
	switch c := domain.Correctness(raw.Correctness); c {
	case domain.CorrectnessCorrect, domain.CorrectnessPartly, domain.CorrectnessWrong:
		x.Correctness = c
	case domain.CorrectnessNone:
	}
	for _, claim := range raw.WrongClaims[:min(len(raw.WrongClaims), maxClaims)] {
		x.WrongClaims = append(x.WrongClaims, clip(claim, maxText))
	}
	return x, nil
}

// maxTokens leaves Haiku room to think before it answers.
const maxTokens = 8000

// Extractor reads answers with Claude Haiku through an Anthropic client.
type Extractor struct {
	client llm.Client
	model  string
}

var _ app.Extractor = (*Extractor)(nil)

// New returns an Extractor asking model, Claude Haiku, through client.
func New(client llm.Client, model string) *Extractor {
	return &Extractor{client: client, model: model}
}

// Extract has Haiku read an answer in the strict format. What it used is priced at Haiku's list
// price (domain.HaikuPrice), also when the reading failed; an answer to read is far shorter than
// the 100,000 tokens that price holds for.
func (e *Extractor) Extract(ctx context.Context, in app.ExtractInput) (domain.Extracted, domain.Reply, error) {
	resp, err := e.client.Complete(ctx, llm.Prompt{
		Model: e.model, System: System, User: User(in), SchemaName: SchemaName, Schema: Schema(), MaxTokens: maxTokens,
	})
	usage := domain.Reply{
		InputTokens: resp.InputTokens, OutputTokens: resp.OutputTokens,
		Cost: domain.HaikuPrice().Of(resp.InputTokens, resp.OutputTokens),
	}
	if err != nil {
		return domain.Extracted{}, usage, err
	}
	x, err := Parse(resp.Text)
	return x, usage, err
}

// clip cuts s to at most n characters.
func clip(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n])
	}
	return s
}
