package app_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/JorisJonkers-dev/findability-explorer/internal/llm/app"
	"github.com/JorisJonkers-dev/findability-explorer/internal/llm/domain"
)

var (
	errBoom = errors.New("boom")
	now     = time.Date(2026, 10, 23, 9, 0, 0, 0, time.UTC)
)

// repo is an in-memory Repository; failOn names the methods that fail, unreadable the ids whose
// key was sealed under another SESSION_KEY.
type repo struct {
	creds      []domain.Credential
	keys       map[string]string
	setting    domain.Setting
	events     []domain.Event
	tests      map[string]domain.Test
	failOn     map[string]bool
	unreadable map[string]bool
}

func newRepo() *repo {
	return &repo{keys: map[string]string{}, tests: map[string]domain.Test{}, failOn: map[string]bool{}, unreadable: map[string]bool{}}
}

func (r *repo) fail(op string) error {
	if r.failOn[op] {
		return errBoom
	}
	return nil
}

func (r *repo) Add(_ context.Context, c domain.Credential, key string) error {
	if err := r.fail("Add"); err != nil {
		return err
	}
	c.LastTest = nil
	r.creds = append(r.creds, c)
	r.keys[c.ID] = key
	return nil
}

func (r *repo) List(context.Context) ([]domain.Credential, error) {
	out := make([]domain.Credential, 0, len(r.creds))
	for _, c := range r.creds {
		c.Readable = !r.unreadable[c.ID]
		out = append(out, c)
	}
	return out, r.fail("List")
}

func (r *repo) Get(_ context.Context, id string) (domain.Credential, error) {
	for _, c := range r.creds {
		if c.ID == id {
			return c, r.fail("Get")
		}
	}
	return domain.Credential{}, domain.ErrNotFound
}

func (r *repo) Secret(_ context.Context, id string) (domain.Provider, string, error) {
	if err := r.fail("Secret"); err != nil {
		return "", "", err
	}
	for _, c := range r.creds {
		if c.ID == id {
			if r.unreadable[id] {
				return "", "", domain.ErrUnreadable
			}
			return c.Provider, r.keys[id], nil
		}
	}
	return "", "", domain.ErrNotFound
}

func (r *repo) Delete(_ context.Context, id string) (bool, error) {
	if err := r.fail("Delete"); err != nil {
		return false, err
	}
	for i, c := range r.creds {
		if c.ID == id {
			r.creds = append(r.creds[:i], r.creds[i+1:]...)
			was := r.setting.Mode == domain.ModeChosen && r.setting.Active.CredentialID == id
			if was {
				r.setting = domain.Setting{}
			}
			return was, nil
		}
	}
	return false, domain.ErrNotFound
}

func (r *repo) Setting(context.Context) (domain.Setting, error) { return r.setting, r.fail("Setting") }

func (r *repo) SaveSetting(_ context.Context, s domain.Setting, _ string, _ time.Time) error {
	if err := r.fail("SaveSetting"); err != nil {
		return err
	}
	r.setting = s
	return nil
}

func (r *repo) AddEvent(_ context.Context, e domain.Event) error {
	if err := r.fail("AddEvent"); err != nil {
		return err
	}
	r.events = append([]domain.Event{e}, r.events...)
	if e.OK != nil && e.CredentialID != "" {
		r.tests[e.CredentialID] = domain.Test{OK: *e.OK, At: e.At, Message: e.Message}
	}
	return nil
}

func (r *repo) Events(_ context.Context, limit int) ([]domain.Event, error) {
	return r.events[:min(limit, len(r.events))], r.fail("Events")
}

func (r *repo) LastTests(context.Context) (map[string]domain.Test, error) {
	return r.tests, r.fail("LastTests")
}

// connector answers per key: a key starting with "bad" is refused, "down" fails to list models.
type connector struct{ connected []string }

func (c *connector) Connect(p domain.Provider, key string) domain.Client {
	c.connected = append(c.connected, string(p)+":"+key)
	return client{key: key}
}

type client struct{ key string }

func (c client) Models(context.Context) ([]domain.Model, error) {
	switch {
	case strings.HasPrefix(c.key, "bad"):
		return nil, errors.New("mistral refused the API key")
	case strings.HasPrefix(c.key, "down"):
		return nil, errors.New("mistral answered 500")
	}
	return []domain.Model{{ID: "z-model", Name: "Z"}, {ID: "a-model", Name: "A"}}, nil
}

func (client) Complete(context.Context, domain.Prompt) (domain.Completion, error) {
	return domain.Completion{}, nil
}

