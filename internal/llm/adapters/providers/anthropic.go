package providers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	"github.com/JorisJonkers-dev/findability-explorer/internal/llm/domain"
)

// anthropicClient asks Claude through the Anthropic SDK, with structured output.
type anthropicClient struct {
	client anthropic.Client
}

func newAnthropic(key string, opts ...option.RequestOption) *anthropicClient {
	return &anthropicClient{client: anthropic.NewClient(append([]option.RequestOption{option.WithAPIKey(key)}, opts...)...)}
}

// Models lists the models the key may use, leaving out retired ones.
func (a *anthropicClient) Models(ctx context.Context) ([]domain.Model, error) {
	var out []domain.Model
	pager := a.client.Models.ListAutoPaging(ctx, anthropic.ModelListParams{})
	for pager.Next() {
		m := pager.Current()
		if m.Lifecycle == "retired" {
			continue
		}
		name := m.DisplayName
		if name == "" {
			name = m.ID
		}
		out = append(out, domain.Model{ID: m.ID, Name: name})
	}
	if err := pager.Err(); err != nil {
		return nil, describeAnthropic(err)
	}
	return out, nil
}

// Complete sends one request with a JSON-schema output format and reads the text answer.
func (a *anthropicClient) Complete(ctx context.Context, p domain.Prompt) (domain.Completion, error) {
	output := anthropic.OutputConfigParam{Format: anthropic.JSONOutputFormatParam{Schema: p.Schema}}
	if p.Model == domain.DefaultModel {
		// The default Haiku takes an effort; not every model a key may choose does.
		output.Effort = anthropic.OutputConfigEffortLow
	}
	resp, err := a.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:        anthropic.Model(p.Model),
		MaxTokens:    p.MaxTokens,
		System:       []anthropic.TextBlockParam{{Text: p.System}},
		OutputConfig: output,
		Messages:     []anthropic.MessageParam{anthropic.NewUserMessage(anthropic.NewTextBlock(p.User))},
	})
	if err != nil {
		return domain.Completion{}, describeAnthropic(err)
	}
	out := domain.Completion{InputTokens: resp.Usage.InputTokens, OutputTokens: resp.Usage.OutputTokens}
	switch resp.StopReason {
	case anthropic.StopReasonRefusal:
		return out, ErrDeclined
	case anthropic.StopReasonMaxTokens, anthropic.StopReasonModelContextWindowExceeded:
		return out, ErrCutOff
	case anthropic.StopReasonEndTurn, anthropic.StopReasonStopSequence, anthropic.StopReasonToolUse,
		anthropic.StopReasonPauseTurn:
	}
	var text strings.Builder
	for _, block := range resp.Content {
		if t, ok := block.AsAny().(anthropic.TextBlock); ok {
			text.WriteString(t.Text)
		}
	}
	out.Text = text.String()
	return out, nil
}

var (
	// ErrDeclined is returned when the model declines the request; it is a domain.ErrNoAnswer.
	ErrDeclined error = noAnswerError("the model declined the request")
	// ErrCutOff is returned when the model's answer hit the token limit; it is a domain.ErrNoAnswer.
	ErrCutOff error = noAnswerError("the model's answer was cut off")
)

// noAnswerError is an unusable answer, which errors.Is matches to domain.ErrNoAnswer.
type noAnswerError string

func (e noAnswerError) Error() string { return string(e) }

func (noAnswerError) Is(target error) bool { return target == domain.ErrNoAnswer } //nolint:errorlint // a sentinel compared by identity

// describeAnthropic turns an SDK error into a reason that names the status and the API's
// message but never echoes the request.
func describeAnthropic(err error) error {
	var apiErr *anthropic.Error
	if errors.As(err, &apiErr) {
		return statusError("Anthropic", apiErr.StatusCode, anthropicMessage(apiErr))
	}
	return fmt.Errorf("Anthropic could not be reached: %w", err) //nolint:staticcheck // a provider's name starts the sentence
}

func anthropicMessage(e *anthropic.Error) string {
	var body struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if jsonUnmarshal([]byte(e.RawJSON()), &body) == nil {
		return body.Error.Message
	}
	return ""
}

// statusError describes a provider's error status, with its message clipped.
func statusError(provider string, status int, message string) error {
	message = clip(strings.TrimSpace(message), 200)
	var what string
	switch status {
	case http.StatusUnauthorized, http.StatusForbidden:
		what = provider + " refused the API key"
	case http.StatusPaymentRequired:
		what = provider + " says the account has no credit left"
	case http.StatusTooManyRequests:
		what = provider + " is rate limiting; try again later"
	default:
		what = fmt.Sprintf("%s answered %d", provider, status)
	}
	if message != "" {
		what += " (" + strings.TrimSuffix(message, ".") + ")"
	}
	return &ProviderError{Status: status, Message: what}
}

// ProviderError is a provider's refusal or failure, as a sentence an admin can act on.
type ProviderError struct {
	Status  int
	Message string
}

func (e *ProviderError) Error() string { return e.Message }

func clip(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n-1]) + "…"
	}
	return s
}
