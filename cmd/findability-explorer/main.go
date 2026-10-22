// Command findability-explorer serves the API and the embedded web app, migrates its database at
// startup, runs queued website analyses in the background, and drains on SIGTERM.
//
// Environment:
//   - DATABASE_URL, or PGHOST with the other libpq variables, which is how the platform hands over
//     a rotating database login (required)
//   - ADDR (default :8080)
//   - ANTHROPIC_API_KEY: enables product matching with Claude; without it matching is disabled
//   - LLM_MONTHLY_TOKENS: the language-model budget per calendar month (default 2000000)
//   - ANALYSES_PER_HOUR: how many analyses one user may start per hour (default 10)
//   - CRAWLER_CONTACT: a contact address for the crawler's User-Agent
//   - DEV_USER, which stands in for the platform's forward-auth on a local run and must never be
//     set in a deployment
package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"

	"github.com/JorisJonkers-dev/findability-explorer/internal/analysis/adapters/crawler"
	"github.com/JorisJonkers-dev/findability-explorer/internal/analysis/adapters/matcher"
	"github.com/JorisJonkers-dev/findability-explorer/internal/analysis/adapters/persistence"
	analysisapp "github.com/JorisJonkers-dev/findability-explorer/internal/analysis/app"
	analysisdomain "github.com/JorisJonkers-dev/findability-explorer/internal/analysis/domain"
	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/httpapi"
	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/httpx"
	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/pg"
	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/webui"
	"github.com/JorisJonkers-dev/findability-explorer/internal/server"
	"github.com/JorisJonkers-dev/findability-explorer/internal/terminology/adapters/seed"
	terminologyapp "github.com/JorisJonkers-dev/findability-explorer/internal/terminology/app"
	terminologydomain "github.com/JorisJonkers-dev/findability-explorer/internal/terminology/domain"
	"github.com/JorisJonkers-dev/findability-explorer/web"
)

// version is set at build time with -ldflags "-X main.version=…".
var version = "dev"

const defaultAddr = ":8080"

func main() {
	os.Exit(start(context.Background(), os.Getenv))
}

// start is main minus os.Exit, so tests can drive it.
func start(parent context.Context, getenv func(string) string) int {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	ctx, stop := signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, logger, getenv); err != nil {
		logger.Error("findability-explorer stopped", "error", err)
		return 1
	}
	return 0
}

// run is the composition root: it reads the environment and builds the store, the use cases and
// the server. httpapi assembles the contexts' web adapters into the one generated API.
func run(ctx context.Context, logger *slog.Logger, getenv func(string) string) error {
	dbURL, err := databaseOf(getenv)
	if err != nil {
		return err
	}
	if err := pg.Migrate(ctx, dbURL); err != nil {
		return err
	}
	store, err := pg.Open(ctx, dbURL)
	if err != nil {
		return err
	}
	defer store.Close()

	catalogue, err := seed.Load()
	if err != nil {
		return err
	}
	terminology := terminologyapp.New(catalogue)
	analysis, err := newAnalysis(ctx, logger, getenv, store, terminology)
	if err != nil {
		return err
	}
	go analysis.Work(ctx, func(id uuid.UUID, err error) {
		logger.ErrorContext(ctx, "analysis could not be recorded", "analysis", id, "error", err)
	})

	api, err := httpapi.New(logger, httpapi.Contexts{Terminology: terminology, Analysis: analysis})
	if err != nil {
		return err
	}
	if subject := getenv("DEV_USER"); subject != "" {
		logger.Warn("DEV_USER is set: every request without an identity runs as it", "subject", subject)
		api = httpx.DevIdentity(subject, api)
	}

	addr := getenv("ADDR")
	if addr == "" {
		addr = defaultAddr
	}
	var lc net.ListenConfig
	ln, err := lc.Listen(ctx, "tcp", addr)
	if err != nil {
		return err
	}
	return server.New(logger, version, server.Routes{
		API:   api,
		Web:   webui.Handler(web.Dist()),
		Ready: store.Ping,
	}).Serve(ctx, ln)
}

// newAnalysis builds the analysis use cases and queues every analysis a restart left waiting.
func newAnalysis(ctx context.Context, logger *slog.Logger, getenv func(string) string, store *pg.Store,
	terminology *terminologyapp.Service,
) (*analysisapp.Service, error) {
	perHour, err := intEnv(getenv, "ANALYSES_PER_HOUR", 10)
	if err != nil {
		return nil, err
	}
	monthly, err := intEnv(getenv, "LLM_MONTHLY_TOKENS", 2_000_000)
	if err != nil {
		return nil, err
	}
	var match analysisdomain.Matcher
	if key := getenv("ANTHROPIC_API_KEY"); key != "" {
		match = matcher.New(key)
	} else {
		logger.Warn("ANTHROPIC_API_KEY is not set: product matching is disabled")
	}
	svc := analysisapp.New(
		persistence.New(store),
		crawler.New(crawler.Options{Contact: getenv("CRAWLER_CONTACT")}),
		match,
		func() []analysisdomain.GlossaryTerm { return glossary(terminology.Terms()) },
		func() time.Time { return time.Now().UTC() },
		analysisapp.Limits{PerUserPerHour: perHour, MonthlyTokens: int64(monthly), SiteTextChars: 60_000, QueueSize: 32},
	)
	if err := svc.Recover(ctx); err != nil {
		return nil, err
	}
	return svc, nil
}

// glossary hands the terminology map to the analysis context by value.
func glossary(terms []terminologydomain.Term) []analysisdomain.GlossaryTerm {
	out := make([]analysisdomain.GlossaryTerm, 0, len(terms))
	for _, t := range terms {
		out = append(out, analysisdomain.GlossaryTerm{
			Text: t.Text, Language: string(t.Language), Family: string(t.Family), Tier: string(t.Tier),
			UsedByTribelt: slices.ContainsFunc(t.UsedBy, func(u string) bool { return strings.HasPrefix(u, "Tribelt") }),
		})
	}
	return out
}

func intEnv(getenv func(string) string, key string, fallback int) (int, error) {
	raw := getenv(key)
	if raw == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		return 0, errors.New(key + " must be a whole number of at least 0")
	}
	return n, nil
}

// libpq is each libpq variable the platform hands a Process whose database login rotates, and the
// connection-string keyword it becomes.
var libpq = []struct{ variable, keyword string }{ //nolint:gochecknoglobals // a constant table
	{"PGHOST", "host"},
	{"PGPORT", "port"},
	{"PGDATABASE", "dbname"},
	{"PGUSER", "user"},
	{"PGPASSWORD", "password"},
	{"PGSSLMODE", "sslmode"},
}

// quoted is a connection-string value in single quotes, its backslashes and quotes escaped.
var quoted = strings.NewReplacer(`\`, `\\`, `'`, `\'`) //nolint:gochecknoglobals // a constant

// databaseOf is DATABASE_URL, or a connection string of the libpq variables where it is unset.
func databaseOf(getenv func(string) string) (string, error) {
	if url := getenv("DATABASE_URL"); url != "" {
		return url, nil
	}
	if getenv("PGHOST") == "" {
		return "", errors.New("neither DATABASE_URL nor PGHOST is set")
	}
	parts := make([]string, 0, len(libpq))
	for _, p := range libpq {
		if value := getenv(p.variable); value != "" {
			parts = append(parts, p.keyword+"='"+quoted.Replace(value)+"'")
		}
	}
	return strings.Join(parts, " "), nil
}
