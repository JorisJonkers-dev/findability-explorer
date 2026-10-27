// Package app holds the language-model context's use cases: adding, testing and deleting API keys,
// choosing the credential and model product matching uses, and handing the analysis context the
// client to use now. They are the only entry points the inbound adapters call.
package app

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode"

	"github.com/JorisJonkers-dev/findability-explorer/internal/llm/domain"
)

// Repository stores the added keys (sealed), the decision about the active model, and the events.
type Repository interface {
	// Add stores c with its secret key.
	Add(ctx context.Context, c domain.Credential, key string) error
	// List returns every stored credential, oldest first, without secrets.
	List(ctx context.Context) ([]domain.Credential, error)
	// Get returns one stored credential without its secret, or domain.ErrNotFound.
	Get(ctx context.Context, id string) (domain.Credential, error)
	// Secret returns a stored credential's provider and key, domain.ErrNotFound, or
	// domain.ErrUnreadable.
	Secret(ctx context.Context, id string) (domain.Provider, string, error)
	// Delete removes a stored credential, or returns domain.ErrNotFound. When it was the active
	// one, the decision about the active model is reset in the same transaction, and wasActive
	// says so.
	Delete(ctx context.Context, id string) (wasActive bool, err error)
	Setting(ctx context.Context) (domain.Setting, error)
	SaveSetting(ctx context.Context, s domain.Setting, actor string, at time.Time) error
	AddEvent(ctx context.Context, e domain.Event) error
	// Events returns the latest limit events, newest first.
	Events(ctx context.Context, limit int) ([]domain.Event, error)
	// LastTests returns the latest test of each credential, by id.
	LastTests(ctx context.Context) (map[string]domain.Test, error)
}

// Budget is what analyses may spend on the language model, and what one scan uses on average.
type Budget struct {
	MonthlyTokens int64
	UsedThisMonth int64
	// AveragePerScan is the tokens one scan's matching used on average, over ScansMatched scans
	// the language model answered for; both are 0 before the first.
	AveragePerScan int64
	ScansMatched   int
}

// Spending is what the analyses used: this month's tokens, and the average of a scan.
type Spending struct {
	ThisMonth      int64
	AveragePerScan int64
	ScansMatched   int
}

// Spent returns what analyses used; the analysis context provides it.
type Spent func(ctx context.Context) (Spending, error)

// Config is the service's fixed settings.
type Config struct {
	// VaultKey is ANTHROPIC_API_KEY, empty when Vault delivers none.
	VaultKey      string
	MonthlyTokens int64
}

// Settings is everything the admin settings show.
type Settings struct {
	Credentials []domain.Credential
	Choice      domain.Choice
	Active      *domain.Active
	// Haiku is the Anthropic key and model product matching and the reading of Prompt test answers
	// use instead of Active while one works; nil when no Anthropic key does.
	Haiku *domain.Active
	// MatchingEnabled is true when the next analysis will match products: on Haiku, or on the
	// active model when its key is readable.
	MatchingEnabled bool
	Budget          Budget
	Events          []domain.Event
}

// Service runs the language-model use cases.
type Service struct {
	repo      Repository
	connector domain.Connector
	reporter  domain.Reporter
	spent     Spent
	cfg       Config
	now       func() time.Time
	newID     func() string
}

// New returns a Service; reporter reads what the providers report a key used, and newID makes
// the id of an added credential.
func New(repo Repository, connector domain.Connector, reporter domain.Reporter, spent Spent, cfg Config, now func() time.Time,
	newID func() string,
) *Service {
	return &Service{repo: repo, connector: connector, reporter: reporter, spent: spent, cfg: cfg, now: now, newID: newID}
}

