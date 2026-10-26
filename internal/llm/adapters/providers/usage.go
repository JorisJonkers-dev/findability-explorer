package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/JorisJonkers-dev/findability-explorer/internal/llm/domain"
)

// OpenRouter reports an OpenRouter key's own spend: GET /key.
func (c *Connector) OpenRouter(ctx context.Context, key string) (domain.Usage, error) {
	var body struct {
		Data struct {
			Usage          float64  `json:"usage"`
			UsageDaily     float64  `json:"usage_daily"`
			UsageWeekly    float64  `json:"usage_weekly"`
			UsageMonthly   float64  `json:"usage_monthly"`
			Limit          *float64 `json:"limit"`
			LimitRemaining *float64 `json:"limit_remaining"`
			LimitReset     string   `json:"limit_reset"`
			Free           *struct {
				Used      int `json:"used"`
				Limit     int `json:"limit"`
				Remaining int `json:"remaining"`
			} `json:"free_model_daily_requests"`
		} `json:"data"`
	}
	ch := &chat{name: "OpenRouter", base: c.OpenRouterURL, key: key, http: c.HTTP, headers: map[string]string{"HTTP-Referer": Referer, "X-Title": Title}}
	if err := ch.do(ctx, http.MethodGet, "/key", nil, &body); err != nil {
		return domain.Usage{}, err
	}
	d := body.Data
	u := domain.Usage{
		Source: domain.UsageOpenRouter, Total: &d.Usage, Daily: &d.UsageDaily, Weekly: &d.UsageWeekly, Monthly: &d.UsageMonthly,
		Limit: d.Limit, LimitRemaining: d.LimitRemaining, LimitReset: d.LimitReset,
		Message: "OpenRouter's own figures for this key, in dollars.",
	}
	if d.Free != nil {
		u.Free = &domain.FreeRequests{Used: d.Free.Used, Limit: d.Free.Limit, Remaining: d.Free.Remaining}
	}
	return u, nil
}

// Anthropic reads, with an Admin key, this month's tokens per model of the key (matched to the
// organisation's keys by its hint) and the organisation's cost since since.
func (c *Connector) Anthropic(ctx context.Context, adminKey, key string, since time.Time) (domain.Usage, error) {
	a := &admin{base: c.AnthropicURL, key: adminKey, http: c.HTTP}
	u := domain.Usage{Source: domain.UsageAnthropic, Since: since, Models: []domain.ModelTokens{}}
	query := url.Values{"starting_at": {since.UTC().Format(time.RFC3339)}, "bucket_width": {"1d"}, "limit": {"31"}, "group_by[]": {"model"}}
	u.Message = "Anthropic's usage report for the whole organisation this month; the cost is the organisation's."
	if key != "" {
		id, err := a.keyID(ctx, key)
		if err != nil {
			return domain.Usage{}, err
		}
		if id == "" {
			u.Message = "The Admin key's organisation has no key like this one, so Anthropic reports nothing for it."
			return u, nil
		}
		query.Set("api_key_ids[]", id)
		u.Message = "Anthropic's usage report for this key this month; the cost is the whole organisation's."
	}
	var report struct {
		Data []struct {
			Results []struct {
				Model                string `json:"model"`
				UncachedInputTokens  int64  `json:"uncached_input_tokens"`
				CacheReadInputTokens int64  `json:"cache_read_input_tokens"`
				OutputTokens         int64  `json:"output_tokens"`
				CacheCreation        struct {
					OneHour  int64 `json:"ephemeral_1h_input_tokens"`
					FiveMins int64 `json:"ephemeral_5m_input_tokens"`
				} `json:"cache_creation"`
			} `json:"results"`
		} `json:"data"`
	}
	if err := a.get(ctx, "/v1/organizations/usage_report/messages?"+query.Encode(), &report); err != nil {
		return domain.Usage{}, err
	}
	for _, bucket := range report.Data {
		for _, r := range bucket.Results {
			i := slices.IndexFunc(u.Models, func(m domain.ModelTokens) bool { return m.Model == r.Model })
			if i < 0 {
				u.Models = append(u.Models, domain.ModelTokens{Model: r.Model})
				i = len(u.Models) - 1
			}
			u.Models[i].InputTokens += r.UncachedInputTokens + r.CacheCreation.OneHour + r.CacheCreation.FiveMins
			u.Models[i].CacheRead += r.CacheReadInputTokens
			u.Models[i].OutputTokens += r.OutputTokens
		}
	}
	cost, err := a.cost(ctx, since)
	if err != nil {
		return domain.Usage{}, err
	}
	u.OrganisationCost = &cost
	return u, nil
}

// admin is Anthropic's Admin API, reached with an Admin key; it reads reports and never asks a model.
type admin struct {
	base string
	key  string
	http *http.Client
}

// Models checks the Admin key by listing one of the organisation's keys; an Admin key may use no
// model.
func (a *admin) Models(ctx context.Context) ([]domain.Model, error) {
	var list struct{}
	if err := a.get(ctx, "/v1/organizations/api_keys?limit=1", &list); err != nil {
		return nil, err
	}
	return []domain.Model{}, nil
}

// Complete refuses: an Admin key only reads reports.
func (a *admin) Complete(context.Context, domain.Prompt) (domain.Completion, error) {
	return domain.Completion{}, domain.ErrReportsOnly
}

// keyID finds the id of the organisation's key that key is, by the hint Anthropic shows of it
// ("sk-ant-api03-R2D...igAA"); "" when none matches.
func (a *admin) keyID(ctx context.Context, key string) (string, error) {
	var list struct {
		Data []struct {
			ID   string `json:"id"`
			Hint string `json:"partial_key_hint"`
		} `json:"data"`
	}
	if err := a.get(ctx, "/v1/organizations/api_keys?limit=1000", &list); err != nil {
		return "", err
	}
	for _, k := range list.Data {
		prefix, suffix, ok := strings.Cut(k.Hint, "...")
		if ok && prefix != "" && strings.HasPrefix(key, prefix) && strings.HasSuffix(key, suffix) {
			return k.ID, nil
		}
	}
	return "", nil
}

// cost sums the organisation's cost report since since, in dollars; amounts come in cents.
func (a *admin) cost(ctx context.Context, since time.Time) (float64, error) {
	var report struct {
		Data []struct {
			Results []struct {
				Amount string `json:"amount"`
			} `json:"results"`
		} `json:"data"`
	}
	query := url.Values{"starting_at": {since.UTC().Format(time.RFC3339)}, "bucket_width": {"1d"}, "limit": {"31"}}
	if err := a.get(ctx, "/v1/organizations/cost_report?"+query.Encode(), &report); err != nil {
		return 0, err
	}
	cents := 0.0
	for _, bucket := range report.Data {
		for _, r := range bucket.Results {
			if v, err := strconv.ParseFloat(r.Amount, 64); err == nil {
				cents += v
			}
		}
	}
	return cents / 100, nil
}

func (a *admin) get(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.base+path, nil)
	if err != nil {
		return fmt.Errorf("build the request: %w", err)
	}
	req.Header.Set("X-Api-Key", a.key)
	req.Header.Set("anthropic-version", "2023-06-01")
	resp, err := a.http.Do(req)
	if err != nil {
		return fmt.Errorf("Anthropic could not be reached: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	if err != nil {
		return fmt.Errorf("read Anthropic's answer: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return statusError("Anthropic", resp.StatusCode, errorMessage(raw))
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("read Anthropic's answer: %w", err)
	}
	return nil
}
