package domain

import (
	"cmp"
	"fmt"
	"regexp"
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

// TextModels are the models of OpenRouter's catalogue a Prompt run can ask: every text model
// (TextModels keeps those that read and write text), but no router such as openrouter/auto and no
// alias such as ~anthropic/claude-haiku-latest, which stand for another model.
func TextModels(catalogue []CatalogueModel) []CatalogueModel {
	return slices.DeleteFunc(slices.Clone(catalogue), func(m CatalogueModel) bool {
		return !m.answersInText() || m.Provider() == "openrouter" || strings.HasPrefix(m.ID, "~")
	})
}

// Recommended are the models a Model list starts from: the cheapest text model of each of the five
// providers whose assistants buyers use most (DefaultProviders), in that order.
func Recommended(catalogue []CatalogueModel) []string {
	var out []string
	for _, c := range ChooseModels(TextModels(catalogue), DefaultProviders()) {
		out = append(out, c.Model)
	}
	return out
}

// ChooseListed is what a run asks for a saved Model list: each listed model still on the catalogue,
// in the list's order, a free one with its provider's cheapest paid model as the fallback. Each is
// its own column. The column keeps the provider's id while the list has one model of that provider,
// as every run before the Model list did; two models of one provider are named by their model ids.
func ChooseListed(catalogue []CatalogueModel, list []string) []ModelChoice {
	text := TextModels(catalogue)
	byPrice := slices.Clone(text)
	slices.SortFunc(byPrice, func(a, b CatalogueModel) int {
		return cmp.Or(cmp.Compare(a.price(), b.price()), strings.Compare(a.ID, b.ID))
	})
	var found []CatalogueModel
	for _, id := range list {
		if i := slices.IndexFunc(text, func(m CatalogueModel) bool { return m.ID == id }); i >= 0 {
			found = append(found, text[i])
		}
	}
	perProvider := map[string]int{}
	for _, m := range found {
		perProvider[m.Provider()]++
	}
	out := make([]ModelChoice, 0, len(found))
	for _, m := range found {
		c := ModelChoice{Provider: m.Provider(), Model: m.ID, Free: m.Free()}
		if perProvider[m.Provider()] > 1 {
			c.Provider = m.ID
		}
		if c.Free {
			if j := slices.IndexFunc(byPrice, func(o CatalogueModel) bool { return o.Provider() == m.Provider() && !o.Free() }); j >= 0 {
				c.Fallback = byPrice[j].ID
			}
		}
		out = append(out, c)
	}
	return out
}

// Rules for the Model list.
const (
	MaxModels  = 20
	maxModelID = 200
)

var modelID = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*/[a-z0-9][a-z0-9._:-]*$`)

// CheckModels trims and de-duplicates a Model list: 1 to MaxModels OpenRouter model ids, such as
// "openai/gpt-oss-20b" or "google/gemma-3-27b-it:free".
func CheckModels(raw []string) ([]string, error) {
	var out []string
	for _, id := range raw {
		id = strings.TrimSpace(id)
		if len(id) > maxModelID || !modelID.MatchString(id) {
			return nil, fmt.Errorf("%w: %q is no OpenRouter model id, such as openai/gpt-oss-20b", ErrInvalid, id)
		}
		if !slices.Contains(out, id) {
			out = append(out, id)
		}
	}
	if len(out) == 0 || len(out) > MaxModels {
		return nil, fmt.Errorf("%w: list 1 to %d models", ErrInvalid, MaxModels)
	}
	return out, nil
}

// CatalogueEntry is one model of the catalogue as Add a model lists it: whether it is recommended,
// and what one response costs from its own knowledge and with web search, by the latest run's own
// responses when that run asked it (BasisRun), else by its list price (BasisCatalogue).
type CatalogueEntry struct {
	Model       CatalogueModel
	Recommended bool
	Basis       PriceBasis
	Knowledge   Micros
	Web         Micros
}

// Catalogue is every text model with what a response costs: the recommended ones first, in the
// order of DefaultProviders, then the rest cheapest first, ties by id. answers are the latest
// run's; a model they answered is priced by them, mode by mode.
func Catalogue(catalogue []CatalogueModel, answers []Answer) []CatalogueEntry {
	text := TextModels(catalogue)
	recommended := Recommended(text)
	out := make([]CatalogueEntry, 0, len(text))
	for _, m := range text {
		e := CatalogueEntry{Model: m, Recommended: slices.Contains(recommended, m.ID), Basis: BasisCatalogue}
		e.Knowledge = TokenPrice{Input: m.PromptPrice, Output: m.CompletionPrice}.Of(typicalAnswerIn, typicalAnswerOut)
		e.Web = e.Knowledge + WebSearchFee(m)
		if k, ok := modelAnswer(answers, m.ID, ModeKnowledge); ok {
			e.Basis, e.Knowledge = BasisRun, k
		}
		if w, ok := modelAnswer(answers, m.ID, ModeWeb); ok {
			e.Basis, e.Web = BasisRun, w
		}
		out = append(out, e)
	}
	rank := func(e CatalogueEntry) int {
		if i := slices.Index(recommended, e.Model.ID); i >= 0 {
			return i
		}
		return len(recommended)
	}
	slices.SortStableFunc(out, func(a, b CatalogueEntry) int {
		return cmp.Or(cmp.Compare(rank(a), rank(b)), cmp.Compare(a.Knowledge+a.Web, b.Knowledge+b.Web), strings.Compare(a.Model.ID, b.Model.ID))
	})
	return out
}

// modelAnswer is what one of model's responses in mode cost on average; false without one.
func modelAnswer(answers []Answer, model string, mode Mode) (Micros, bool) {
	var sum, n Micros
	for _, a := range answers {
		if a.Model == model && a.Mode == mode && a.Failure == "" {
			sum += a.Cost
			n++
		}
	}
	if n == 0 {
		return 0, false
	}
	return sum / n, true
}
