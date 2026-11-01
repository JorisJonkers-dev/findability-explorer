package main

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	activitypersistence "github.com/JorisJonkers-dev/findability-explorer/internal/activity/adapters/persistence"
	activitydomain "github.com/JorisJonkers-dev/findability-explorer/internal/activity/domain"
	"github.com/JorisJonkers-dev/findability-explorer/internal/analysis/adapters/persistence"
	claudestepapp "github.com/JorisJonkers-dev/findability-explorer/internal/claudestep/app"
	claudestepdomain "github.com/JorisJonkers-dev/findability-explorer/internal/claudestep/domain"
	"github.com/JorisJonkers-dev/findability-explorer/internal/identity/adapters/oidc/oidctest"
	insightpersistence "github.com/JorisJonkers-dev/findability-explorer/internal/insight/adapters/persistence"
	"github.com/JorisJonkers-dev/findability-explorer/internal/llm/adapters/providers"
	llmapp "github.com/JorisJonkers-dev/findability-explorer/internal/llm/app"
	llmdomain "github.com/JorisJonkers-dev/findability-explorer/internal/llm/domain"
	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/pg"
	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/pg/pgtest"
	prompttestpersistence "github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/adapters/persistence"
	prompttestdomain "github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain"
	terminologypersistence "github.com/JorisJonkers-dev/findability-explorer/internal/terminology/adapters/persistence"
	terminologydomain "github.com/JorisJonkers-dev/findability-explorer/internal/terminology/domain"
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
	t.Run("serves with matching enabled until cancelled", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
		defer cancel()
		vars := map[string]string{
			"DATABASE_URL": url, "ADDR": "127.0.0.1:0", "ANTHROPIC_API_KEY": "test-key", "DEV_USER": "dev",
			"DEV_ROLES": "ROLE_ADMIN", "SESSION_KEY": sessionKeyValue,
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
func (s *settings) Budget(context.Context) (llmdomain.Micros, error) { return 0, nil }
func (s *settings) SaveBudget(context.Context, llmdomain.Micros, string, time.Time) error {
	return nil
}

func TestTheMonthsSpendingPricesEveryJobAtTheModelThatDidIt(t *testing.T) {
	store, err := pg.Open(t.Context(), pgtest.URL(t))
	if err != nil {
		t.Fatal(err)
	}
	ctx := t.Context()
	since := monthStart(time.Now())
	sp := spenders{persistence.New(store), terminologypersistence.New(store), prompttestpersistence.New(store), insightpersistence.New(store)}
	s, err := spending(ctx, sp, since)
	if err != nil || len(s.ThisMonth) != 0 || s.ScansMatched != 0 || s.AveragePerScan.Tokens() != 0 {
		t.Fatalf("spending of an empty month = %+v, %v", s, err)
	}
	// The term review and the candidate review on Claude Fable 5.1, and one review from before
	// reviews recorded their model, which counts as Claude Haiku 5.5.
	const fable = "claude-fable-5-1"
	at := time.Now().UTC()
	if err := sp.terms.SaveReview(ctx, nil, nil, []terminologydomain.Usage{{InputTokens: 100_000, OutputTokens: 10_000, At: at, Model: fable}, {InputTokens: 50_000, At: at}}); err != nil {
		t.Fatal(err)
	}
	if err := sp.candidates.SaveReviewUsage(ctx, []prompttestdomain.ReviewUsage{{InputTokens: 200_000, OutputTokens: 20_000, At: at, Model: fable}}); err != nil {
		t.Fatal(err)
	}
	fablePrice := func(in, out int64) llmdomain.Micros { return llmdomain.ListPrice(fable).Of(in, out) }
	want := fablePrice(100_000, 10_000) + fablePrice(200_000, 20_000) + llmdomain.ListPrice(llmdomain.DefaultModel).Of(50_000, 0)
	s, err = spending(ctx, sp, since)
	if err != nil {
		t.Fatal(err)
	}
	var cost llmdomain.Micros
	for _, u := range s.ThisMonth {
		cost += u.Cost()
	}
	if cost != want {
		t.Errorf("the budget's spending = %s, want %s at Fable's price", cost, want)
	}
	// Activity prices the same work at the same models.
	usage, err := activitypersistence.New(store).MonthUsage(ctx, since)
	if err != nil {
		t.Fatal(err)
	}
	var activity llmdomain.Micros
	for _, l := range activitydomain.SpendingOf(usage) {
		if l.Key == activitydomain.KeyAnthropic {
			activity += l.Cost
		}
	}
	if activity != want {
		t.Errorf("Activity's spending = %s, want %s", activity, want)
	}
	store.Close()
	if _, err := spending(ctx, sp, since); err == nil {
		t.Fatal("spending without a database did not fail")
	}
}

// stepVersions is the Claude steps' store in memory.
type stepVersions struct {
	saved []claudestepdomain.Version
	err   error
}

func (s *stepVersions) Versions(context.Context, claudestepdomain.ID) ([]claudestepdomain.Version, error) {
	return s.saved, s.err
}

func (s *stepVersions) Save(_ context.Context, v claudestepdomain.Version, original *claudestepdomain.Version) error {
	if original != nil {
		s.saved = append(s.saved, *original)
	}
	s.saved = append(s.saved, v)
	return nil
}

func TestMatchersPreferHaikuUnlessMatchingIsOff(t *testing.T) {
	repo := &settings{}
	versions := &stepVersions{}
	steps := claudestepapp.New(versions, func(context.Context, claudestepdomain.ID) (llmdomain.Micros, error) { return 0, nil }, time.Now)
	svc := func(vault string) matchers {
		return matchers{llm: llmapp.New(repo, providers.New(), nil, nil, llmapp.Config{VaultKey: vault}, time.Now, uuid.NewString), steps: steps}
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
	// The step matching products names another Claude model: the Anthropic key asks that one.
	if _, err := steps.Save(t.Context(), claudestepdomain.Matching, "Joris", claudestepdomain.Change{Model: "claude-sonnet-5-5", Instructions: "Match."}); err != nil {
		t.Fatal(err)
	}
	if m, label, err := svc("sk-ant").Current(t.Context()); m == nil || label != "anthropic: claude-sonnet-5-5" || err != nil {
		t.Fatalf("with the step on Sonnet: %v %q %v", m, label, err)
	}
	versions.err = errors.New("database down")
	if m, _, err := svc("sk-ant").Current(t.Context()); m != nil || err == nil {
		t.Fatalf("a step that cannot be read: %v %v", m, err)
	}
	versions.err = nil
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

// stepsOn returns the Claude steps over versions in memory.
func stepsOn(versions *stepVersions) *claudestepapp.Service {
	return claudestepapp.New(versions, func(context.Context, claudestepdomain.ID) (llmdomain.Micros, error) { return 0, nil }, time.Now)
}

func TestTheClaudeJobsOnAnAnthropicKeyUseTheirStepsNewestVersions(t *testing.T) {
	repo := &settings{}
	versions := &stepVersions{}
	steps := stepsOn(versions)
	llm := func(vault string) *llmapp.Service {
		return llmapp.New(repo, providers.New(), nil, nil, llmapp.Config{VaultKey: vault}, time.Now, uuid.NewString)
	}
	type job func(vault string) (bool, string, error)
	jobs := map[string]job{
		"reading": func(v string) (bool, string, error) {
			x, label, err := extractors{llm: llm(v), steps: steps}.Current(t.Context())
			return x != nil, label, err
		},
		"reviewing terms": func(v string) (bool, string, error) {
			j, label, err := judges{llm: llm(v), steps: steps}.Current(t.Context())
			return j != nil, label, err
		},
		"checking companies": func(v string) (bool, string, error) {
			r, label, err := reviewers{llm: llm(v), steps: steps}.Current(t.Context())
			return r != nil, label, err
		},
	}
	for name, run := range jobs {
		if ok, label, err := run(""); ok || label != "" || err != nil {
			t.Errorf("%s without a key: %v %q %v", name, ok, label, err)
		}
		if ok, label, err := run("sk-ant"); !ok || label != "anthropic: claude-haiku-5-5" || err != nil {
			t.Errorf("%s by default: %v %q %v", name, ok, label, err)
		}
	}
	// Each step on another model: its job asks that one. The versions here are one list, so every
	// step reads the newest of it.
	if _, err := steps.Save(t.Context(), claudestepdomain.ReviewingTerms, "Joris", claudestepdomain.Change{Model: "claude-sonnet-5-5", Instructions: "Review."}); err != nil {
		t.Fatal(err)
	}
	for name, run := range jobs {
		if ok, label, err := run("sk-ant"); !ok || label != "anthropic: claude-sonnet-5-5" || err != nil {
			t.Errorf("%s on Sonnet: %v %q %v", name, ok, label, err)
		}
	}
	versions.err = errors.New("database down")
	for name, run := range jobs {
		if ok, _, err := run("sk-ant"); ok || err == nil {
			t.Errorf("%s with steps that cannot be read: %v %v", name, ok, err)
		}
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

func TestAScansMatchingIsEstimatedFromTheAverageScan(t *testing.T) {
	spent := func(context.Context) (llmapp.Spending, error) {
		return llmapp.Spending{AveragePerScan: llmdomain.TokenUse{Model: llmdomain.DefaultModel, Input: 10_000, Output: 1_000}, ScansMatched: 2}, nil
	}
	b := stepBase{llm: llmapp.New(&settings{}, providers.New(), nil, spent, llmapp.Config{}, time.Now, uuid.NewString)}
	if got, err := b.Of(t.Context(), claudestepdomain.Matching); err != nil || got != 1_500 {
		t.Errorf("matching = %d %v", got, err)
	}
	if got, err := b.Of(t.Context(), "other"); err != nil || got != 0 {
		t.Errorf("another step = %d %v", got, err)
	}
	broken := stepBase{llm: llmapp.New(&settings{listErr: errors.New("boom")}, providers.New(), nil, spent, llmapp.Config{}, time.Now, uuid.NewString)}
	if _, err := broken.Of(t.Context(), claudestepdomain.Matching); err == nil {
		t.Error("a failure was not reported")
	}
}
