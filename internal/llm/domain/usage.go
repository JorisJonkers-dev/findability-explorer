package domain

import (
	"context"
	"time"
)

// UsageSource is where a key's usage figures come from.
type UsageSource string

const (
	// UsageOpenRouter means OpenRouter reported the key's own spend.
	UsageOpenRouter UsageSource = "openrouter"
	// UsageAnthropic means Anthropic's usage and cost reports, read with an Admin key.
	UsageAnthropic UsageSource = "anthropic"
	// UsageNone means the provider reports nothing for the key; Message says why.
	UsageNone UsageSource = "none"
)

// FreeRequests is OpenRouter's count of the day's requests to free models.
type FreeRequests struct {
	Used      int
	Limit     int
	Remaining int
}

// ModelTokens is the tokens a key used with one model this month, as Anthropic reports them.
type ModelTokens struct {
	Model        string
	InputTokens  int64
	OutputTokens int64
	CacheRead    int64
}

// Usage is what a provider reports a key used, in dollars where it says.
type Usage struct {
	Source  UsageSource
	Message string
	// OpenRouter: the key's spend all time, today, this week and this month (UTC), its limit and
	// what is left of it (nil when it has none), and the day's free-model requests.
	Total, Daily, Weekly, Monthly *float64
	Limit, LimitRemaining         *float64
	LimitReset                    string
	Free                          *FreeRequests
	// Anthropic: this month's tokens per model of the key, and the whole organisation's cost
	// this month (Anthropic does not report cost per key).
	Models           []ModelTokens
	OrganisationCost *float64
	// Since is when the month the figures cover started.
	Since time.Time
}

// Reporter is the port to the providers' usage reports.
type Reporter interface {
	// OpenRouter reports an OpenRouter key's own spend.
	OpenRouter(ctx context.Context, key string) (Usage, error)
	// Anthropic reads, with adminKey, this month's tokens of the key per model (key empty: the
	// whole organisation's) and the organisation's cost since since.
	Anthropic(ctx context.Context, adminKey, key string, since time.Time) (Usage, error)
}
