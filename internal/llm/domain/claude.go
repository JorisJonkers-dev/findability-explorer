package domain

import (
	"slices"
	"strings"
)

// ClaudeInfo is one Claude model a Claude step may use: its id on Anthropic's API, its name, and
// its list price.
type ClaudeInfo struct {
	ID    string
	Name  string
	Price Price
}

// claudeModels are the Claude models on Anthropic's API with their list prices per million tokens,
// for prompts up to 100,000 tokens, as OpenRouter's catalogue lists them on 28 October 2026.
var claudeModels = []ClaudeInfo{ //nolint:gochecknoglobals // a constant table
	{ID: DefaultModel, Name: "Claude Haiku 5.5", Price: Price{InputPerMillion: 100_000, OutputPerMillion: 500_000}},
	{ID: "claude-haiku-4-5", Name: "Claude Haiku 4.5", Price: Price{InputPerMillion: 1_000_000, OutputPerMillion: 5_000_000}},
	{ID: "claude-sonnet-5-5", Name: "Claude Sonnet 5.5", Price: Price{InputPerMillion: 2_000_000, OutputPerMillion: 10_000_000}},
	{ID: "claude-sonnet-5", Name: "Claude Sonnet 5", Price: Price{InputPerMillion: 2_000_000, OutputPerMillion: 10_000_000}},
	{ID: "claude-sonnet-4-6", Name: "Claude Sonnet 4.6", Price: Price{InputPerMillion: 3_000_000, OutputPerMillion: 15_000_000}},
	{ID: "claude-opus-5-5", Name: "Claude Opus 5.5", Price: Price{InputPerMillion: 4_000_000, OutputPerMillion: 20_000_000}},
	{ID: "claude-opus-5", Name: "Claude Opus 5", Price: Price{InputPerMillion: 5_000_000, OutputPerMillion: 25_000_000}},
	{ID: "claude-opus-4-8", Name: "Claude Opus 4.8", Price: Price{InputPerMillion: 5_000_000, OutputPerMillion: 25_000_000}},
	{ID: "claude-fable-5-1", Name: "Claude Fable 5.1", Price: Price{InputPerMillion: 10_000_000, OutputPerMillion: 50_000_000}},
}

// ClaudeModels returns the Claude models a step may use, cheapest first, Claude Haiku 5.5 the default.
func ClaudeModels() []ClaudeInfo { return slices.Clone(claudeModels) }

// ClaudeModel returns the Claude model with this id; false when there is none.
func ClaudeModel(id string) (ClaudeInfo, bool) {
	i := slices.IndexFunc(claudeModels, func(m ClaudeInfo) bool { return m.ID == id })
	if i < 0 {
		return ClaudeInfo{}, false
	}
	return claudeModels[i], true
}

// ClaudeName is a model's name for people, or its id when it is no Claude model listed here.
func ClaudeName(id string) string {
	if m, ok := ClaudeModel(id); ok {
		return m.Name
	}
	return id
}

// ClaudeIDOf is a Claude model's id however a job recorded it: bare ("claude-sonnet-5-5"), or as
// Describe labels a model on the Anthropic key ("anthropic: claude-sonnet-5-5"), as scans and
// readings record it. Anything else is returned as it is, and ListPrice prices it, like an empty
// model, as Claude Haiku 5.5.
func ClaudeIDOf(model string) string {
	return strings.TrimPrefix(model, string(Anthropic)+": ")
}
