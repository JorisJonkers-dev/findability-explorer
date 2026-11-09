// Command findability-explorer serves the API, the sign-in routes and the embedded web app,
// migrates its database at startup, runs queued website analyses in the background, and drains on
// SIGTERM. The whole app is private: every page and API call needs an auth-api sign-in, and the
// admin settings need auth-api's ROLE_ADMIN (docs/adr/0001).
//
// Environment:
//   - DATABASE_URL, or PGHOST with the other libpq variables, which is how the platform hands over
//     a rotating database login (required)
//   - ADDR (default :8080)
//   - OIDC_ISSUER, OIDC_CLIENT_ID, OIDC_CLIENT_SECRET, OIDC_REDIRECT_URL: the client registration
//     at auth-api (required unless DEV_USER is set); cookies are Secure unless the redirect URL is
//     plain http
//   - SESSION_KEY: at least 32 characters; seals the session cookie, the stored refresh tokens and
//     the language-model API keys an admin adds (required unless DEV_USER is set)
//   - ANTHROPIC_API_KEY: the Vault-delivered key, shown as the read-only credential
//     vault-anthropic; until an admin chooses, product matching uses it with claude-haiku-5-5
//   - LLM_MONTHLY_TOKENS: the language-model budget per calendar month (default 10000000)
//   - CRAWLER_CONTACT: a contact address for the crawler's User-Agent
//   - DEV_USER, and DEV_ROLES (comma-separated, such as ROLE_ADMIN): stand in for a signed-in
//     account on a local run, without auth-api; never set in a deployment
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
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
	companiespersistence "github.com/JorisJonkers-dev/findability-explorer/internal/companies/adapters/persistence"
	"github.com/JorisJonkers-dev/findability-explorer/internal/companies/adapters/research"
	companiesapp "github.com/JorisJonkers-dev/findability-explorer/internal/companies/app"
	companiesdomain "github.com/JorisJonkers-dev/findability-explorer/internal/companies/domain"
	"github.com/JorisJonkers-dev/findability-explorer/internal/identity/adapters/oidc"
	identitypersistence "github.com/JorisJonkers-dev/findability-explorer/internal/identity/adapters/persistence"
	identityweb "github.com/JorisJonkers-dev/findability-explorer/internal/identity/adapters/web"
	identityapp "github.com/JorisJonkers-dev/findability-explorer/internal/identity/app"
	identitydomain "github.com/JorisJonkers-dev/findability-explorer/internal/identity/domain"
	insightpersistence "github.com/JorisJonkers-dev/findability-explorer/internal/insight/adapters/persistence"
	llmpersistence "github.com/JorisJonkers-dev/findability-explorer/internal/llm/adapters/persistence"
	"github.com/JorisJonkers-dev/findability-explorer/internal/llm/adapters/providers"
	llmapp "github.com/JorisJonkers-dev/findability-explorer/internal/llm/app"
	llmdomain "github.com/JorisJonkers-dev/findability-explorer/internal/llm/domain"
	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/httpapi"
	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/pg"
	"github.com/JorisJonkers-dev/findability-explorer/internal/platform/webui"
	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/adapters/extractor"
	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/adapters/openrouter"
	prompttestpersistence "github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/adapters/persistence"
	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/adapters/reviewer"
	prompttestapp "github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/app"
	prompttestdomain "github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain"
	"github.com/JorisJonkers-dev/findability-explorer/internal/server"
	"github.com/JorisJonkers-dev/findability-explorer/internal/shared/seal"
	"github.com/JorisJonkers-dev/findability-explorer/internal/terminology/adapters/judge"
	terminologypersistence "github.com/JorisJonkers-dev/findability-explorer/internal/terminology/adapters/persistence"
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

	secret, err := sessionKey(logger, getenv)
	if err != nil {
		return err
	}
	catalogue, err := seed.Load()
	if err != nil {
		return err
	}
	now := func() time.Time { return time.Now().UTC() }
	analysisRepo := persistence.New(store)
	var companies *companiesapp.Service
	terminologyRepo := terminologypersistence.New(store)
	terminology := terminologyapp.New(catalogue, terminologyRepo,
		func(ctx context.Context) ([]terminologydomain.Observation, error) {
			return observations(ctx, analysisRepo, companies)
		},
		func(ctx context.Context) ([]terminologydomain.Sighting, error) { return sightings(ctx, analysisRepo) },
		now)
	if err := terminology.Load(ctx); err != nil {
		return err
	}
	companies = companiesapp.New(
		func() []companiesdomain.Company {
			return append(research.Companies(terminology.Companies()), companiesdomain.Mirror())
		},
		companiespersistence.New(store.Queries()),
		scans{repo: analysisRepo},
		now,
	)
	limits, err := limitsOf(getenv)
	if err != nil {
		return err
	}
	llm, err := newLLM(logger, getenv, store, analysisRepo, terminologyRepo, secret, limits, now)
	if err != nil {
		return err
	}
	// Haiku reviews the phrases scans find, on an Anthropic key.
	terminology.WithReviewer(terminologyRepo, judges{llm: llm})
	go terminology.ReviewLoop(ctx, termReviewEvery, func(err error) { logger.WarnContext(ctx, "term review", "error", err) })
	analysis, err := newAnalysis(ctx, getenv, analysisRepo, directory{companies: companies}, terminology, llm, limits, now)
	if err != nil {
		return err
	}
	go analysis.Work(ctx, func(id uuid.UUID, err error) {
		logger.ErrorContext(ctx, "analysis could not be recorded", "analysis", id, "error", err)
	})

	promptTest := prompttestapp.New(prompttestpersistence.New(store), gateways{llm: llm}, promptDirectory{companies: companies}, now,
		prompttestapp.Config{}, func(id uuid.UUID, err error) {
			logger.ErrorContext(ctx, "prompt run could not be recorded", "run", id, "error", err)
		},
		// Haiku on an Anthropic key reviews the competitor candidates (ADR-0006).
		prompttestapp.WithReviewers(reviewers{llm: llm}, func(err error) { logger.WarnContext(ctx, "candidate review", "error", err) }),
		// Haiku on an Anthropic key reads the answers while one works (ADR-0007).
		prompttestapp.WithExtractors(extractors{llm: llm}))
	go promptTest.Loop(ctx, func(err error) { logger.WarnContext(ctx, "prompt test schedule", "error", err) })
	insight := newInsight(insightpersistence.New(store.Queries()), companies, promptTest, analysis, terminology, now)
	companies.WithBatch(candidateRecorder{prompts: promptTest}, scanner{analysis: analysis})

	gate, err := newGate(ctx, logger, getenv, store, secret, now)
	if err != nil {
		return err
	}
	api, err := httpapi.New(logger, httpapi.Contexts{
		Terminology: terminology, Analysis: analysis, Companies: companies, LLM: llm, PromptTest: promptTest, Insight: insight, Auth: gate.auth,
	})
	if err != nil {
		return err
	}
	if gate.dev {
		api = identityweb.WithDevCookie(api)
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
		Auth:  gate.routes,
		Web:   identityweb.Gate(gate.auth, logger, webui.Handler(web.Dist())),
		Ready: store.Ping,
	}).Serve(ctx, ln)
}

