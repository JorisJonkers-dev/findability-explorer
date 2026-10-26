// Package providers implements the language-model Client port for Anthropic (through its SDK)
// and for OpenRouter and Mistral (through their OpenAI-compatible chat completions APIs).
package providers

import (
	"net/http"
	"time"

	"github.com/anthropics/anthropic-sdk-go/option"

	"github.com/JorisJonkers-dev/findability-explorer/internal/llm/domain"
)

const (
	// OpenRouterURL is OpenRouter's API.
	OpenRouterURL = "https://openrouter.ai/api/v1"
	// MistralURL is Mistral's API.
	MistralURL = "https://api.mistral.ai/v1"
	// AnthropicURL is Anthropic's API, which the Admin key's reports are read from.
	AnthropicURL = "https://api.anthropic.com"
	// Referer and Title identify the app to OpenRouter.
	Referer = "https://digital-strategy.jorisjonkers.dev"
	Title   = "Findability Explorer"

	requestTimeout = 3 * time.Minute
)

// Connector makes a Client per provider and key.
type Connector struct {
	// HTTP is the client for OpenRouter and Mistral.
	HTTP *http.Client
	// AnthropicOptions come after the key, such as a base URL in tests.
	AnthropicOptions []option.RequestOption
	OpenRouterURL    string
	MistralURL       string
	AnthropicURL     string
}

var (
	_ domain.Connector = (*Connector)(nil)
	_ domain.Reporter  = (*Connector)(nil)
)

// New returns a Connector for the providers' real APIs.
func New() *Connector {
	return &Connector{
		HTTP: &http.Client{Timeout: requestTimeout}, OpenRouterURL: OpenRouterURL, MistralURL: MistralURL, AnthropicURL: AnthropicURL,
	}
}

// Connect returns the Client for p with key.
func (c *Connector) Connect(p domain.Provider, key string) domain.Client {
	switch p {
	case domain.OpenRouter:
		return &chat{
			name: "OpenRouter", base: c.OpenRouterURL, key: key, http: c.HTTP, check: "/key",
			headers: map[string]string{"HTTP-Referer": Referer, "X-Title": Title},
		}
	case domain.Mistral:
		return &chat{name: "Mistral", base: c.MistralURL, key: key, http: c.HTTP}
	case domain.AnthropicAdmin:
		return &admin{base: c.AnthropicURL, key: key, http: c.HTTP}
	case domain.Anthropic:
	}
	return newAnthropic(key, c.AnthropicOptions...)
}
