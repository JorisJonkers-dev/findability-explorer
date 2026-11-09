package httpapi_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	identity "github.com/JorisJonkers-dev/findability-explorer/internal/identity/domain"
	llmapp "github.com/JorisJonkers-dev/findability-explorer/internal/llm/app"
	llm "github.com/JorisJonkers-dev/findability-explorer/internal/llm/domain"
)

// auth resolves the cookie values the tests use: "tester" is an account, "admin" an administrator,
// "unconfirmed" one whose roles auth-api cannot confirm, "broken" a database failure; anything
// else has no session.
type auth struct{}

func (auth) Authenticate(_ context.Context, cookie string, need identity.Need) (identity.Principal, error) {
	switch cookie {
	case "tester":
		return need.Authorise(identity.Principal{Subject: "tester", Name: "Tester", Email: "t@example.com"})
	case "admin":
		return need.Authorise(identity.Principal{Subject: "admin-1", Name: "Admin", Roles: []string{identity.AdminRole}})
	case "unconfirmed":
		return identity.Principal{}, identity.ErrRolesUnconfirmed
	case "broken":
		return identity.Principal{}, errors.New("database down")
	default:
		return identity.Principal{}, identity.ErrNoSession
	}
}

// llmCases is a stand-in for the language-model use cases; err fails every call.
type llmCases struct {
	usageErr error
	err      error
	actor    string
}

var at = time.Date(2026, 10, 23, 9, 0, 0, 0, time.UTC)

func (l *llmCases) settings() llmapp.Settings {
	ok := true
	test := llm.Test{OK: true, At: at, Message: "The provider accepted the key; it may use 3 models."}
	return llmapp.Settings{
		Credentials: []llm.Credential{
			llm.VaultCredential("sk-ant-test-key"),
			{
				ID: "5b0c1a9e-2d3f-4e5a-8b6c-7d8e9f0a1b2c", Provider: llm.OpenRouter, Label: "Course", Source: llm.SourceStored,
				Fingerprint: "3a7bd3e2360a", Readable: true, CreatedBy: "Admin", CreatedAt: at, LastTest: &test,
			},
		},
		Choice: llm.ChoiceChosen, Active: &llm.Active{CredentialID: llm.VaultID, Model: llm.DefaultModel}, MatchingEnabled: true,
		Haiku:  &llm.Active{CredentialID: llm.VaultID, Model: llm.DefaultModel},
		Budget: llmapp.Budget{MonthlyTokens: 2_000_000, UsedThisMonth: 1234, AveragePerScan: 617, ScansMatched: 2},
		Events: []llm.Event{
			{At: at, Actor: "Admin", Action: llm.ActionCredentialTested, CredentialID: llm.VaultID, Provider: llm.Anthropic, OK: &ok, Message: "ok"},
			{At: at, Actor: "Admin", Action: llm.ActionActiveSet, CredentialID: llm.VaultID, Provider: llm.Anthropic, Model: llm.DefaultModel, Message: "set"},
		},
	}
}

func (l *llmCases) Settings(context.Context) (llmapp.Settings, error) { return l.settings(), l.err }

func (l *llmCases) Add(_ context.Context, actor, provider, label, _ string) (llm.Credential, error) {
	l.actor = actor
	if l.err != nil {
		return llm.Credential{}, l.err
	}
	return llm.Credential{ID: "5b0c1a9e-2d3f-4e5a-8b6c-7d8e9f0a1b2c", Provider: llm.Provider(provider), Label: label, Source: llm.SourceStored, Fingerprint: "3a7bd3e2360a", Readable: true}, nil
}

func (l *llmCases) Delete(_ context.Context, actor, _ string) error {
	l.actor = actor
	return l.err
}

func (l *llmCases) Test(_ context.Context, actor, _ string) (llm.Test, error) {
	l.actor = actor
	return llm.Test{OK: false, At: at, Message: "OpenRouter refused the API key."}, l.err
}