// gate is how requests are authenticated: auth-api sessions, or the DEV_USER stand-in.
type gate struct {
	auth   identityweb.Authenticator
	routes http.Handler
	dev    bool
}

// oidcVars are the variables a deployment needs to sign people in with auth-api.
var oidcVars = []string{"OIDC_ISSUER", "OIDC_CLIENT_ID", "OIDC_CLIENT_SECRET", "OIDC_REDIRECT_URL"} //nolint:gochecknoglobals // a constant list

// newGate signs people in with auth-api, or with DEV_USER on a local run.
func newGate(ctx context.Context, logger *slog.Logger, getenv func(string) string, store *pg.Store, secret string,
	now func() time.Time,
) (gate, error) {
	if subject := getenv("DEV_USER"); subject != "" {
		roles := identitydomain.ParseRoles(getenv("DEV_ROLES"))
		logger.Warn("DEV_USER is set: every request runs as it, without signing in", "subject", subject, "roles", roles)
		dev := identityweb.Dev{Principal: identitydomain.Principal{Subject: subject, Name: subject, Roles: roles}}
		return gate{auth: dev, routes: identityweb.DevRoutes(), dev: true}, nil
	}
	var missing []string
	for _, key := range oidcVars {
		if getenv(key) == "" {
			missing = append(missing, key)
		}
	}
	if len(missing) > 0 {
		return gate{}, fmt.Errorf("signing in needs %s (or DEV_USER on a local run)", strings.Join(missing, ", "))
	}
	redirect, err := url.Parse(getenv("OIDC_REDIRECT_URL"))
	if err != nil || (redirect.Scheme != "https" && redirect.Scheme != "http") || redirect.Host == "" {
		return gate{}, errors.New("OIDC_REDIRECT_URL must be an absolute http or https address")
	}
	client, err := oidc.New(ctx, oidc.Config{
		Issuer: getenv("OIDC_ISSUER"), ClientID: getenv("OIDC_CLIENT_ID"),
		ClientSecret: getenv("OIDC_CLIENT_SECRET"), RedirectURL: redirect.String(),
	})
	if err != nil {
		return gate{}, err
	}
	sealer, err := seal.NewSealer(secret, identitypersistence.HKDFInfo)
	if err != nil {
		return gate{}, err
	}
	codec, err := seal.NewCodec(secret, identityweb.CodecInfo)
	if err != nil {
		return gate{}, err
	}
	sessions := identityapp.New(identitypersistence.New(store, sealer), client, client.Issuer(), now,
		func(msg string, err error) { logger.Warn(msg, "error", err) })
	go prune(ctx, logger, sessions)
	return gate{
		auth:   identityweb.NewSessions(sessions, codec, now),
		routes: identityweb.NewRoutes(client, sessions, codec, redirect.Scheme == "https", now, logger).Handler(),
	}, nil
}