// Settings returns the credentials, the active model, the budget and the latest events.
func (s *Service) Settings(ctx context.Context) (Settings, error) {
	creds, err := s.credentials(ctx)
	if err != nil {
		return Settings{}, err
	}
	setting, err := s.repo.Setting(ctx)
	if err != nil {
		return Settings{}, fmt.Errorf("read the active model: %w", err)
	}
	choice, active := domain.Resolve(setting, s.cfg.VaultKey != "")
	spent, err := s.spent(ctx)
	if err != nil {
		return Settings{}, fmt.Errorf("sum this month's tokens: %w", err)
	}
	events, err := s.repo.Events(ctx, domain.MaxEvents)
	if err != nil {
		return Settings{}, fmt.Errorf("list events: %w", err)
	}
	enabled := active != nil && slices.ContainsFunc(creds, func(c domain.Credential) bool {
		return c.ID == active.CredentialID && c.Readable
	})
	var haiku *domain.Active
	if c, ok := domain.AnthropicKey(creds); ok {
		haiku = &domain.Active{CredentialID: c.ID, Model: domain.DefaultModel}
	}
	return Settings{
		Credentials: creds, Choice: choice, Active: active, Haiku: haiku, MatchingEnabled: enabled || haiku != nil, Events: events,
		Budget: Budget{
			MonthlyTokens: s.cfg.MonthlyTokens, UsedThisMonth: spent.ThisMonth,
			AveragePerScan: spent.AveragePerScan, ScansMatched: spent.ScansMatched,
		},
	}, nil
}

// credentials is the Vault credential, when there is one, then the stored ones oldest first,
// each with its latest test.
func (s *Service) credentials(ctx context.Context) ([]domain.Credential, error) {
	stored, err := s.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list credentials: %w", err)
	}
	tests, err := s.repo.LastTests(ctx)
	if err != nil {
		return nil, fmt.Errorf("list tests: %w", err)
	}
	out := make([]domain.Credential, 0, len(stored)+1)
	if s.cfg.VaultKey != "" {
		out = append(out, domain.VaultCredential(s.cfg.VaultKey))
	}
	out = append(out, stored...)
	for i := range out {
		if t, ok := tests[out[i].ID]; ok {
			out[i].LastTest = &t
		}
	}
	return out, nil
}

// Add checks the key with its provider and stores it. A key the provider refuses is not stored
// and returns an error wrapping domain.ErrKeyRefused and the provider's reason.
func (s *Service) Add(ctx context.Context, actor, provider, label, key string) (domain.Credential, error) {
	nc, err := domain.CheckNew(provider, label, key)
	if err != nil {
		return domain.Credential{}, err
	}
	models, err := s.connector.Connect(nc.Provider, nc.Key).Models(ctx)
	if err != nil {
		return domain.Credential{}, fmt.Errorf("%w: %w", domain.ErrKeyRefused, err)
	}
	now := s.now()
	test := domain.Test{OK: true, At: now, Message: acceptedFor(nc.Provider, len(models))}
	c := domain.Credential{
		ID: s.newID(), Provider: nc.Provider, Label: nc.Label, Source: domain.SourceStored,
		Fingerprint: domain.Fingerprint(nc.Key), Readable: true, CreatedBy: actor, CreatedAt: now, LastTest: &test,
	}
	if err := s.repo.Add(ctx, c, nc.Key); err != nil {
		return domain.Credential{}, fmt.Errorf("store credential: %w", err)
	}
	ok := true
	if err := s.record(ctx, domain.Event{
		At: now, Actor: actor, Action: domain.ActionCredentialAdded, CredentialID: c.ID, Provider: c.Provider, OK: &ok,
		Message: fmt.Sprintf("Added %s key %q (%s). %s", c.Provider, c.Label, c.Fingerprint, test.Message),
	}); err != nil {
		return domain.Credential{}, err
	}
	return c, nil
}

// Delete removes a stored key. When it was the active one, matching falls back to the Vault key
// with the default model, or is off without one.
func (s *Service) Delete(ctx context.Context, actor, id string) error {
	if id == domain.VaultID {
		return domain.ErrVaultReadOnly
	}
	c, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}
	wasActive, err := s.repo.Delete(ctx, id)
	if err != nil {
		return err
	}
	msg := fmt.Sprintf("Deleted %s key %q (%s).", c.Provider, c.Label, c.Fingerprint)
	if wasActive {
		msg += " It was the active key; matching falls back to the default."
	}
	return s.record(ctx, domain.Event{
		At: s.now(), Actor: actor, Action: domain.ActionCredentialDeleted, CredentialID: id, Provider: c.Provider, Message: msg,
	})
}

