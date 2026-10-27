package domain_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/JorisJonkers-dev/findability-explorer/internal/llm/domain"
)

func TestParseProvider(t *testing.T) {
	for _, p := range []string{"anthropic", "openrouter", "mistral"} {
		if got, err := domain.ParseProvider(p); err != nil || string(got) != p {
			t.Errorf("ParseProvider(%q) = %q, %v", p, got, err)
		}
	}
	if _, err := domain.ParseProvider("openai"); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("openai: %v", err)
	}
}

func TestVaultCredential(t *testing.T) {
	c := domain.VaultCredential("sk-ant-key")
	if c.ID != domain.VaultID || c.Provider != domain.Anthropic || c.Source != domain.SourceVault || !c.Readable ||
		c.Fingerprint != domain.Fingerprint("sk-ant-key") || len(c.Fingerprint) != 12 || c.Label == "" {
		t.Fatalf("vault = %+v", c)
	}
	if domain.Fingerprint("a") == domain.Fingerprint("b") {
		t.Fatal("two keys share a fingerprint")
	}
}

func TestCheckNew(t *testing.T) {
	nc, err := domain.CheckNew("mistral", "  Course key ", " abcdefgh12 ")
	if err != nil || nc.Provider != domain.Mistral || nc.Label != "Course key" || nc.Key != "abcdefgh12" {
		t.Fatalf("CheckNew = %+v, %v", nc, err)
	}
	cases := map[string]struct {
		provider, label, key string
		want                 error
	}{
		"provider":      {"openai", "x", "abcdefgh12", domain.ErrInvalidProvider},
		"blank label":   {"mistral", "  ", "abcdefgh12", domain.ErrInvalidLabel},
		"long label":    {"mistral", strings.Repeat("x", 101), "abcdefgh12", domain.ErrInvalidLabel},
		"control label": {"mistral", "a\x00b", "abcdefgh12", domain.ErrInvalidLabel},
		"short key":     {"mistral", "x", "abc", domain.ErrInvalidKey},
		"long key":      {"mistral", "x", strings.Repeat("k", 501), domain.ErrInvalidKey},
		"spaced key":    {"mistral", "x", "abcd efgh12", domain.ErrInvalidKey},
	}
	for name, c := range cases {
		if _, err := domain.CheckNew(c.provider, c.label, c.key); !errors.Is(err, c.want) || !errors.Is(err, domain.ErrInvalid) {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestCheckModel(t *testing.T) {
	if m, err := domain.CheckModel("anthropic/claude-haiku-5-5"); err != nil || m != "anthropic/claude-haiku-5-5" {
		t.Fatalf("CheckModel = %q, %v", m, err)
	}
	for _, bad := range []string{"", "a b", strings.Repeat("m", 201)} {
		if _, err := domain.CheckModel(bad); !errors.Is(err, domain.ErrInvalidModel) {
			t.Errorf("CheckModel(%q) = %v", bad, err)
		}
	}
}

func TestResolve(t *testing.T) {
	chosen := domain.Setting{Mode: domain.ModeChosen, Active: domain.Active{CredentialID: "c1", Model: "m"}}
	cases := []struct {
		name   string
		s      domain.Setting
		vault  bool
		choice domain.Choice
		active *domain.Active
	}{
		{"chosen", chosen, true, domain.ChoiceChosen, &chosen.Active},
		{"off", domain.Setting{Mode: domain.ModeOff}, true, domain.ChoiceOff, nil},
		{"default with Vault", domain.Setting{}, true, domain.ChoiceDefault, &domain.Active{CredentialID: domain.VaultID, Model: domain.DefaultModel}},
		{"default without Vault", domain.Setting{}, false, domain.ChoiceDefault, nil},
	}
	for _, c := range cases {
		choice, active := domain.Resolve(c.s, c.vault)
		if choice != c.choice || (active == nil) != (c.active == nil) || (active != nil && *active != *c.active) {
			t.Errorf("%s: %s %+v", c.name, choice, active)
		}
	}
}

func TestDescribeAndClip(t *testing.T) {
	if got := domain.Describe(domain.OpenRouter, "anthropic/claude-haiku-5-5"); got != "openrouter: anthropic/claude-haiku-5-5" {
		t.Fatalf("Describe = %q", got)
	}
	if got := domain.Clip("short"); got != "short" {
		t.Fatalf("Clip = %q", got)
	}
	if got := []rune(domain.Clip(strings.Repeat("é", 600))); len(got) != 500 || got[499] != '…' {
		t.Fatalf("Clip of a long message has %d runes", len(got))
	}
}

func TestBackgroundWorkUsesAWorkingAnthropicKey(t *testing.T) {
	vault := domain.VaultCredential("sk-ant-vault")
	stored := domain.Credential{ID: "a", Provider: domain.Anthropic, Readable: true}
	failing := domain.Credential{ID: "f", Provider: domain.Anthropic, Readable: true, LastTest: &domain.Test{OK: false}}
	sealed := domain.Credential{ID: "s", Provider: domain.Anthropic}
	router := domain.Credential{ID: "o", Provider: domain.OpenRouter, Readable: true}
	admin := domain.Credential{ID: "x", Provider: domain.AnthropicAdmin, Readable: true}
	if c, ok := domain.AnthropicKey([]domain.Credential{vault, stored}); !ok || c.ID != domain.VaultID {
		t.Errorf("with the Vault key = %+v, %v", c, ok)
	}
	if c, ok := domain.AnthropicKey([]domain.Credential{router, admin, failing, sealed, stored}); !ok || c.ID != "a" {
		t.Errorf("without it = %+v, %v", c, ok)
	}
	if _, ok := domain.AnthropicKey([]domain.Credential{router, failing}); ok {
		t.Error("an OpenRouter or failing key was chosen")
	}
}