type fixture struct {
	svc      *app.Service
	repo     *repo
	conn     *connector
	reporter *reporter
}

// reporter reports what it is asked, or fails with err.
type reporter struct {
	err   error
	asked []string
}

func (r *reporter) OpenRouter(_ context.Context, key string) (domain.Usage, error) {
	r.asked = append(r.asked, "openrouter:"+key)
	total := 1.25
	return domain.Usage{Source: domain.UsageOpenRouter, Total: &total}, r.err
}

func (r *reporter) Anthropic(_ context.Context, adminKey, key string, since time.Time) (domain.Usage, error) {
	r.asked = append(r.asked, "anthropic:"+adminKey+":"+key+":"+since.Format("2006-01-02"))
	return domain.Usage{Source: domain.UsageAnthropic, Since: since}, r.err
}

func setup(vault string, spentErr error) fixture {
	r := newRepo()
	conn := &connector{}
	rep := &reporter{}
	n := 0
	svc := app.New(r, conn, rep, func(context.Context) (app.Spending, error) {
		return app.Spending{ThisMonth: 1234, AveragePerScan: 617, ScansMatched: 2}, spentErr
	},
		app.Config{VaultKey: vault, MonthlyTokens: 2_000_000},
		func() time.Time { return now },
		func() string { n++; return fmt.Sprintf("00000000-0000-4000-8000-%012d", n) })
	return fixture{svc: svc, repo: r, conn: conn, reporter: rep}
}

func TestSettingsByDefault(t *testing.T) {
	f := setup("sk-ant-vault", nil)
	s, err := f.svc.Settings(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Credentials) != 1 || s.Credentials[0].ID != domain.VaultID || s.Choice != domain.ChoiceDefault ||
		s.Active == nil || s.Active.Model != domain.DefaultModel || !s.MatchingEnabled ||
		s.Haiku == nil || *s.Haiku != (domain.Active{CredentialID: domain.VaultID, Model: domain.DefaultModel}) ||
		s.Budget != (app.Budget{MonthlyTokens: 2_000_000, UsedThisMonth: 1234, AveragePerScan: 617, ScansMatched: 2}) {
		t.Fatalf("settings = %+v", s)
	}
	f = setup("", nil)
	if s, _ := f.svc.Settings(t.Context()); len(s.Credentials) != 0 || s.Active != nil || s.Haiku != nil || s.MatchingEnabled {
		t.Fatalf("settings without Vault = %+v", s)
	}
}

func TestSettingsFailures(t *testing.T) {
	for _, op := range []string{"List", "LastTests", "Setting", "Events"} {
		f := setup("v-key-123", nil)
		f.repo.failOn[op] = true
		if _, err := f.svc.Settings(t.Context()); !errors.Is(err, errBoom) {
			t.Errorf("%s failing: %v", op, err)
		}
	}
	if _, err := setup("", errBoom).svc.Settings(t.Context()); !errors.Is(err, errBoom) {
		t.Errorf("spent failing: %v", err)
	}
}

func TestBudgetSpentAtTheMonthlyTokens(t *testing.T) {
	f := setup("", nil)
	if spent, err := f.svc.BudgetSpent(t.Context()); err != nil || spent {
		t.Fatalf("1,234 of 2,000,000 = %v, %v", spent, err)
	}
	full := app.New(newRepo(), &connector{}, &reporter{}, func(context.Context) (app.Spending, error) {
		return app.Spending{ThisMonth: 2_000_000}, nil
	}, app.Config{MonthlyTokens: 2_000_000}, func() time.Time { return now }, func() string { return "id" })
	if spent, err := full.BudgetSpent(t.Context()); err != nil || !spent {
		t.Fatalf("2,000,000 of 2,000,000 = %v, %v", spent, err)
	}
	if _, err := setup("", errBoom).svc.BudgetSpent(t.Context()); !errors.Is(err, errBoom) {
		t.Fatalf("spent failing = %v", err)
	}
}

