package persistence_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/JorisJonkers-dev/findability-explorer/internal/llm/adapters/persistence"
	"github.com/JorisJonkers-dev/findability-explorer/internal/llm/domain"
	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/pg"
	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/pg/pgtest"
	"github.com/JorisJonkers-dev/findability-explorer/internal/shared/seal"
)

var at = time.Date(2026, 10, 23, 9, 0, 0, 0, time.UTC)

func sealer(t *testing.T, secret string) *seal.Sealer {
	t.Helper()
	s, err := seal.NewSealer(secret, persistence.HKDFInfo)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func open(t *testing.T) (*pg.Store, *persistence.Repository) {
	t.Helper()
	db, _, r := openURL(t)
	return db, r
}

func openURL(t *testing.T) (*pg.Store, string, *persistence.Repository) {
	t.Helper()
	url := pgtest.URL(t)
	db, err := pg.Open(t.Context(), url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(db.Close)
	return db, url, persistence.New(db, sealer(t, strings.Repeat("k", 32)))
}

func credential(id string, p domain.Provider, at time.Time) domain.Credential {
	return domain.Credential{
		ID: id, Provider: p, Label: "Key " + id[:4], Source: domain.SourceStored, Fingerprint: domain.Fingerprint(id),
		Readable: true, CreatedBy: "joris", CreatedAt: at,
	}
}

const (
	id1 = "11111111-1111-4111-8111-111111111111"
	id2 = "22222222-2222-4222-8222-222222222222"
)

func TestCredentials(t *testing.T) {
	t.Parallel()
	db, url, r := openURL(t)
	ctx := t.Context()
	if err := r.Add(ctx, credential(id2, domain.Mistral, at.Add(time.Minute)), "mistral-key"); err != nil {
		t.Fatal(err)
	}
	if err := r.Add(ctx, credential(id1, domain.OpenRouter, at), "sk-or-key"); err != nil {
		t.Fatal(err)
	}
	if err := r.Add(ctx, credential("not-a-uuid", domain.Mistral, at), "k"); err == nil {
		t.Fatal("a credential without a UUID was stored")
	}
	if err := r.Add(ctx, credential(id1, domain.Mistral, at), "k"); err == nil {
		t.Fatal("the same id was stored twice")
	}
	list, err := r.List(ctx)
	if err != nil || len(list) != 2 || list[0].ID != id1 || list[1].ID != id2 || !list[0].Readable ||
		list[0].Provider != domain.OpenRouter || list[0].CreatedBy != "joris" || !list[0].CreatedAt.Equal(at) {
		t.Fatalf("list = %+v, %v", list, err)
	}
	if c, err := r.Get(ctx, id1); err != nil || c.Label != "Key 1111" || c.Fingerprint != domain.Fingerprint(id1) {
		t.Fatalf("get = %+v, %v", c, err)
	}
	for _, id := range []string{"nope", "33333333-3333-4333-8333-333333333333"} {
		if _, err := r.Get(ctx, id); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("get %s: %v", id, err)
		}
		if _, _, err := r.Secret(ctx, id); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("secret %s: %v", id, err)
		}
		if _, err := r.Delete(ctx, id); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("delete %s: %v", id, err)
		}
	}
	if p, key, err := r.Secret(ctx, id1); err != nil || p != domain.OpenRouter || key != "sk-or-key" {
		t.Fatalf("secret = %s %q, %v", p, key, err)
	}

	// A rotated SESSION_KEY makes every stored key unreadable.
	rotated := persistence.New(db, sealer(t, strings.Repeat("r", 32)))
	if list, _ := rotated.List(ctx); list[0].Readable {
		t.Fatal("a key sealed under another SESSION_KEY reads as readable")
	}
	if _, _, err := rotated.Secret(ctx, id1); !errors.Is(err, domain.ErrUnreadable) {
		t.Fatalf("secret under another key: %v", err)
	}
	// A sealed key cannot move to another row.
	conn, err := pgx.Connect(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close(ctx) }()
	if _, err := conn.Exec(ctx, "UPDATE llm_credentials SET secret = (SELECT secret FROM llm_credentials WHERE id = $1) WHERE id = $2", id1, id2); err != nil {
		t.Fatal(err)
	}
	if _, _, err := r.Secret(ctx, id2); !errors.Is(err, domain.ErrUnreadable) {
		t.Fatalf("a key copied to another row opened: %v", err)
	}
}

