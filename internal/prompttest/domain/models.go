package domain

import (
	"cmp"
	"slices"
	"strings"
)

// DefaultProviders are the large providers whose cheapest model a Prompt run asks, by their
// OpenRouter id prefix: OpenAI, Google, Anthropic, Mistral and DeepSeek. Meta and xAI were
// dropped: xAI's web search cost the most, and neither is an assistant these buyers use.
func DefaultProviders() []string {
	return []string{"openai", "google", "anthropic", "mistralai", "deepseek"}
}

// ExtractorModel is the one model that reads every answer, fixed in code and changed only by a
// release, so every answer of every run is read the same way (docs/adr/0004).
const ExtractorModel = "google/gemini-2.5-flash-lite"

// CatalogueModel is a model on OpenRouter's price list. Prices are dollars per token; a price
// below zero means it varies (a router, not a model).
type CatalogueModel struct {
	ID              string
	Name            string
	PromptPrice     float64
	CompletionPrice float64
	// WebSearchPrice is what one search of the model's own costs, in dollars, for a model that
	// searches the web itself; 0 when it does not, and OpenRouter searches for it.
	WebSearchPrice float64
	// InputModalities and OutputModalities are what the model reads and writes: "text", "image".
	InputModalities  []string
	OutputModalities []string
}

// Provider is the provider part of a model id: "openai" of "openai/gpt-oss-20b".
func (m CatalogueModel) Provider() string {
	p, _, _ := strings.Cut(m.ID, "/")
	return p
}

func (m CatalogueModel) price() float64 { return m.PromptPrice + m.CompletionPrice }

// Free reports whether the model costs nothing.
func (m CatalogueModel) Free() bool { return m.price() == 0 }

// answersInText reports whether the model reads text and writes text only: not an image,
// audio or embedding model, a batch-only variant, a variable-priced router or a moderation model.
func (m CatalogueModel) answersInText() bool {
	return slices.Contains(m.InputModalities, "text") && slices.Equal(m.OutputModalities, []string{"text"}) &&
		m.PromptPrice >= 0 && m.CompletionPrice >= 0 &&
		!strings.HasSuffix(m.ID, ":batch") && !strings.Contains(m.ID, "-guard")
}

// ModelChoice is the model a Prompt run asks for one provider: its cheapest text model, free
// ones included, and the cheapest paid one to answer what a free model cannot finish. Fallback
// is empty when Model is paid already.
type ModelChoice struct {
	Provider string
	Model    string
	Free     bool
	Fallback string
}

// ChooseModels picks, for each provider in providers, its cheapest text model by prompt plus
// completion price (ties by id), free models included, and its cheapest paid one as the fallback
// of a free choice. A provider with no text model on the list is left out.
func ChooseModels(catalogue []CatalogueModel, providers []string) []ModelChoice {
	text := slices.DeleteFunc(slices.Clone(catalogue), func(m CatalogueModel) bool { return !m.answersInText() })
	slices.SortFunc(text, func(a, b CatalogueModel) int {
		return cmp.Or(cmp.Compare(a.price(), b.price()), strings.Compare(a.ID, b.ID))
	})
	var out []ModelChoice
	for _, p := range providers {
		i := slices.IndexFunc(text, func(m CatalogueModel) bool { return m.Provider() == p })
		if i < 0 {
			continue
		}
		c := ModelChoice{Provider: p, Model: text[i].ID, Free: text[i].Free()}
		if c.Free {
			if j := slices.IndexFunc(text, func(m CatalogueModel) bool { return m.Provider() == p && !m.Free() }); j >= 0 {
				c.Fallback = text[j].ID
			}
		}
		out = append(out, c)
	}
	return out
}
