package main

import (
	"context"
	"testing"
	"time"

	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/pg/pgtest"
)

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
		if code := start(t.Context(), env(map[string]string{"DATABASE_URL": url, "ADDR": "not-an-address"})); code != 1 {
			t.Fatalf("start with an invalid ADDR = %d, want 1", code)
		}
	})
	for _, key := range []string{"ANALYSES_PER_HOUR", "LLM_MONTHLY_TOKENS"} {
		t.Run("bad "+key, func(t *testing.T) {
			if code := start(t.Context(), env(map[string]string{"DATABASE_URL": url, key: "lots"})); code != 1 {
				t.Fatalf("start with %s=lots = %d, want 1", key, code)
			}
		})
	}
	t.Run("serves with matching enabled until cancelled", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
		defer cancel()
		vars := map[string]string{
			"DATABASE_URL": url, "ADDR": "127.0.0.1:0", "ANTHROPIC_API_KEY": "test-key",
			"ANALYSES_PER_HOUR": "3", "LLM_MONTHLY_TOKENS": "1000",
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
