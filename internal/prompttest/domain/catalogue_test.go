package domain_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain"
)

func modelIDs(models []domain.CatalogueModel) []string {
	out := make([]string, 0, len(models))
	for _, m := range models {
		out = append(out, m.ID)
	}
	return out
}

func TestTextModelsKeepsEveryTextModelAndNoRouterOrAlias(t *testing.T) {
	catalogue := []domain.CatalogueModel{
		model("openai/gpt-oss-20b", 0.00000003, 0.00000014),
		model("openai/gpt-oss-20b:batch", 0.00000001, 0.00000001),
		model("openai/gpt-image", 0, 0, "image"),
		model("google/gemma:free", 0, 0),
		model("google/lyria", 0, 0, "audio", "text"),
		model("meta-llama/llama-guard-4", 0, 0),
		model("openrouter/auto", -1, -1),
		model("openrouter/some-model", 0.000001, 0.000001),
		model("~anthropic/claude-haiku-latest", 0.000001, 0.000005),
		model("anthropic/claude-haiku", 0.000001, 0.000005),
	}
	got := modelIDs(domain.TextModels(catalogue))
	want := []string{"openai/gpt-oss-20b", "google/gemma:free", "anthropic/claude-haiku"}
	if !slices.Equal(got, want) {
		t.Errorf("text models = %v, want %v", got, want)
	}
}

func TestRecommendedAreTheCheapestOfTheFiveLargeProviders(t *testing.T) {
	catalogue := []domain.CatalogueModel{
		model("openai/gpt-oss-20b", 0.00000003, 0.00000014),
		model("openai/gpt-5", 0.000001, 0.00001),
		model("google/gemma:free", 0, 0),
		model("google/gemini-flash-lite", 0.0000001, 0.0000004),
		model("anthropic/claude-haiku", 0.000001, 0.000005),
		model("mistralai/small", 0.0000001, 0.0000003),
		model("deepseek/v3", 0.0000002, 0.0000008),
		model("meta-llama/llama-3.1-8b", 0.00000001, 0.00000001),
		model("~google/gemini-latest", 0, 0),
	}
	got := domain.Recommended(catalogue)
	want := []string{"openai/gpt-oss-20b", "google/gemma:free", "anthropic/claude-haiku", "mistralai/small", "deepseek/v3"}
	if !slices.Equal(got, want) {
		t.Errorf("recommended = %v, want %v", got, want)
	}
}

func TestChooseListedAsksEachListedModelAsItsOwnColumn(t *testing.T) {
	catalogue := []domain.CatalogueModel{
		model("openai/gpt-oss-20b", 0.00000003, 0.00000014),
		model("openai/gpt-5-mini", 0.0000003, 0.000002),
		model("google/gemma:free", 0, 0),
		model("google/gemini-flash-lite", 0.0000001, 0.0000004),
		model("anthropic/claude-haiku", 0.000001, 0.000005),
		model("anthropic/claude-image", 0, 0, "image"),
	}
	got := domain.ChooseListed(catalogue, []string{"google/gemma:free", "openai/gpt-5-mini", "gone/model", "openai/gpt-oss-20b", "anthropic/claude-haiku", "anthropic/claude-image"})
	want := []domain.ModelChoice{
		// One model of a provider keeps the provider as its column, as runs before the list had it.
		{Provider: "google", Model: "google/gemma:free", Free: true, Fallback: "google/gemini-flash-lite"},
		// Two of one provider are a column each, named by the model.
		{Provider: "openai/gpt-5-mini", Model: "openai/gpt-5-mini"},
		{Provider: "openai/gpt-oss-20b", Model: "openai/gpt-oss-20b"},
		{Provider: "anthropic", Model: "anthropic/claude-haiku"},
	}
	if !slices.Equal(got, want) {
		t.Errorf("choices = %+v\nwant %+v", got, want)
	}
	if len(domain.ChooseListed(catalogue, nil)) != 0 {
		t.Error("an empty list asks nobody")
	}
}