func (l *llmCases) Models(context.Context, string) ([]llm.Model, error) {
	return []llm.Model{{ID: "anthropic/claude-haiku-5-5", Name: "Claude Haiku 5.5"}}, l.err
}

func (l *llmCases) SetActive(_ context.Context, actor, _, _ string) (llmapp.Settings, error) {
	l.actor = actor
	return l.settings(), l.err
}

func (l *llmCases) ClearActive(_ context.Context, actor string) (llmapp.Settings, error) {
	l.actor = actor
	s := l.settings()
	s.Choice, s.Active, s.Haiku, s.MatchingEnabled = llm.ChoiceOff, nil, nil, false
	return s, l.err
}

func TestGetMe(t *testing.T) {
	srv := serve(t, &analyses{})
	r := callAs(t, srv, http.MethodGet, "/api/v1/me", "", "tester")
	if r.code != http.StatusOK || r.body["subject"] != "tester" || r.body["name"] != "Tester" || r.body["email"] != "t@example.com" || r.body["isAdmin"] != false {
		t.Fatalf("me = %d %v", r.code, r.body)
	}
	if r := callAs(t, srv, http.MethodGet, "/api/v1/me", "", "admin"); r.body["isAdmin"] != true || r.body["email"] != nil {
		t.Fatalf("admin me = %v", r.body)
	}
	for cookie, want := range map[string]int{
		"":            http.StatusUnauthorized,
		"stale":       http.StatusUnauthorized,
		"unconfirmed": http.StatusServiceUnavailable,
		"broken":      http.StatusInternalServerError,
	} {
		if r := callAs(t, srv, http.MethodGet, "/api/v1/me", "", cookie); r.code != want {
			t.Errorf("cookie %q: status %d, want %d", cookie, r.code, want)
		}
	}
}

func TestAdminNeedsTheAdministratorRole(t *testing.T) {
	srv := serve(t, &analyses{})
	paths := []struct{ method, path, body string }{
		{http.MethodGet, "/api/v1/admin/llm", ""},
		{http.MethodPost, "/api/v1/admin/llm/credentials", `{"provider":"mistral","label":"x","apiKey":"abcdefgh12"}`},
		{http.MethodDelete, "/api/v1/admin/llm/credentials/vault-anthropic", ""},
		{http.MethodPost, "/api/v1/admin/llm/credentials/vault-anthropic/test", ""},
		{http.MethodGet, "/api/v1/admin/llm/credentials/vault-anthropic/models", ""},
		{http.MethodPut, "/api/v1/admin/llm/active", `{"credentialId":"vault-anthropic","model":"m"}`},
		{http.MethodDelete, "/api/v1/admin/llm/active", ""},
	}
	for _, p := range paths {
		r := callAs(t, srv, p.method, p.path, p.body, "tester")
		if r.code != http.StatusForbidden || r.body["detail"] != "Only an administrator may do this." {
			t.Errorf("%s %s as a user: %d %v", p.method, p.path, r.code, r.body)
		}
		if r := callAs(t, srv, p.method, p.path, p.body, ""); r.code != http.StatusUnauthorized {
			t.Errorf("%s %s anonymous: %d", p.method, p.path, r.code)
		}
		if r := callAs(t, srv, p.method, p.path, p.body, "admin"); r.code >= 300 {
			t.Errorf("%s %s as an admin: %d %v", p.method, p.path, r.code, r.body)
		}
	}
}