func TestActiveSetting(t *testing.T) {
	t.Parallel()
	_, r := open(t)
	ctx := t.Context()
	if s, err := r.Setting(ctx); err != nil || s.Mode != domain.ModeUnset {
		t.Fatalf("initial setting = %+v, %v", s, err)
	}
	if err := r.Add(ctx, credential(id1, domain.Mistral, at), "k-mistral"); err != nil {
		t.Fatal(err)
	}
	chosen := domain.Setting{Mode: domain.ModeChosen, Active: domain.Active{CredentialID: id1, Model: "mistral-small-latest"}}
	if err := r.SaveSetting(ctx, chosen, "joris", at); err != nil {
		t.Fatal(err)
	}
	if s, err := r.Setting(ctx); err != nil || s != chosen {
		t.Fatalf("chosen = %+v, %v", s, err)
	}
	if was, err := r.Delete(ctx, id1); err != nil || !was {
		t.Fatalf("deleting the active key = %v, %v", was, err)
	}
	if s, _ := r.Setting(ctx); s.Mode != domain.ModeUnset {
		t.Fatalf("after deleting the active key: %+v", s)
	}
	if err := r.SaveSetting(ctx, domain.Setting{Mode: domain.ModeOff}, "joris", at); err != nil {
		t.Fatal(err)
	}
	if s, _ := r.Setting(ctx); s.Mode != domain.ModeOff || s.Active != (domain.Active{}) {
		t.Fatalf("off = %+v", s)
	}
	if err := r.Add(ctx, credential(id2, domain.Mistral, at), "k2-mistral"); err != nil {
		t.Fatal(err)
	}
	if was, err := r.Delete(ctx, id2); err != nil || was {
		t.Fatalf("deleting an inactive key = %v, %v", was, err)
	}
	if err := r.SaveSetting(ctx, domain.Setting{Mode: "sideways"}, "joris", at); err == nil {
		t.Fatal("an unknown mode was stored")
	}
}

func TestEvents(t *testing.T) {
	t.Parallel()
	_, r := open(t)
	ctx := t.Context()
	yes, no := true, false
	events := []domain.Event{
		{At: at, Actor: "joris", Action: domain.ActionCredentialAdded, CredentialID: id1, Provider: domain.Mistral, OK: &yes, Message: "added"},
		{At: at.Add(time.Minute), Actor: "joris", Action: domain.ActionCredentialTested, CredentialID: id1, Provider: domain.Mistral, OK: &no, Message: "refused"},
		{At: at.Add(2 * time.Minute), Actor: "joris", Action: domain.ActionCredentialTested, CredentialID: domain.VaultID, Provider: domain.Anthropic, OK: &yes, Message: "ok"},
		{At: at.Add(3 * time.Minute), Actor: "joris", Action: domain.ActionActiveSet, CredentialID: domain.VaultID, Provider: domain.Anthropic, Model: "m", Message: "set"},
		{At: at.Add(4 * time.Minute), Actor: "joris", Action: domain.ActionActiveCleared, Message: "off"},
	}
	for _, e := range events {
		if err := r.AddEvent(ctx, e); err != nil {
			t.Fatal(err)
		}
	}
	got, err := r.Events(ctx, 50)
	if err != nil || len(got) != 5 || got[0].Action != domain.ActionActiveCleared || got[0].OK != nil || got[0].CredentialID != "" ||
		got[1].Model != "m" || *got[3].OK || !got[4].At.Equal(at) {
		t.Fatalf("events = %+v, %v", got, err)
	}
	if got, _ := r.Events(ctx, 2); len(got) != 2 {
		t.Fatalf("limited to %d", len(got))
	}
	tests, err := r.LastTests(ctx)
	if err != nil || len(tests) != 2 || tests[id1].OK || tests[id1].Message != "refused" || !tests[domain.VaultID].OK {
		t.Fatalf("tests = %+v, %v", tests, err)
	}
	if err := r.AddEvent(ctx, domain.Event{At: at, Actor: "joris", Action: "exploded", Message: "x"}); err == nil {
		t.Fatal("an unknown action was stored")
	}
}

func TestDatabaseFailures(t *testing.T) {
	t.Parallel()
	db, r := open(t)
	db.Close()
	ctx := t.Context()
	if err := r.Add(ctx, credential(id1, domain.Mistral, at), "k"); err == nil {
		t.Error("Add")
	}
	if _, err := r.List(ctx); err == nil {
		t.Error("List")
	}
	if _, err := r.Get(ctx, id1); err == nil || errors.Is(err, domain.ErrNotFound) {
		t.Errorf("Get: %v", err)
	}
	if _, err := r.Delete(ctx, id1); err == nil || errors.Is(err, domain.ErrNotFound) {
		t.Errorf("Delete: %v", err)
	}
	if _, err := r.Setting(ctx); err == nil {
		t.Error("Setting")
	}
	if _, err := r.Events(ctx, 5); err == nil {
		t.Error("Events")
	}
	if _, err := r.LastTests(ctx); err == nil {
		t.Error("LastTests")
	}
}

func TestAnAdminKeyIsStoredLikeTheOthers(t *testing.T) {
	t.Parallel()
	_, _, r := openURL(t)
	ctx := t.Context()
	if err := r.Add(ctx, credential(id1, domain.AnthropicAdmin, at), "sk-ant-admin01-key"); err != nil {
		t.Fatal(err)
	}
	if err := r.AddEvent(ctx, domain.Event{At: at, Actor: "joris", Action: domain.ActionCredentialAdded, CredentialID: id1, Provider: domain.AnthropicAdmin, Message: "Added."}); err != nil {
		t.Fatal(err)
	}
	if p, key, err := r.Secret(ctx, id1); err != nil || p != domain.AnthropicAdmin || key != "sk-ant-admin01-key" {
		t.Fatalf("secret = %s %q, %v", p, key, err)
	}
}
