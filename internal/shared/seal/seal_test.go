package seal_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/JorisJonkers-dev/findability-explorer/internal/shared/seal"
)

var secret = strings.Repeat("s", 32)

func TestSealer(t *testing.T) {
	s, err := seal.NewSealer(secret, "test-v1")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(s.KeyID(), "v1-") || len(s.KeyID()) != 11 {
		t.Fatalf("key id %q", s.KeyID())
	}
	blob, err := s.Seal("row-1", []byte("sk-secret"))
	if err != nil || strings.Contains(string(blob), "sk-secret") {
		t.Fatalf("blob %q, %v", blob, err)
	}
	if plain, err := s.Open("row-1", s.KeyID(), blob); err != nil || string(plain) != "sk-secret" {
		t.Fatalf("open = %q, %v", plain, err)
	}
	if _, err := s.Open("row-2", s.KeyID(), blob); !errors.Is(err, seal.ErrTampered) {
		t.Fatalf("moved to another row: %v", err)
	}
	if _, err := s.Open("row-1", s.KeyID(), blob[:5]); !errors.Is(err, seal.ErrTampered) {
		t.Fatalf("truncated: %v", err)
	}
	if _, err := s.Open("row-1", "v1-00000000", blob); !errors.Is(err, seal.ErrKeyChanged) {
		t.Fatalf("other key id: %v", err)
	}
	other, _ := seal.NewSealer(secret, "other-v1")
	if other.KeyID() == s.KeyID() {
		t.Fatal("two uses share a key id")
	}
	if _, err := other.Open("row-1", s.KeyID(), blob); !errors.Is(err, seal.ErrKeyChanged) {
		t.Fatalf("opened by another use: %v", err)
	}
	if _, err := seal.NewSealer("short", "x"); !errors.Is(err, seal.ErrShortSecret) {
		t.Fatalf("short secret: %v", err)
	}
}

func TestCodec(t *testing.T) {
	c, err := seal.NewCodec(secret, "cookie-v1")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 23, 9, 0, 0, 0, time.UTC)
	v, err := c.Seal("fe_session", "sid-1", now, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	var got string
	if err := c.Open("fe_session", v, now.Add(59*time.Minute), &got); err != nil || got != "sid-1" {
		t.Fatalf("open = %q, %v", got, err)
	}
	var n int
	for name, check := range map[string]func() error{
		"expired":      func() error { return c.Open("fe_session", v, now.Add(time.Hour), &got) },
		"other cookie": func() error { return c.Open("fe_oidc", v, now, &got) },
		"not base64":   func() error { return c.Open("fe_session", "!!", now, &got) },
		"too short":    func() error { return c.Open("fe_session", "AAAA", now, &got) },
		"wrong type":   func() error { return c.Open("fe_session", v, now, &n) },
	} {
		if err := check(); !errors.Is(err, seal.ErrTampered) {
			t.Errorf("%s: %v", name, err)
		}
	}
	if _, err := c.Seal("x", func() {}, now, time.Hour); err == nil {
		t.Fatal("a function was sealed")
	}
	if _, err := seal.NewCodec("short", "x"); !errors.Is(err, seal.ErrShortSecret) {
		t.Fatalf("short secret: %v", err)
	}
}