func TestLlmSettings(t *testing.T) {
	l := &llmCases{}
	srv := serveWith(t, l)
	r := callAs(t, srv, http.MethodGet, "/api/v1/admin/llm", "", "admin")
	creds, _ := r.body["credentials"].([]any)
	events, _ := r.body["events"].([]any)
	if r.code != http.StatusOK || len(creds) != 2 || len(events) != 2 || r.body["choice"] != "chosen" || r.body["matchingEnabled"] != true {
		t.Fatalf("settings = %d %v", r.code, r.body)
	}
	vault := creds[0].(map[string]any)
	stored := creds[1].(map[string]any)
	if vault["id"] != "vault-anthropic" || vault["source"] != "vault" || vault["createdAt"] != nil || vault["lastTest"] != nil ||
		stored["createdBy"] != "Admin" || stored["lastTest"].(map[string]any)["ok"] != true {
		t.Fatalf("credentials = %v", creds)
	}
	if h := r.body["haiku"].(map[string]any); h["credentialId"] != "vault-anthropic" || h["model"] != "claude-haiku-5-5" {
		t.Fatalf("haiku = %v", h)
	}
	if a := r.body["active"].(map[string]any); a["credentialId"] != "vault-anthropic" || a["model"] != "claude-haiku-5-5" {
		t.Fatalf("active = %v", a)
	}
	if b := r.body["budget"].(map[string]any); b["usedThisMonth"] != float64(1234) || b["averageTokensPerScan"] != float64(617) || b["scansMatched"] != float64(2) {
		t.Fatalf("budget = %v", b)
	}
	if e := events[0].(map[string]any); e["ok"] != true || e["model"] != nil || events[1].(map[string]any)["model"] != "claude-haiku-5-5" {
		t.Fatalf("events = %v", events)
	}

	added := callAs(t, srv, http.MethodPost, "/api/v1/admin/llm/credentials", `{"provider":"openrouter","label":"Course","apiKey":"sk-or-v1-0000"}`, "admin")
	if added.code != http.StatusCreated || added.body["provider"] != "openrouter" || added.body["apiKey"] != nil || l.actor != "Admin" {
		t.Fatalf("add = %d %v by %q", added.code, added.body, l.actor)
	}
	if r := callAs(t, srv, http.MethodPost, "/api/v1/admin/llm/credentials", `{"provider":"openai","label":"x","apiKey":"abcdefgh12"}`, "admin"); r.code != http.StatusBadRequest {
		t.Errorf("unknown provider: %d", r.code)
	}
	if r := callAs(t, srv, http.MethodDelete, "/api/v1/admin/llm/credentials/vault-anthropic", "", "admin"); r.code != http.StatusNoContent {
		t.Errorf("delete: %d", r.code)
	}
	if r := callAs(t, srv, http.MethodDelete, "/api/v1/admin/llm/credentials/nope", "", "admin"); r.code != http.StatusBadRequest {
		t.Errorf("delete a malformed id: %d", r.code)
	}
	if r := callAs(t, srv, http.MethodPost, "/api/v1/admin/llm/credentials/vault-anthropic/test", "", "admin"); r.code != http.StatusOK || r.body["ok"] != false {
		t.Errorf("test = %d %v", r.code, r.body)
	}
	models := callAs(t, srv, http.MethodGet, "/api/v1/admin/llm/credentials/vault-anthropic/models", "", "admin")
	if items, _ := models.body["items"].([]any); models.code != http.StatusOK || len(items) != 1 {
		t.Errorf("models = %d %v", models.code, models.body)
	}
	if r := callAs(t, srv, http.MethodPut, "/api/v1/admin/llm/active", `{"credentialId":"vault-anthropic","model":"claude-haiku-5-5"}`, "admin"); r.code != http.StatusOK || r.body["choice"] != "chosen" {
		t.Errorf("set active = %d %v", r.code, r.body)
	}
	if r := callAs(t, srv, http.MethodDelete, "/api/v1/admin/llm/active", "", "admin"); r.code != http.StatusOK || r.body["choice"] != "off" || r.body["active"] != nil || r.body["haiku"] != nil {
		t.Errorf("clear active = %d %v", r.code, r.body)
	}
}