// Test asks the provider whether it accepts the key and records the outcome. A refused key is a
// Test that is not OK, not an error.
func (s *Service) Test(ctx context.Context, actor, id string) (domain.Test, error) {
	provider, key, err := s.secret(ctx, id)
	if err != nil {
		return domain.Test{}, err
	}
	models, err := s.connector.Connect(provider, key).Models(ctx)
	test := domain.Test{OK: err == nil, At: s.now()}
	if err != nil {
		test.Message = domain.Clip(capitalise(err.Error()))
	} else {
		test.Message = acceptedFor(provider, len(models))
	}
	ok := test.OK
	if err := s.record(ctx, domain.Event{
		At: test.At, Actor: actor, Action: domain.ActionCredentialTested, CredentialID: id, Provider: provider, OK: &ok,
		Message: test.Message,
	}); err != nil {
		return domain.Test{}, err
	}
	return test, nil
}

// Models lists the models the key may use, by id. A provider that fails or refuses returns an
// error wrapping domain.ErrProvider.
func (s *Service) Models(ctx context.Context, id string) ([]domain.Model, error) {
	provider, key, err := s.secret(ctx, id)
	if err != nil {
		return nil, err
	}
	models, err := s.connector.Connect(provider, key).Models(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", domain.ErrProvider, err)
	}
	slices.SortFunc(models, func(a, b domain.Model) int { return strings.Compare(a.ID, b.ID) })
	return models, nil
}

// SetActive makes product matching use the credential and model. The credential must exist and
// be readable.
func (s *Service) SetActive(ctx context.Context, actor, credentialID, model string) (Settings, error) {
	model, err := domain.CheckModel(model)
	if err != nil {
		return Settings{}, err
	}
	provider, _, err := s.secret(ctx, credentialID)
	if err != nil {
		return Settings{}, err
	}
	if provider == domain.AnthropicAdmin {
		return Settings{}, domain.ErrReportsOnly
	}
	now := s.now()
	active := domain.Active{CredentialID: credentialID, Model: model}
	if err := s.repo.SaveSetting(ctx, domain.Setting{Mode: domain.ModeChosen, Active: active}, actor, now); err != nil {
		return Settings{}, fmt.Errorf("save the active model: %w", err)
	}
	if err := s.record(ctx, domain.Event{
		At: now, Actor: actor, Action: domain.ActionActiveSet, CredentialID: credentialID, Provider: provider, Model: model,
		Message: "Product matching uses " + domain.Describe(provider, model) + ".",
	}); err != nil {
		return Settings{}, err
	}
	return s.Settings(ctx)
}

// ClearActive turns product matching off until a credential and model are chosen again.
func (s *Service) ClearActive(ctx context.Context, actor string) (Settings, error) {
	now := s.now()
	if err := s.repo.SaveSetting(ctx, domain.Setting{Mode: domain.ModeOff}, actor, now); err != nil {
		return Settings{}, fmt.Errorf("save the active model: %w", err)
	}
	if err := s.record(ctx, domain.Event{
		At: now, Actor: actor, Action: domain.ActionActiveCleared, Message: "Product matching is off.",
	}); err != nil {
		return Settings{}, err
	}
	return s.Settings(ctx)
}

// Current returns the client and model product matching uses now, and the provider; a nil
// Client means matching is off. An active credential that cannot be read is an error.
func (s *Service) Current(ctx context.Context) (domain.Client, domain.Provider, string, error) {
	setting, err := s.repo.Setting(ctx)
	if err != nil {
		return nil, "", "", fmt.Errorf("read the active model: %w", err)
	}
	_, active := domain.Resolve(setting, s.cfg.VaultKey != "")
	if active == nil {
		return nil, "", "", nil
	}
	provider, key, err := s.secret(ctx, active.CredentialID)
	if err != nil {
		return nil, "", "", fmt.Errorf("the active credential: %w", err)
	}
	return s.connector.Connect(provider, key), provider, active.Model, nil
}

// ErrNotOpenRouter is returned when a credential other than an OpenRouter key is asked for one.
var ErrNotOpenRouter = errors.New("the credential is no OpenRouter key")

// OpenRouterKey returns the secret of a stored OpenRouter key, for the Prompt test, which reaches
// OpenRouter itself; domain.ErrNotFound, domain.ErrUnreadable or ErrNotOpenRouter otherwise.
func (s *Service) OpenRouterKey(ctx context.Context, id string) (string, error) {
	provider, key, err := s.secret(ctx, id)
	if err != nil {
		return "", err
	}
	if provider != domain.OpenRouter {
		return "", ErrNotOpenRouter
	}
	return key, nil
}