// prune deletes expired sessions every hour.
func prune(ctx context.Context, logger *slog.Logger, sessions *identityapp.Service) {
	tick := time.NewTicker(time.Hour)
	defer tick.Stop()
	for {
		if err := sessions.Prune(ctx); err != nil && ctx.Err() == nil {
			logger.Warn("prune expired sessions", "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		}
	}
}

// sessionKey is SESSION_KEY. A local run with DEV_USER may go without; it then gets a key of its
// own, and the API keys it stores cannot be read after a restart.
func sessionKey(logger *slog.Logger, getenv func(string) string) (string, error) {
	secret := getenv("SESSION_KEY")
	switch {
	case secret == "" && getenv("DEV_USER") != "":
		b := make([]byte, 32)
		_, _ = rand.Read(b)
		logger.Warn("SESSION_KEY is not set: API keys added on this run cannot be read after a restart")
		return hex.EncodeToString(b), nil
	case len(secret) < seal.MinSecretLength:
		return "", fmt.Errorf("SESSION_KEY is required and has at least %d characters", seal.MinSecretLength)
	default:
		return secret, nil
	}
}

// limits are what analyses may cost.
type limits struct {
	monthly int64
}

func limitsOf(getenv func(string) string) (limits, error) {
	monthly, err := intEnv(getenv, "LLM_MONTHLY_TOKENS", 10_000_000)
	if err != nil {
		return limits{}, err
	}
	return limits{monthly: int64(monthly)}, nil
}

// newLLM builds the language-model settings: the Vault key, the keys admins add, and the active model.
func newLLM(logger *slog.Logger, getenv func(string) string, store *pg.Store, analyses *persistence.Repository,
	reviews *terminologypersistence.Repository, secret string, l limits, now func() time.Time,
) (*llmapp.Service, error) {
	sealer, err := seal.NewSealer(secret, llmpersistence.HKDFInfo)
	if err != nil {
		return nil, err
	}
	vault := getenv("ANTHROPIC_API_KEY")
	if vault == "" {
		logger.Info("ANTHROPIC_API_KEY is not set: product matching is off until an admin adds an Anthropic key or chooses a model")
	}
	spent := func(ctx context.Context) (llmapp.Spending, error) {
		month, err := analyses.TokensSince(ctx, monthStart(now()))
		if err != nil {
			return llmapp.Spending{}, err
		}
		reviewed, err := reviews.TokensSince(ctx, monthStart(now()))
		if err != nil {
			return llmapp.Spending{}, err
		}
		month += reviewed
		scans, average, err := analyses.TokensPerScan(ctx)
		if err != nil {
			return llmapp.Spending{}, err
		}
		return llmapp.Spending{ThisMonth: month, AveragePerScan: average, ScansMatched: scans}, nil
	}
	connector := providers.New()
	return llmapp.New(llmpersistence.New(store, sealer), connector, connector, spent,
		llmapp.Config{VaultKey: vault, MonthlyTokens: l.monthly},
		now, uuid.NewString), nil
}

func monthStart(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
}

// termReviewEvery is how often the language model looks for new phrases to review.
const termReviewEvery = 10 * time.Minute

// reviewers hands the prompt-test context its reviewer of competitor candidates: Haiku on an
// Anthropic key, as for the term review, since it is cheap there.
type reviewers struct {
	llm *llmapp.Service
}

func (r reviewers) Current(ctx context.Context) (prompttestapp.Reviewer, string, error) {
	client, model, err := r.llm.Haiku(ctx)
	if err != nil || client == nil {
		return nil, "", err
	}
	return reviewer.New(client, model), llmdomain.Describe(llmdomain.Anthropic, model), nil
}

// judges hands the terminology context its judge: Haiku on an Anthropic key, whatever model
// product matching uses, since reviewing hundreds of phrases must stay cheap.
type judges struct {
	llm *llmapp.Service
}

func (j judges) Current(ctx context.Context) (terminologyapp.Judge, string, error) {
	client, model, err := j.llm.Haiku(ctx)
	if err != nil || client == nil {
		return nil, "", err
	}
	return judge.New(client, model), llmdomain.Describe(llmdomain.Anthropic, model), nil
}

// extractors hands the prompt-test context its reader of the answers: Haiku on an Anthropic key,
// whose usage is subsidised (ADR-0007). Without one, OpenRouter reads them as before.
type extractors struct {
	llm *llmapp.Service
}

func (x extractors) Current(ctx context.Context) (prompttestapp.Extractor, string, error) {
	client, model, err := x.llm.Haiku(ctx)
	if err != nil || client == nil {
		return nil, "", err
	}
	return extractor.New(client, model), llmdomain.Describe(llmdomain.Anthropic, model), nil
}

// matchers hands the analysis context its matcher: Haiku on an Anthropic key while one works
// (ADR-0007), else the model the language-model settings name now.
type matchers struct {
	llm *llmapp.Service
}

func (m matchers) Current(ctx context.Context) (analysisdomain.Matcher, string, error) {
	haiku, model, err := m.llm.Haiku(ctx)
	if err != nil {
		return nil, "", err
	}
	if haiku != nil {
		return matcher.New(haiku, model), llmdomain.Describe(llmdomain.Anthropic, model), nil
	}
	client, provider, model, err := m.llm.Current(ctx)
	if err != nil || client == nil {
		return nil, "", err
	}
	return matcher.New(client, model), llmdomain.Describe(provider, model), nil
}

// newAnalysis builds the analysis use cases and queues every analysis a restart left waiting.
func newAnalysis(ctx context.Context, getenv func(string) string, repo *persistence.Repository,
	directory analysisdomain.Directory, terminology *terminologyapp.Service, llm *llmapp.Service, l limits,
	now func() time.Time,
) (*analysisapp.Service, error) {
	svc := analysisapp.New(
		repo,
		directory,
		crawler.New(crawler.Options{Contact: getenv("CRAWLER_CONTACT")}),
		matchers{llm: llm},
		func() []analysisdomain.GlossaryTerm { return glossary(terminology.Terms()) },
		now,
		analysisapp.Limits{
			MonthlyTokens: l.monthly, SiteTextChars: 60_000, Workers: scanWorkers,
		},
	)
	if err := svc.Recover(ctx); err != nil {
		return nil, err
	}
	return svc, nil
}

// Scans run scanWorkers at a time, each on another company's site; the rest wait their turn,
// however many there are.
const scanWorkers = 3

// directory hands the companies to the analysis context by value.
type directory struct {
	companies *companiesapp.Service
}

func (d directory) Resolve(ctx context.Context, user string, website *url.URL, name string) (string, error) {
	return d.companies.Resolve(ctx, user, website, name)
}

func (d directory) Companies(ctx context.Context) ([]analysisdomain.Company, error) {
	listed, err := d.companies.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]analysisdomain.Company, 0, len(listed))
	for _, l := range listed {
		out = append(out, analysisdomain.Company{
			ID: l.ID, Name: l.Name, Role: analysisdomain.Role(l.Role()), Website: l.URL, Archived: l.Archived,
			Researched: l.Source == companiesdomain.SourceResearch,
		})
	}
	return out, nil
}