func TestLlmErrors(t *testing.T) {
	refused := errors.Join(llm.ErrKeyRefused) // keeps the sentinel, and the message is the sentinel's
	cases := []struct {
		err                error
		method, path, body string
		want               int
		detail             string
	}{
		{refused, http.MethodPost, "/api/v1/admin/llm/credentials", `{"provider":"mistral","label":"x","apiKey":"abcdefgh12"}`, http.StatusUnprocessableEntity, "The provider refused the key."},
		{llm.ErrInvalidLabel, http.MethodPost, "/api/v1/admin/llm/credentials", `{"provider":"mistral","label":"x","apiKey":"abcdefgh12"}`, http.StatusUnprocessableEntity, "A label has 1 to 100 characters."},
		{llm.ErrNotFound, http.MethodDelete, "/api/v1/admin/llm/credentials/vault-anthropic", "", http.StatusNotFound, "No credential has this id."},
		{llm.ErrVaultReadOnly, http.MethodDelete, "/api/v1/admin/llm/credentials/vault-anthropic", "", http.StatusUnprocessableEntity, "The Vault key is managed in Vault and cannot be deleted here."},
		{llm.ErrNotFound, http.MethodPost, "/api/v1/admin/llm/credentials/vault-anthropic/test", "", http.StatusNotFound, ""},
		{llm.ErrUnreadable, http.MethodPost, "/api/v1/admin/llm/credentials/vault-anthropic/test", "", http.StatusUnprocessableEntity, ""},
		{llm.ErrNotFound, http.MethodGet, "/api/v1/admin/llm/credentials/vault-anthropic/models", "", http.StatusNotFound, ""},
		{llm.ErrUnreadable, http.MethodGet, "/api/v1/admin/llm/credentials/vault-anthropic/models", "", http.StatusUnprocessableEntity, ""},
		{errors.Join(llm.ErrProvider), http.MethodGet, "/api/v1/admin/llm/credentials/vault-anthropic/models", "", http.StatusBadGateway, "The provider failed."},
		{llm.ErrNotFound, http.MethodPut, "/api/v1/admin/llm/active", `{"credentialId":"vault-anthropic","model":"m"}`, http.StatusNotFound, ""},
		{llm.ErrUnreadable, http.MethodPut, "/api/v1/admin/llm/active", `{"credentialId":"vault-anthropic","model":"m"}`, http.StatusUnprocessableEntity, ""},
	}
	for _, c := range cases {
		srv := serveWith(t, &llmCases{err: c.err})
		r := callAs(t, srv, c.method, c.path, c.body, "admin")
		if r.code != c.want || (c.detail != "" && r.body["detail"] != c.detail) {
			t.Errorf("%s %s with %v: %d %v, want %d %q", c.method, c.path, c.err, r.code, r.body["detail"], c.want, c.detail)
		}
	}
	boom := errors.New("database down")
	srv := serveWith(t, &llmCases{err: boom})
	for _, p := range []struct{ method, path, body string }{
		{http.MethodGet, "/api/v1/admin/llm", ""},
		{http.MethodPost, "/api/v1/admin/llm/credentials", `{"provider":"mistral","label":"x","apiKey":"abcdefgh12"}`},
		{http.MethodDelete, "/api/v1/admin/llm/credentials/vault-anthropic", ""},
		{http.MethodPost, "/api/v1/admin/llm/credentials/vault-anthropic/test", ""},
		{http.MethodGet, "/api/v1/admin/llm/credentials/vault-anthropic/models", ""},
		{http.MethodPut, "/api/v1/admin/llm/active", `{"credentialId":"vault-anthropic","model":"m"}`},
		{http.MethodDelete, "/api/v1/admin/llm/active", ""},
	} {
		if r := callAs(t, srv, p.method, p.path, p.body, "admin"); r.code != http.StatusInternalServerError || r.body["detail"] != nil {
			t.Errorf("%s %s failing: %d %v", p.method, p.path, r.code, r.body)
		}
	}
}