func TestAddChooseAndDelete(t *testing.T) {
	f := setup("sk-ant-vault", nil)
	c, err := f.svc.Add(t.Context(), "joris", "mistral", " Course ", "good-key-1")
	if err != nil {
		t.Fatal(err)
	}
	if c.ID == "" || c.Label != "Course" || c.Fingerprint != domain.Fingerprint("good-key-1") || c.CreatedBy != "joris" ||
		c.LastTest == nil || !c.LastTest.OK || f.repo.keys[c.ID] != "good-key-1" {
		t.Fatalf("added %+v", c)
	}
	if e := f.repo.events[0]; e.Action != domain.ActionCredentialAdded || !*e.OK || !strings.Contains(e.Message, "2 models") || strings.Contains(e.Message, "good-key-1") {
		t.Fatalf("event %+v", e)
	}
	s, err := f.svc.SetActive(t.Context(), "joris", c.ID, "mistral-small-latest")
	if err != nil || s.Choice != domain.ChoiceChosen || s.Active.CredentialID != c.ID || !s.MatchingEnabled || len(s.Credentials) != 2 ||
		s.Credentials[1].LastTest == nil {
		t.Fatalf("set active = %+v, %v", s, err)
	}
	client, provider, model, err := f.svc.Current(t.Context())
	if err != nil || client == nil || provider != domain.Mistral || model != "mistral-small-latest" {
		t.Fatalf("current = %v %s %s, %v", client, provider, model, err)
	}
	if err := f.svc.Delete(t.Context(), "joris", c.ID); err != nil {
		t.Fatal(err)
	}
	if e := f.repo.events[0]; e.Action != domain.ActionCredentialDeleted || !strings.Contains(e.Message, "falls back") {
		t.Fatalf("delete event %+v", e)
	}
	if s, _ := f.svc.Settings(t.Context()); s.Choice != domain.ChoiceDefault || s.Active.CredentialID != domain.VaultID {
		t.Fatalf("after deleting the active key: %+v", s)
	}
	c2, _ := f.svc.Add(t.Context(), "joris", "openrouter", "Spare", "good-key-2")
	if err := f.svc.Delete(t.Context(), "joris", c2.ID); err != nil || strings.Contains(f.repo.events[0].Message, "falls back") {
		t.Fatalf("delete an inactive key: %v, %q", err, f.repo.events[0].Message)
	}
}

func TestAddRefusals(t *testing.T) {
	f := setup("", nil)
	if _, err := f.svc.Add(t.Context(), "j", "openai", "x", "good-key-1"); !errors.Is(err, domain.ErrInvalidProvider) || !app.IsInputError(err) {
		t.Fatalf("bad provider: %v", err)
	}
	_, err := f.svc.Add(t.Context(), "j", "mistral", "x", "bad-key-12")
	if !errors.Is(err, domain.ErrKeyRefused) || !strings.Contains(err.Error(), "refused the API key") || !app.IsInputError(err) {
		t.Fatalf("refused key: %v", err)
	}
	if len(f.repo.creds) != 0 || len(f.repo.events) != 0 {
		t.Fatal("a refused key was stored or logged")
	}
	f.repo.failOn["Add"] = true
	if _, err := f.svc.Add(t.Context(), "j", "mistral", "x", "good-key-1"); !errors.Is(err, errBoom) || app.IsInputError(err) {
		t.Fatalf("store fails: %v", err)
	}
	f.repo.failOn = map[string]bool{"AddEvent": true}
	if _, err := f.svc.Add(t.Context(), "j", "mistral", "x", "good-key-1"); !errors.Is(err, errBoom) {
		t.Fatalf("event fails: %v", err)
	}
}

func TestDeleteRefusals(t *testing.T) {
	f := setup("sk-ant-vault", nil)
	if err := f.svc.Delete(t.Context(), "j", domain.VaultID); !errors.Is(err, domain.ErrVaultReadOnly) {
		t.Fatalf("vault: %v", err)
	}
	if err := f.svc.Delete(t.Context(), "j", "nobody"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("unknown: %v", err)
	}
	c, _ := f.svc.Add(t.Context(), "j", "mistral", "x", "good-key-1")
	f.repo.failOn["Delete"] = true
	if err := f.svc.Delete(t.Context(), "j", c.ID); !errors.Is(err, errBoom) {
		t.Fatalf("store fails: %v", err)
	}
}

