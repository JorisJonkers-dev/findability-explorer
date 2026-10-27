package main

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/JorisJonkers-dev/findability-explorer/internal/identity/adapters/oidc/oidctest"
	"github.com/JorisJonkers-dev/findability-explorer/internal/llm/adapters/providers"
	llmapp "github.com/JorisJonkers-dev/findability-explorer/internal/llm/app"
	llmdomain "github.com/JorisJonkers-dev/findability-explorer/internal/llm/domain"
	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/pg/pgtest"
)

var sessionKeyValue = strings.Repeat("k", 32)

func env(vars map[string]string) func(string) string {
	return func(key string) string { return vars[key] }
}

const unreachable = "postgres://nobody@127.0.0.1:1/none?sslmode=disable&connect_timeout=1"

func TestStartRefusesABadEnvironment(t *testing.T) {
	cases := map[string]map[string]string{
		"no database":          {},
		"unreachable database": {"DATABASE_URL": unreachable},
	}
	for name, vars := range cases {
		t.Run(name, func(t *testing.T) {
			if code := start(t.Context(), env(vars)); code != 1 {
				t.Fatalf("start = %d, want 1", code)
			}
		})
	}
}

func TestStartWithADatabase(t *testing.T) {
	url := pgtest.URL(t)
	t.Run("invalid address", func(t *testing.T) {
		if code := start(t.Context(), env(map[string]string{"DATABASE_URL": url, "DEV_USER": "dev", "ADDR": "not-an-address"})); code != 1 {
			t.Fatalf("start with an invalid ADDR = %d, want 1", code)
		}
	})
	for _, key := range []string{"LLM_MONTHLY_TOKENS"} {
		t.Run("bad "+key, func(t *testing.T) {
			if code := start(t.Context(), env(map[string]string{"DATABASE_URL": url, "DEV_USER": "dev", key: "lots"})); code != 1 {
				t.Fatalf("start with %s=lots = %d, want 1", key, code)
			}
		})
	}
	t.Run("serves with matching enabled until cancelled", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
		defer cancel()
		vars := map[string]string{
			"DATABASE_URL": url, "ADDR": "127.0.0.1:0", "ANTHROPIC_API_KEY": "test-key", "DEV_USER": "dev",
			"DEV_ROLES": "ROLE_ADMIN", "SESSION_KEY": sessionKeyValue, "LLM_MONTHLY_TOKENS": "1000",
		}
		if code := start(ctx, env(vars)); code != 0 {
			t.Fatalf("start after a clean shutdown = %d, want 0", code)
		}
	})
	t.Run("serves until cancelled", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
		defer cancel()
		vars := map[string]string{"DATABASE_URL": url, "ADDR": "127.0.0.1:0", "DEV_USER": "dev"}
		if code := start(ctx, env(vars)); code != 0 {
			t.Fatalf("start after a clean shutdown = %d, want 0", code)
		}
	})
}

func TestStartRefusesAnIncompleteSignIn(t *testing.T) {
	url := pgtest.URL(t)
	issuer := oidctest.New(t, "findability-explorer")
	full := map[string]string{
		"DATABASE_URL": url, "ADDR": "127.0.0.1:0", "SESSION_KEY": sessionKeyValue, "OIDC_ISSUER": issuer.URL,
		"OIDC_CLIENT_ID": "findability-explorer", "OIDC_CLIENT_SECRET": "secret",
		"OIDC_REDIRECT_URL": "https://digital-strategy.test/auth/callback",
	}
	without := func(key string, replaced ...string) map[string]string {
		vars := map[string]string{}
		for k, v := range full {
			vars[k] = v
		}
		delete(vars, key)
		if len(replaced) == 1 {
			vars[key] = replaced[0]
		}
		return vars
	}
	cases := map[string]map[string]string{
		"no SESSION_KEY":      without("SESSION_KEY"),
		"short SESSION_KEY":   without("SESSION_KEY", "short"),
		"no client secret":    without("OIDC_CLIENT_SECRET"),
		"relative redirect":   without("OIDC_REDIRECT_URL", "/auth/callback"),
		"unparsable redirect": without("OIDC_REDIRECT_URL", "https://bad host/"),
		"unreachable issuer":  without("OIDC_ISSUER", "http://127.0.0.1:1"),
	}
	for name, vars := range cases {
		t.Run(name, func(t *testing.T) {
			if code := start(t.Context(), env(vars)); code != 1 {
				t.Fatalf("start = %d, want 1", code)
			}
		})
	}
	t.Run("signs in with auth-api", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
		defer cancel()
		if code := start(ctx, env(full)); code != 0 {
			t.Fatalf("start after a clean shutdown = %d, want 0", code)
		}
	})
}

func TestSessionKeyOnALocalRun(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)
	a, err := sessionKey(logger, env(map[string]string{"DEV_USER": "dev"}))
	b, _ := sessionKey(logger, env(map[string]string{"DEV_USER": "dev"}))
	if err != nil || len(a) != 64 || a == b {
		t.Fatalf("generated keys %q, %q, %v", a, b, err)
	}
	if got, err := sessionKey(logger, env(map[string]string{"DEV_USER": "dev", "SESSION_KEY": sessionKeyValue})); err != nil || got != sessionKeyValue {
		t.Fatalf("given key = %q, %v", got, err)
	}
}

