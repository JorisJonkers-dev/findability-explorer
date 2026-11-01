package domain_test

import (
	"testing"

	"github.com/JorisJonkers-dev/findability-explorer/internal/llm/domain"
)

func TestTheClaudeModelsAreListedCheapestFirstWithHaikuTheDefault(t *testing.T) {
	models := domain.ClaudeModels()
	if len(models) < 3 {
		t.Fatalf("models = %v", models)
	}
	if models[0].ID != domain.DefaultModel || models[0].Name != "Claude Haiku 5.5" {
		t.Errorf("first = %+v, want Claude Haiku 5.5", models[0])
	}
	for i := 1; i < len(models); i++ {
		a, b := models[i-1].Price, models[i].Price
		if a.Of(1_000_000, 1_000_000) > b.Of(1_000_000, 1_000_000) {
			t.Errorf("%s is listed before the cheaper %s", models[i-1].ID, models[i].ID)
		}
	}
}

func TestAClaudeModelIsFoundByItsID(t *testing.T) {
	m, ok := domain.ClaudeModel("claude-sonnet-5-5")
	if !ok || m.Name != "Claude Sonnet 5.5" || m.Price.Of(1_000_000, 1_000_000) != 12_000_000 {
		t.Errorf("sonnet = %+v %v", m, ok)
	}
	if _, ok := domain.ClaudeModel("gpt-5"); ok {
		t.Error("an unknown model was found")
	}
	if domain.ListPrice("claude-opus-5-5").Of(1_000_000, 0) != 4_000_000 {
		t.Error("Opus 5.5 is not priced at its own list price")
	}
	if domain.ClaudeName("claude-sonnet-5-5") != "Claude Sonnet 5.5" || domain.ClaudeName("claude-x") != "claude-x" {
		t.Error("names")
	}
}