func TestTest(t *testing.T) {
	f := setup("sk-ant-vault", nil)
	test, err := f.svc.Test(t.Context(), "j", domain.VaultID)
	if err != nil || !test.OK || !strings.Contains(test.Message, "2 models") || f.conn.connected[0] != "anthropic:sk-ant-vault" {
		t.Fatalf("test vault = %+v, %v (%v)", test, err, f.conn.connected)
	}
	if s, _ := f.svc.Settings(t.Context()); s.Credentials[0].LastTest == nil || !s.Credentials[0].LastTest.OK {
		t.Fatalf("the test was not recorded: %+v", s.Credentials[0])
	}
	f.repo.creds = append(f.repo.creds, domain.Credential{ID: "c-bad", Provider: domain.Mistral})
	f.repo.keys["c-bad"] = "bad-key-12"
	test, err = f.svc.Test(t.Context(), "j", "c-bad")
	if err != nil || test.OK || test.Message != "Mistral refused the API key." || *f.repo.events[0].OK {
		t.Fatalf("test refused key = %+v, %v", test, err)
	}
	if _, err := f.svc.Test(t.Context(), "j", "nobody"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("unknown: %v", err)
	}
	f.repo.unreadable["c-bad"] = true
	if _, err := f.svc.Test(t.Context(), "j", "c-bad"); !errors.Is(err, domain.ErrUnreadable) || !app.IsInputError(err) {
		t.Fatalf("unreadable: %v", err)
	}
	f.repo.failOn["AddEvent"] = true
	if _, err := f.svc.Test(t.Context(), "j", domain.VaultID); !errors.Is(err, errBoom) {
		t.Fatalf("event fails: %v", err)
	}
	if _, err := setup("", nil).svc.Test(t.Context(), "j", domain.VaultID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("vault without a key: %v", err)
	}
}

func TestModels(t *testing.T) {
	f := setup("sk-ant-vault", nil)
	models, err := f.svc.Models(t.Context(), domain.VaultID)
	if err != nil || len(models) != 2 || models[0].ID != "a-model" {
		t.Fatalf("models = %+v, %v", models, err)
	}
	f.repo.creds = append(f.repo.creds, domain.Credential{ID: "c-down", Provider: domain.Mistral})
	f.repo.keys["c-down"] = "down-key-1"
	if _, err := f.svc.Models(t.Context(), "c-down"); !errors.Is(err, domain.ErrProvider) || app.IsInputError(err) {
		t.Fatalf("provider down: %v", err)
	}
	if _, err := f.svc.Models(t.Context(), "nobody"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("unknown: %v", err)
	}
}

func TestSetActiveRefusals(t *testing.T) {
	f := setup("sk-ant-vault", nil)
	if _, err := f.svc.SetActive(t.Context(), "j", domain.VaultID, "a model"); !errors.Is(err, domain.ErrInvalidModel) {
		t.Fatalf("bad model: %v", err)
	}
	if _, err := f.svc.SetActive(t.Context(), "j", "nobody", "m"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("unknown credential: %v", err)
	}
	f.repo.failOn["SaveSetting"] = true
	if _, err := f.svc.SetActive(t.Context(), "j", domain.VaultID, "m"); !errors.Is(err, errBoom) {
		t.Fatalf("store fails: %v", err)
	}
	if _, err := f.svc.ClearActive(t.Context(), "j"); !errors.Is(err, errBoom) {
		t.Fatalf("store fails on clear: %v", err)
	}
	f.repo.failOn = map[string]bool{"AddEvent": true}
	if _, err := f.svc.SetActive(t.Context(), "j", domain.VaultID, "m"); !errors.Is(err, errBoom) {
		t.Fatalf("event fails: %v", err)
	}
	if _, err := f.svc.ClearActive(t.Context(), "j"); !errors.Is(err, errBoom) {
		t.Fatalf("event fails on clear: %v", err)
	}
}

func TestClearActiveAndCurrent(t *testing.T) {
	f := setup("sk-ant-vault", nil)
	client, provider, model, err := f.svc.Current(t.Context())
	if err != nil || client == nil || provider != domain.Anthropic || model != domain.DefaultModel {
		t.Fatalf("default current = %v %s %s, %v", client, provider, model, err)
	}
	s, err := f.svc.ClearActive(t.Context(), "j")
	// Turned off, matching stays off, though Haiku on the Vault key still reads Prompt answers.
	if err != nil || s.Choice != domain.ChoiceOff || s.Active != nil || s.Haiku == nil || s.MatchingEnabled || f.repo.events[0].Action != domain.ActionActiveCleared {
		t.Fatalf("clear = %+v, %v", s, err)
	}
	if client, _, _, err := f.svc.Current(t.Context()); client != nil || err != nil {
		t.Fatalf("current when off = %v, %v", client, err)
	}
	f = setup("", nil) // no Anthropic key, so no Haiku
	f.repo.creds = append(f.repo.creds, domain.Credential{ID: "c-gone", Provider: domain.Mistral})
	f.repo.setting = domain.Setting{Mode: domain.ModeChosen, Active: domain.Active{CredentialID: "c-gone", Model: "m"}}
	f.repo.unreadable["c-gone"] = true
	if s, _ := f.svc.Settings(t.Context()); s.MatchingEnabled || s.Haiku != nil {
		t.Fatal("matching is enabled with an unreadable key")
	}
	if _, _, _, err := f.svc.Current(t.Context()); !errors.Is(err, domain.ErrUnreadable) {
		t.Fatalf("current with an unreadable key: %v", err)
	}
	f.repo.failOn["Setting"] = true
	if _, _, _, err := f.svc.Current(t.Context()); !errors.Is(err, errBoom) {
		t.Fatalf("current with the store down: %v", err)
	}
}