func (s *Service) secret(ctx context.Context, id string) (domain.Provider, string, error) {
	if id == domain.VaultID {
		if s.cfg.VaultKey == "" {
			return "", "", domain.ErrNotFound
		}
		return domain.Anthropic, s.cfg.VaultKey, nil
	}
	return s.repo.Secret(ctx, id)
}

func (s *Service) record(ctx context.Context, e domain.Event) error {
	e.Message = domain.Clip(e.Message)
	if err := s.repo.AddEvent(ctx, e); err != nil {
		return fmt.Errorf("record event: %w", err)
	}
	return nil
}

func accepted(models int) string {
	return fmt.Sprintf("The provider accepted the key; it may use %d models.", models)
}

// acceptedFor is the test message for a key of provider: an Admin key reads reports, no models.
func acceptedFor(p domain.Provider, models int) string {
	if p == domain.AnthropicAdmin {
		return "Anthropic accepted the Admin key; it reads the organisation's usage and cost reports."
	}
	return accepted(models)
}

// Usage returns what the provider reports the credential used: OpenRouter's own figures for an
// OpenRouter key; Anthropic's usage and cost reports for an Anthropic key, read with the first
// readable Admin key stored (without one, nothing). A provider that fails returns an error
// wrapping domain.ErrProvider.
func (s *Service) Usage(ctx context.Context, id string) (domain.Usage, error) {
	provider, key, err := s.secret(ctx, id)
	if err != nil {
		return domain.Usage{}, err
	}
	var u domain.Usage
	switch provider {
	case domain.OpenRouter:
		u, err = s.reporter.OpenRouter(ctx, key)
	case domain.Anthropic, domain.AnthropicAdmin:
		u, err = s.anthropicUsage(ctx, provider, key)
	case domain.Mistral:
		return domain.Usage{Source: domain.UsageNone, Message: "Mistral does not report a key's usage through its API."}, nil
	}
	if err != nil {
		return domain.Usage{}, fmt.Errorf("%w: %w", domain.ErrProvider, err)
	}
	return u, nil
}

func (s *Service) anthropicUsage(ctx context.Context, provider domain.Provider, key string) (domain.Usage, error) {
	adminKey, err := s.adminKey(ctx)
	if err != nil {
		return domain.Usage{}, err
	}
	if adminKey == "" {
		return domain.Usage{
			Source: domain.UsageNone,
			Message: "Anthropic reports usage only to an Admin key. Add an Anthropic Admin key (sk-ant-admin…) to see " +
				"Anthropic's own figures for this key.",
		}, nil
	}
	if provider == domain.AnthropicAdmin {
		key = "" // the whole organisation
	}
	now := s.now().UTC()
	return s.reporter.Anthropic(ctx, adminKey, key, time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC))
}

// adminKey is the first readable Anthropic Admin key stored, or "".
func (s *Service) adminKey(ctx context.Context) (string, error) {
	stored, err := s.repo.List(ctx)
	if err != nil {
		return "", fmt.Errorf("list credentials: %w", err)
	}
	for _, c := range stored {
		if c.Provider != domain.AnthropicAdmin || !c.Readable {
			continue
		}
		if _, key, err := s.repo.Secret(ctx, c.ID); err == nil {
			return key, nil
		}
	}
	return "", nil
}

// capitalise makes a sentence of an error message.
func capitalise(s string) string {
	r := []rune(strings.TrimSuffix(s, ".") + ".")
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

// IsInputError reports whether err is the caller's to fix (422) rather than the service's.
func IsInputError(err error) bool {
	return errors.Is(err, domain.ErrInvalid) || errors.Is(err, domain.ErrUnreadable) ||
		errors.Is(err, domain.ErrVaultReadOnly) || errors.Is(err, domain.ErrKeyRefused)
}

// Haiku returns a client on an Anthropic key and the Haiku model, for background work that must
// stay cheap whatever model product matching uses: the Vault key, else a stored Anthropic key
// whose latest test did not fail. A nil Client means there is no such key.
func (s *Service) Haiku(ctx context.Context) (domain.Client, string, error) {
	creds, err := s.credentials(ctx)
	if err != nil {
		return nil, "", err
	}
	c, ok := domain.AnthropicKey(creds)
	if !ok {
		return nil, "", nil
	}
	_, key, err := s.secret(ctx, c.ID)
	if err != nil {
		return nil, "", fmt.Errorf("the Anthropic key: %w", err)
	}
	return s.connector.Connect(domain.Anthropic, key), domain.DefaultModel, nil
}