// gateways hands the prompt-test context OpenRouter, reached with a stored OpenRouter key of the
// language-model settings.
type gateways struct {
	llm *llmapp.Service
}

func (g gateways) Check(ctx context.Context, id string) error {
	_, err := g.key(ctx, id)
	return err
}

func (g gateways) Gateway(ctx context.Context, id string) (prompttestapp.Gateway, error) {
	key, err := g.key(ctx, id)
	if err != nil {
		return nil, err
	}
	return openrouter.New(openrouter.URL, key, nil), nil
}

func (g gateways) key(ctx context.Context, id string) (string, error) {
	if id == "" {
		return "", prompttestdomain.ErrNoKey
	}
	key, err := g.llm.OpenRouterKey(ctx, id)
	switch {
	case errors.Is(err, llmdomain.ErrNotFound), errors.Is(err, llmdomain.ErrUnreadable), errors.Is(err, llmapp.ErrNotOpenRouter):
		return "", fmt.Errorf("%w (%w)", prompttestdomain.ErrNoKey, err)
	case err != nil:
		return "", err
	}
	return key, nil
}

// promptDirectory hands the prompt-test context the companies to match names to, and adds or
// removes a competitor candidate's company, by value.
type promptDirectory struct {
	companies *companiesapp.Service
}