// settings is an in-memory llm Repository with one decision about the active model, whose list
// of keys fails with listErr.
type settings struct {
	setting llmdomain.Setting
	listErr error
}

func (s *settings) Add(context.Context, llmdomain.Credential, string) error { return nil }
func (s *settings) List(context.Context) ([]llmdomain.Credential, error)    { return nil, s.listErr }

func (s *settings) Get(context.Context, string) (llmdomain.Credential, error) {
	return llmdomain.Credential{}, llmdomain.ErrNotFound
}

func (s *settings) Secret(context.Context, string) (llmdomain.Provider, string, error) {
	return "", "", llmdomain.ErrUnreadable
}
func (s *settings) Delete(context.Context, string) (bool, error)       { return false, nil }
func (s *settings) Setting(context.Context) (llmdomain.Setting, error) { return s.setting, nil }

func (s *settings) SaveSetting(context.Context, llmdomain.Setting, string, time.Time) error {
	return nil
}
func (s *settings) AddEvent(context.Context, llmdomain.Event) error        { return nil }
func (s *settings) Events(context.Context, int) ([]llmdomain.Event, error) { return nil, nil }
func (s *settings) LastTests(context.Context) (map[string]llmdomain.Test, error) {
	return map[string]llmdomain.Test{}, nil
}

func TestMatchersPreferHaikuUnlessMatchingIsOff(t *testing.T) {
	repo := &settings{}
	svc := func(vault string) matchers {
		return matchers{llm: llmapp.New(repo, providers.New(), nil, nil, llmapp.Config{VaultKey: vault}, time.Now, uuid.NewString)}
	}
	if m, label, err := svc("").Current(t.Context()); m != nil || label != "" || err != nil {
		t.Fatalf("without a key: %v %q %v", m, label, err)
	}
	if m, label, err := svc("sk-ant").Current(t.Context()); m == nil || label != "anthropic: claude-haiku-5-5" || err != nil {
		t.Fatalf("by default: %v %q %v", m, label, err)
	}
	// Another model chosen: Haiku on the Anthropic key still matches.
	repo.setting = llmdomain.Setting{Mode: llmdomain.ModeChosen, Active: llmdomain.Active{CredentialID: "gone", Model: "m"}}
	if m, label, err := svc("sk-ant").Current(t.Context()); m == nil || label != "anthropic: claude-haiku-5-5" || err != nil {
		t.Fatalf("with another model chosen: %v %q %v", m, label, err)
	}
	// Matching turned off stays off, Haiku or not.
	repo.setting = llmdomain.Setting{Mode: llmdomain.ModeOff}
	if m, label, err := svc("sk-ant").Current(t.Context()); m != nil || label != "" || err != nil {
		t.Fatalf("with matching off: %v %q %v", m, label, err)
	}
	// Without an Anthropic key, the chosen model matches as before.
	repo.setting = llmdomain.Setting{Mode: llmdomain.ModeChosen, Active: llmdomain.Active{CredentialID: "gone", Model: "m"}}
	if m, _, err := svc("").Current(t.Context()); m != nil || err == nil {
		t.Fatalf("an unreadable key: %v %v", m, err)
	}
	repo.listErr = errors.New("boom")
	if m, _, err := svc("sk-ant").Current(t.Context()); m != nil || err == nil {
		t.Fatalf("keys that cannot be listed: %v %v", m, err)
	}
}

func TestExtractorsAreHaikuOnAnAnthropicKey(t *testing.T) {
	repo := &settings{}
	svc := func(vault string) extractors {
		return extractors{llm: llmapp.New(repo, providers.New(), nil, nil, llmapp.Config{VaultKey: vault}, time.Now, uuid.NewString)}
	}
	if x, model, err := svc("").Current(t.Context()); x != nil || model != "" || err != nil {
		t.Fatalf("without a key: %v %q %v", x, model, err)
	}
	if x, model, err := svc("sk-ant").Current(t.Context()); x == nil || model != "anthropic: claude-haiku-5-5" || err != nil {
		t.Fatalf("with the Vault key: %v %q %v", x, model, err)
	}
}

func TestMonthStart(t *testing.T) {
	got := monthStart(time.Date(2026, 10, 23, 23, 30, 0, 0, time.FixedZone("CEST", -2*3600)))
	if !got.Equal(time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("monthStart = %v", got)
	}
}

func TestDatabaseOf(t *testing.T) {
	if got, err := databaseOf(env(map[string]string{"DATABASE_URL": "postgres://x"})); err != nil || got != "postgres://x" {
		t.Fatalf("DATABASE_URL = %q, %v", got, err)
	}
	got, err := databaseOf(env(map[string]string{
		"PGHOST": "db", "PGPORT": "5432", "PGDATABASE": "findability-explorer_db", "PGUSER": "u", "PGPASSWORD": `p'w\x`,
	}))
	if err != nil || got != `host='db' port='5432' dbname='findability-explorer_db' user='u' password='p\'w\\x'` {
		t.Fatalf("libpq = %q, %v", got, err)
	}
	if _, err := databaseOf(env(nil)); err == nil {
		t.Fatal("no database settings were accepted")
	}
}
