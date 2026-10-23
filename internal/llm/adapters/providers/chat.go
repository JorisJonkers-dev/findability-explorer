package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/JorisJonkers-dev/findability-explorer/internal/llm/domain"
)

// maxBody bounds what is read of a provider's answer; a model list is the largest.
const maxBody = 16 << 20

var jsonUnmarshal = json.Unmarshal //nolint:gochecknoglobals // an alias, so both clients read JSON the same way

// chat is an OpenAI-compatible chat completions API: OpenRouter and Mistral.
type chat struct {
	name    string
	base    string
	key     string
	http    *http.Client
	headers map[string]string
	// check is a path that needs the key, asked before /models where /models does not.
	check string
}

// Models lists the models the key may use; the check path (OpenRouter's /key) proves the key
// first, since OpenRouter lists its models to anyone.
func (c *chat) Models(ctx context.Context) ([]domain.Model, error) {
	if c.check != "" {
		if err := c.do(ctx, http.MethodGet, c.check, nil, &struct{}{}); err != nil {
			return nil, err
		}
	}
	var list struct {
		Data []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"data"`
	}
	if err := c.do(ctx, http.MethodGet, "/models", nil, &list); err != nil {
		return nil, err
	}
	out := make([]domain.Model, 0, len(list.Data))
	for _, m := range list.Data {
		if m.ID == "" {
			continue
		}
		name := m.Name
		if name == "" {
			name = m.ID
		}
		out = append(out, domain.Model{ID: m.ID, Name: name})
	}
	return out, nil
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type completionRequest struct {
	Model          string         `json:"model"`
	Messages       []message      `json:"messages"`
	MaxTokens      int64          `json:"max_tokens"`
	ResponseFormat map[string]any `json:"response_format"`
}

type completionResponse struct {
	Choices []struct {
		FinishReason string `json:"finish_reason"`
		Message      struct {
			Content json.RawMessage `json:"content"`
			Refusal string          `json:"refusal"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int64 `json:"prompt_tokens"`
		CompletionTokens int64 `json:"completion_tokens"`
	} `json:"usage"`
}

// Complete asks for an answer that follows the prompt's JSON schema (response_format
// json_schema, strict). A model that does not take a schema (400 or 422) is asked once more in
// JSON mode, with the schema in the system prompt.
func (c *chat) Complete(ctx context.Context, p domain.Prompt) (domain.Completion, error) {
	req := completionRequest{
		Model:     p.Model,
		Messages:  []message{{Role: "system", Content: p.System}, {Role: "user", Content: p.User}},
		MaxTokens: p.MaxTokens,
		ResponseFormat: map[string]any{
			"type":        "json_schema",
			"json_schema": map[string]any{"name": p.SchemaName, "strict": true, "schema": p.Schema},
		},
	}
	var resp completionResponse
	err := c.do(ctx, http.MethodPost, "/chat/completions", req, &resp)
	var pe *ProviderError
	if errors.As(err, &pe) && (pe.Status == http.StatusBadRequest || pe.Status == http.StatusUnprocessableEntity) {
		schema, _ := json.Marshal(p.Schema) // a map of JSON values always marshals
		req.Messages[0].Content = p.System + "\n\nAnswer with one JSON object that follows this JSON schema, and nothing else:\n" + string(schema)
		req.ResponseFormat = map[string]any{"type": "json_object"}
		err = c.do(ctx, http.MethodPost, "/chat/completions", req, &resp)
	}
	if err != nil {
		return domain.Completion{}, err
	}
	out := domain.Completion{InputTokens: resp.Usage.PromptTokens, OutputTokens: resp.Usage.CompletionTokens}
	if len(resp.Choices) == 0 {
		return out, fmt.Errorf("%s answered without a choice", c.name)
	}
	choice := resp.Choices[0]
	switch {
	case choice.Message.Refusal != "", choice.FinishReason == "content_filter":
		return out, ErrDeclined
	case choice.FinishReason == "length", choice.FinishReason == "model_length":
		return out, ErrCutOff
	}
	out.Text = text(choice.Message.Content)
	return out, nil
}

// text reads a message's content: a string, or (Mistral) a list of chunks of which the text ones count.
func text(raw json.RawMessage) string {
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	var chunks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	_ = json.Unmarshal(raw, &chunks)
	var b strings.Builder
	for _, ch := range chunks {
		if ch.Type == "text" {
			b.WriteString(ch.Text)
		}
	}
	return b.String()
}

// do sends one request with the key and decodes a 2xx answer into out.
func (c *chat) do(ctx context.Context, method, path string, body, out any) error {
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode the request: %w", err)
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, reader)
	if err != nil {
		return fmt.Errorf("build the request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.key)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("%s could not be reached: %w", c.name, err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	if err != nil {
		return fmt.Errorf("read %s's answer: %w", c.name, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return statusError(c.name, resp.StatusCode, errorMessage(raw))
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("read %s's answer: %w", c.name, err)
	}
	return nil
}

// errorMessage finds the message in an error body: OpenRouter's {"error":{"message"}}, Mistral's
// {"message"} or {"detail"}.
func errorMessage(raw []byte) string {
	var body struct {
		Error   json.RawMessage `json:"error"`
		Message string          `json:"message"`
		Detail  json.RawMessage `json:"detail"`
	}
	if json.Unmarshal(raw, &body) != nil {
		return ""
	}
	var nested struct {
		Message string `json:"message"`
	}
	if json.Unmarshal(body.Error, &nested) == nil && nested.Message != "" {
		return nested.Message
	}
	var s string
	switch {
	case body.Message != "":
		return body.Message
	case json.Unmarshal(body.Error, &s) == nil:
		return s
	case json.Unmarshal(body.Detail, &s) == nil:
		return s
	}
	return ""
}
