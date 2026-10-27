// Package openrouter implements the prompt-test Gateway over OpenRouter's API: its price list, a
// model's natural answer to a prompt with or without web search, and the fixed extractor's
// reading of an answer in the strict JSON format of the extractor package.
package openrouter

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/adapters/extractor"
	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/app"
	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain"
)

const (
	// URL is OpenRouter's API.
	URL = "https://openrouter.ai/api/v1"
	// Referer and Title identify the app to OpenRouter.
	Referer = "https://digital-strategy.jorisjonkers.dev"
	Title   = "Findability Explorer"

	maxBody      = 16 << 20
	maxAnswer    = 2000 // tokens a model may answer with
	maxExtracted = 2000
	timeout      = 3 * time.Minute
)

// Client is OpenRouter reached with one key.
type Client struct {
	base string
	key  string
	http *http.Client
}

var _ app.Gateway = (*Client)(nil)

// New returns a Client for key on base (URL in production).
func New(base, key string, client *http.Client) *Client {
	if client == nil {
		client = &http.Client{Timeout: timeout}
	}
	return &Client{base: strings.TrimSuffix(base, "/"), key: key, http: client}
}

// Models lists OpenRouter's models with their prices per token.
func (c *Client) Models(ctx context.Context) ([]domain.CatalogueModel, error) {
	var list struct {
		Data []struct {
			ID           string `json:"id"`
			Name         string `json:"name"`
			Architecture struct {
				InputModalities  []string `json:"input_modalities"`
				OutputModalities []string `json:"output_modalities"`
			} `json:"architecture"`
			Pricing struct {
				Prompt     string `json:"prompt"`
				Completion string `json:"completion"`
			} `json:"pricing"`
		} `json:"data"`
	}
	if err := c.do(ctx, http.MethodGet, "/models", nil, &list); err != nil {
		return nil, err
	}
	out := make([]domain.CatalogueModel, 0, len(list.Data))
	for _, m := range list.Data {
		prompt, err1 := strconv.ParseFloat(m.Pricing.Prompt, 64)
		completion, err2 := strconv.ParseFloat(m.Pricing.Completion, 64)
		if m.ID == "" || err1 != nil || err2 != nil {
			continue
		}
		out = append(out, domain.CatalogueModel{
			ID: m.ID, Name: m.Name, PromptPrice: prompt, CompletionPrice: completion,
			InputModalities: m.Architecture.InputModalities, OutputModalities: m.Architecture.OutputModalities,
		})
	}
	return out, nil
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type request struct {
	Model          string           `json:"model"`
	Messages       []message        `json:"messages"`
	MaxTokens      int              `json:"max_tokens"`
	Plugins        []map[string]any `json:"plugins,omitempty"`
	ResponseFormat map[string]any   `json:"response_format,omitempty"`
	Usage          map[string]bool  `json:"usage"`
}

type response struct {
	Choices []struct {
		Message struct {
			Content     string `json:"content"`
			Annotations []struct {
				Type        string `json:"type"`
				URLCitation struct {
					URL string `json:"url"`
				} `json:"url_citation"`
			} `json:"annotations"`
		} `json:"message"`
		Error *apiError `json:"error"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int64   `json:"prompt_tokens"`
		CompletionTokens int64   `json:"completion_tokens"`
		Cost             float64 `json:"cost"`
	} `json:"usage"`
	Error *apiError `json:"error"`
}

type apiError struct {
	Code    json.RawMessage `json:"code"`
	Message string          `json:"message"`
}

// Ask asks model one prompt, as a buyer would, with OpenRouter's web search when web is true.
// Nothing about format is asked, so the answer is what a person would see.
func (c *Client) Ask(ctx context.Context, model, prompt string, web bool) (domain.Reply, error) {
	req := request{Model: model, Messages: []message{{Role: "user", Content: prompt}}, MaxTokens: maxAnswer, Usage: map[string]bool{"include": true}}
	if web {
		req.Plugins = []map[string]any{{"id": "web"}}
	}
	resp, err := c.complete(ctx, req)
	reply := domain.Reply{InputTokens: resp.Usage.PromptTokens, OutputTokens: resp.Usage.CompletionTokens, Cost: domain.MicrosOf(resp.Usage.Cost)}
	if err != nil {
		return reply, err
	}
	m := resp.Choices[0].Message
	reply.Text = m.Content
	for _, a := range m.Annotations {
		if a.Type == "url_citation" && a.URLCitation.URL != "" {
			reply.Citations = append(reply.Citations, a.URLCitation.URL)
		}
	}
	return reply, nil
}

// complete sends a chat completion and reads its answer: a 429, or an error the answer carries
// with code 429, is domain.ErrRateLimited.
func (c *Client) complete(ctx context.Context, req request) (response, error) {
	var resp response
	if err := c.do(ctx, http.MethodPost, "/chat/completions", req, &resp); err != nil {
		return resp, err
	}
	e := resp.Error
	if e == nil && len(resp.Choices) > 0 {
		e = resp.Choices[0].Error
	}
	switch {
	case e != nil && strings.Trim(string(e.Code), `"`) == "429":
		return resp, fmt.Errorf("%w: %s", domain.ErrRateLimited, e.Message)
	case e != nil:
		return resp, fmt.Errorf("OpenRouter: %s", e.Message)
	case len(resp.Choices) == 0:
		return resp, errors.New("OpenRouter answered without a choice")
	}
	return resp, nil
}

// Extract has model read an answer in the strict format.
func (c *Client) Extract(ctx context.Context, model string, in app.ExtractInput) (domain.Extracted, domain.Reply, error) {
	req := request{
		Model: model, Messages: []message{{Role: "system", Content: extractor.System}, {Role: "user", Content: extractor.User(in)}},
		MaxTokens: maxExtracted, Usage: map[string]bool{"include": true},
		ResponseFormat: map[string]any{"type": "json_schema", "json_schema": map[string]any{"name": extractor.SchemaName, "strict": true, "schema": extractor.Schema()}},
	}
	resp, err := c.complete(ctx, req)
	usage := domain.Reply{InputTokens: resp.Usage.PromptTokens, OutputTokens: resp.Usage.CompletionTokens, Cost: domain.MicrosOf(resp.Usage.Cost)}
	if err != nil {
		return domain.Extracted{}, usage, err
	}
	x, err := extractor.Parse(resp.Choices[0].Message.Content)
	return x, usage, err
}

// do sends one request with the key and decodes a 2xx answer into out; a 429 is
// domain.ErrRateLimited.
func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
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
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HTTP-Referer", Referer)
	req.Header.Set("X-Title", Title)
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("OpenRouter could not be reached: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	if err != nil {
		return fmt.Errorf("read OpenRouter's answer: %w", err)
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return fmt.Errorf("%w: %s", domain.ErrRateLimited, errorText(raw))
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("OpenRouter answered %d: %s", resp.StatusCode, errorText(raw))
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("read OpenRouter's answer: %w", err)
	}
	return nil
}

// errorText is the message of an error body, {"error":{"message"}}, or the body itself, clipped.
func errorText(raw []byte) string {
	var body struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(raw, &body) == nil && body.Error.Message != "" {
		return body.Error.Message
	}
	return clip(strings.TrimSpace(string(raw)), 200)
}

func clip(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n])
	}
	return s
}