func TestCheckModelsTrimsAndDeduplicates(t *testing.T) {
	got, err := domain.CheckModels([]string{" openai/gpt-oss-20b ", "google/gemma:free", "openai/gpt-oss-20b", "z-ai/glm-4.5-air:free"})
	if err != nil || !slices.Equal(got, []string{"openai/gpt-oss-20b", "google/gemma:free", "z-ai/glm-4.5-air:free"}) {
		t.Fatalf("models = %v, %v", got, err)
	}
	for _, bad := range [][]string{nil, {}, {"openai"}, {"Open AI/x"}, {"/x"}, {strings.Repeat("a", 190) + "/" + strings.Repeat("b", 20)}} {
		if _, err := domain.CheckModels(bad); !errors.Is(err, domain.ErrInvalid) {
			t.Errorf("%v: err = %v", bad, err)
		}
	}
	many := make([]string, domain.MaxModels+1)
	for i := range many {
		many[i] = "p/m" + strings.Repeat("x", i)
	}
	if _, err := domain.CheckModels(many); !errors.Is(err, domain.ErrInvalid) {
		t.Errorf("too many: err = %v", err)
	}
}

func TestCatalogueListsTheRecommendedFirstThenTheCheapest(t *testing.T) {
	haiku := model("anthropic/claude-haiku", 0.000001, 0.000005)
	haiku.WebSearchPrice = 0.01
	catalogue := []domain.CatalogueModel{
		model("meta-llama/llama-3.1-8b", 0.00000002, 0.00000005),
		model("openai/gpt-5", 0.000001, 0.00001),
		model("openai/gpt-oss-20b", 0.00000003, 0.00000014),
		haiku,
		model("qwen/qwen3:free", 0, 0),
		model("openai/gpt-image", 0, 0, "image"),
	}
	// The run asked gpt-oss-20b already: two answers in each mode.
	answers := []domain.Answer{
		{Model: "openai/gpt-oss-20b", Mode: domain.ModeKnowledge, Reply: domain.Reply{Cost: 100}},
		{Model: "openai/gpt-oss-20b", Mode: domain.ModeKnowledge, Reply: domain.Reply{Cost: 300}},
		{Model: "openai/gpt-oss-20b", Mode: domain.ModeWeb, Reply: domain.Reply{Cost: 20_500}},
		{Model: "openai/gpt-oss-20b", Mode: domain.ModeWeb, Failure: "no answer", Reply: domain.Reply{Cost: 1}},
	}
	got := domain.Catalogue(catalogue, answers)
	order := make([]string, 0, len(got))
	for _, e := range got {
		order = append(order, e.Model.ID)
	}
	want := []string{"openai/gpt-oss-20b", "anthropic/claude-haiku", "qwen/qwen3:free", "meta-llama/llama-3.1-8b", "openai/gpt-5"}
	if !slices.Equal(order, want) {
		t.Fatalf("order = %v, want %v", order, want)
	}
	asked, claude, free := got[0], got[1], got[2]
	if !asked.Recommended || asked.Basis != domain.BasisRun || asked.Knowledge != 200 || asked.Web != 20_500 {
		t.Errorf("asked = %+v", asked)
	}
	// 300 tokens in, 900 out, and the model's own search at a cent.
	if !claude.Recommended || claude.Basis != domain.BasisCatalogue || claude.Knowledge != 4_800 || claude.Web != 14_800 {
		t.Errorf("claude = %+v", claude)
	}
	// A free model pays OpenRouter's search: five results at $4 per 1,000.
	if free.Recommended || free.Knowledge != 0 || free.Web != 20_000 {
		t.Errorf("free = %+v", free)
	}
}

func TestAskedModelsSplitsTheListByTheRun(t *testing.T) {
	run := []domain.ModelChoice{{Provider: "openai", Model: "openai/a"}, {Provider: "google", Model: "google/b"}}
	kept, missing := domain.AskedModels(run, []string{"google/b", "mistralai/c"})
	if !slices.Equal(kept, run[1:]) || !slices.Equal(missing, []string{"mistralai/c"}) {
		t.Errorf("kept %v, missing %v", kept, missing)
	}
	if got := domain.OfModels(run, []string{"openai/a"}); !slices.Equal(got, run[:1]) {
		t.Errorf("of models = %v", got)
	}
}