func TestMatcherPrefersHaikuUnlessMatchingIsOff(t *testing.T) {
	f := setup("sk-ant-vault", nil)
	f.repo.setting = domain.Setting{Mode: domain.ModeChosen, Active: domain.Active{CredentialID: "c-1", Model: "m"}}
	client, provider, model, err := f.svc.Matcher(t.Context())
	if err != nil || client == nil || provider != domain.Anthropic || model != domain.DefaultModel {
		t.Fatalf("with an Anthropic key = %v %s %s, %v", client, provider, model, err)
	}
	f.repo.setting = domain.Setting{Mode: domain.ModeOff}
	if client, provider, model, err := f.svc.Matcher(t.Context()); client != nil || provider != "" || model != "" || err != nil {
		t.Fatalf("turned off = %v %s %s, %v", client, provider, model, err)
	}

	without := setup("", nil)
	if _, err := without.svc.Add(t.Context(), "joris", "mistral", "Matching", "mk-stored"); err != nil {
		t.Fatal(err)
	}
	without.repo.setting = domain.Setting{Mode: domain.ModeChosen, Active: domain.Active{CredentialID: without.repo.creds[0].ID, Model: "mistral-small"}}
	if client, provider, model, err := without.svc.Matcher(t.Context()); client == nil || provider != domain.Mistral || model != "mistral-small" || err != nil {
		t.Fatalf("without an Anthropic key = %v %s %s, %v", client, provider, model, err)
	}
	without.repo.failOn["List"] = true
	if _, _, _, err := without.svc.Matcher(t.Context()); !errors.Is(err, errBoom) {
		t.Fatalf("keys that cannot be listed = %v", err)
	}
	without.repo.failOn["Setting"] = true
	if _, _, _, err := without.svc.Matcher(t.Context()); !errors.Is(err, errBoom) {
		t.Fatalf("the store down = %v", err)
	}
}

func TestOpenRouterKeyHandsOutOnlyAStoredOpenRouterKey(t *testing.T) {
	f := setup("sk-ant-vault", nil)
	c, err := f.svc.Add(t.Context(), "joris", "openrouter", "Prompt runs", "sk-or-v1-key")
	if err != nil {
		t.Fatal(err)
	}
	if key, err := f.svc.OpenRouterKey(t.Context(), c.ID); err != nil || key != "sk-or-v1-key" {
		t.Errorf("key = %q, %v", key, err)
	}
	if _, err := f.svc.OpenRouterKey(t.Context(), domain.VaultID); !errors.Is(err, app.ErrNotOpenRouter) {
		t.Errorf("the Vault key = %v", err)
	}
	if _, err := f.svc.OpenRouterKey(t.Context(), "00000000-0000-4000-8000-000000000099"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("no key = %v", err)
	}
}