func (d promptDirectory) Companies(ctx context.Context) ([]prompttestdomain.DirectoryCompany, error) {
	listed, err := d.companies.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]prompttestdomain.DirectoryCompany, 0, len(listed))
	for _, l := range listed {
		out = append(out, prompttestdomain.DirectoryCompany{ID: l.ID, Name: l.Name, OtherNames: l.OtherNames, Host: l.Host, Role: prompttestdomain.Role(l.Role())})
	}
	return out, nil
}

func (d promptDirectory) Add(ctx context.Context, user, name, website string) (string, error) {
	c, err := d.companies.Add(ctx, user, name, website)
	var known companiesdomain.AlreadyKnownError
	switch {
	case errors.As(err, &known):
		return "", fmt.Errorf("%w: %s", prompttestdomain.ErrCompanyKnown, known.Name)
	case errors.Is(err, companiesdomain.ErrInvalidName), errors.Is(err, companiesdomain.ErrInvalidWebsite):
		return "", fmt.Errorf("%w: %w", prompttestdomain.ErrInvalid, err)
	case err != nil:
		return "", err
	}
	return c.ID, nil
}

func (d promptDirectory) Remove(ctx context.Context, user, id string) error {
	err := d.companies.Delete(ctx, user, id)
	if errors.Is(err, companiesdomain.ErrNotFound) {
		return nil // removed already
	}
	if errors.Is(err, companiesapp.ErrScanActive) {
		return fmt.Errorf("%w: %w", prompttestdomain.ErrInvalid, err)
	}
	return err
}

// observations hands the review queue the phrases finished scans found outside the map, with the
// names of the companies scanned, by value.
func observations(ctx context.Context, repo *persistence.Repository, companies *companiesapp.Service) ([]terminologydomain.Observation, error) {
	findings, err := repo.UnmappedFindings(ctx)
	if err != nil {
		return nil, err
	}
	listed, err := companies.List(ctx)
	if err != nil {
		return nil, err
	}
	names := make(map[string]string, len(listed))
	for _, l := range listed {
		names[l.ID] = l.Name
	}
	out := make([]terminologydomain.Observation, 0, len(findings))
	for _, f := range findings {
		name := names[f.CompetitorID]
		if name == "" {
			name = f.CompetitorID
		}
		out = append(out, terminologydomain.Observation{
			ScanID: f.AnalysisID.String(), CompanyID: f.CompetitorID, CompanyName: name, ScanURL: f.URL,
			StartedAt: f.StartedAt, Phrase: f.Phrase, Language: f.Language, Score: f.Score, Pages: f.Pages,
			Tier: f.Tier, Family: f.Family, Closest: f.Closest,
		})
	}
	return out, nil
}

// scans hands the analyses of each company to the companies context by value.
type scans struct {
	repo *persistence.Repository
}

func (s scans) Stats(ctx context.Context) (map[string]companiesdomain.Stats, error) {
	stats, err := s.repo.CompetitorStats(ctx)
	if err != nil {
		return nil, err
	}
	products, err := s.repo.MatchedProducts(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[string]companiesdomain.Stats, len(stats))
	for id, st := range stats {
		last := scanOf(st.Last)
		out[id] = companiesdomain.Stats{Count: st.Count, Last: &last}
	}
	for _, p := range products {
		st := out[p.CompetitorID]
		st.Products = append(st.Products, companiesdomain.BeltName{Name: p.Product, Family: p.Family})
		out[p.CompetitorID] = st
	}
	return out, nil
}

func (s scans) ByCompany(ctx context.Context, id string, limit int) ([]companiesdomain.Scan, error) {
	list, err := s.repo.ListByCompetitor(ctx, id, limit)
	if err != nil {
		return nil, err
	}
	out := make([]companiesdomain.Scan, 0, len(list))
	for _, a := range list {
		out = append(out, scanOf(a))
	}
	return out, nil
}

func (s scans) Active(ctx context.Context, id string) (int, error) {
	return s.repo.CountActive(ctx, id)
}

func (s scans) DeleteByCompany(ctx context.Context, id string) error {
	return s.repo.DeleteByCompetitor(ctx, id)
}

func scanOf(a analysisdomain.Analysis) companiesdomain.Scan {
	return companiesdomain.Scan{
		ID: a.ID, URL: a.URL, Status: string(a.Status), Failure: a.Failure, MaxPages: a.MaxPages,
		PagesCrawled: a.PagesCrawled, CreatedAt: a.CreatedAt, FinishedAt: a.FinishedAt, Number: a.Number,
		MatchingModel: a.MatchingModel, Blocked: a.Blocked(),
	}
}

// sightings hands the terminology context the map terms the companies' newest scans found, by value.
func sightings(ctx context.Context, repo *persistence.Repository) ([]terminologydomain.Sighting, error) {
	seen, err := repo.Sightings(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]terminologydomain.Sighting, 0, len(seen))
	for _, s := range seen {
		out = append(out, terminologydomain.Sighting{Text: s.Term, Language: terminologydomain.Language(s.Language), CompanyID: s.CompetitorID})
	}
	return out, nil
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