func TestUsageAsksTheProviderWhatTheKeyUsed(t *testing.T) {
	f := setup("sk-ant-vault", nil)
	or, _ := f.svc.Add(t.Context(), "joris", "openrouter", "Runs", "or-key-123")
	u, err := f.svc.Usage(t.Context(), or.ID)
	if err != nil || u.Source != domain.UsageOpenRouter || *u.Total != 1.25 {
		t.Fatalf("openrouter usage = %+v, %v", u, err)
	}
	// Without an Admin key Anthropic reports nothing.
	u, err = f.svc.Usage(t.Context(), domain.VaultID)
	if err != nil || u.Source != domain.UsageNone || !strings.Contains(u.Message, "Admin key") {
		t.Fatalf("anthropic usage without an Admin key = %+v, %v", u, err)
	}
	admin, err := f.svc.Add(t.Context(), "joris", "anthropic_admin", "Reports", "sk-ant-admin-1")
	if err != nil || !strings.Contains(admin.LastTest.Message, "usage and cost reports") {
		t.Fatalf("admin key = %+v, %v", admin, err)
	}
	if _, err := f.svc.Usage(t.Context(), domain.VaultID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Usage(t.Context(), admin.ID); err != nil {
		t.Fatal(err)
	}
	want := []string{"openrouter:or-key-123", "anthropic:sk-ant-admin-1:sk-ant-vault:2026-10-01", "anthropic:sk-ant-admin-1::2026-10-01"}
	if !slices.Equal(f.reporter.asked, want) {
		t.Errorf("asked %v", f.reporter.asked)
	}
	mistral, _ := f.svc.Add(t.Context(), "joris", "mistral", "M", "mistral-key-1")
	if u, err := f.svc.Usage(t.Context(), mistral.ID); err != nil || u.Source != domain.UsageNone {
		t.Errorf("mistral usage = %+v, %v", u, err)
	}
	if _, err := f.svc.SetActive(t.Context(), "joris", admin.ID, "claude-haiku-5-5"); !errors.Is(err, domain.ErrReportsOnly) || !app.IsInputError(err) {
		t.Errorf("an Admin key chosen for matching = %v", err)
	}
	test, err := f.svc.Test(t.Context(), "joris", admin.ID)
	if err != nil || !strings.Contains(test.Message, "usage and cost reports") {
		t.Errorf("admin test = %+v, %v", test, err)
	}
}

func TestUsageFailures(t *testing.T) {
	f := setup("sk-ant-vault", nil)
	if _, err := f.svc.Usage(t.Context(), "00000000-0000-4000-8000-000000000099"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("no key = %v", err)
	}
	or, _ := f.svc.Add(t.Context(), "joris", "openrouter", "Runs", "or-key-123")
	f.reporter.err = errBoom
	if _, err := f.svc.Usage(t.Context(), or.ID); !errors.Is(err, domain.ErrProvider) {
		t.Errorf("a failing report = %v", err)
	}
	admin, _ := f.svc.Add(t.Context(), "joris", "anthropic_admin", "Reports", "sk-ant-admin-1")
	f.repo.unreadable[admin.ID] = true
	f.reporter.err = nil
	if u, err := f.svc.Usage(t.Context(), domain.VaultID); err != nil || u.Source != domain.UsageNone {
		t.Errorf("an unreadable Admin key is no Admin key: %+v, %v", u, err)
	}
	f.repo.failOn["List"] = true
	if _, err := f.svc.Usage(t.Context(), domain.VaultID); !errors.Is(err, errBoom) {
		t.Errorf("listing failing = %v", err)
	}
}

func TestHaikuRunsOnAnAnthropicKeyWhateverMatchingUses(t *testing.T) {
	f := setup("sk-ant-vault", nil)
	client, model, err := f.svc.Haiku(t.Context())
	if err != nil || client == nil || model != domain.DefaultModel || f.conn.connected[len(f.conn.connected)-1] != "anthropic:sk-ant-vault" {
		t.Fatalf("with the Vault key = %v, %q, %v; connected %v", client, model, err, f.conn.connected)
	}

	stored := setup("", nil)
	if client, _, err := stored.svc.Haiku(t.Context()); client != nil || err != nil {
		t.Errorf("without an Anthropic key = %v, %v", client, err)
	}
	if _, err := stored.svc.Add(t.Context(), "joris", "openrouter", "Prompt runs", "sk-or-v1-key"); err != nil {
		t.Fatal(err)
	}
	if client, _, _ := stored.svc.Haiku(t.Context()); client != nil {
		t.Error("an OpenRouter key was used")
	}
	if _, err := stored.svc.Add(t.Context(), "joris", "anthropic", "Course", "sk-ant-stored"); err != nil {
		t.Fatal(err)
	}
	if client, model, err := stored.svc.Haiku(t.Context()); client == nil || err != nil || model != domain.DefaultModel ||
		stored.conn.connected[len(stored.conn.connected)-1] != "anthropic:sk-ant-stored" {
		t.Errorf("with a stored key = %v, %q, %v; connected %v", client, model, err, stored.conn.connected)
	}

	stored.repo.failOn["Secret"] = true
	if _, _, err := stored.svc.Haiku(t.Context()); !errors.Is(err, errBoom) {
		t.Errorf("an unreadable key = %v", err)
	}
	stored.repo.failOn["List"] = true
	if _, _, err := stored.svc.Haiku(t.Context()); !errors.Is(err, errBoom) {
		t.Errorf("failing to list keys = %v", err)
	}
}
