package app

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain"
)

// Config tunes a Service; the zero value of each field picks the production default.
type Config struct {
	// Owner names this app instance as the holder of the runs it runs. Default a new UUID.
	Owner string
	// Pace is the pause before each call to a free model, within OpenRouter's limit of about
	// 20 free requests a minute. Default 3s.
	Pace time.Duration
	// RetryWait is the pause before the first retry of a rate-limited call; each next retry
	// waits once more as long. Default 20s.
	RetryWait time.Duration
	// Retries is how often a rate-limited call is tried again. Default 3.
	Retries int
	// Stale is how long a running run's holder may go unheard before another instance takes it
	// over. Default 2 minutes.
	Stale time.Duration
	// Tick is how often Loop checks the schedule and for runs to take over. Default 1 minute.
	Tick time.Duration
}

func (c Config) withDefaults() Config {
	if c.Owner == "" {
		c.Owner = uuid.NewString()
	}
	if c.Pace == 0 {
		c.Pace = 3 * time.Second
	}
	if c.RetryWait == 0 {
		c.RetryWait = 20 * time.Second
	}
	if c.Retries == 0 {
		c.Retries = 3
	}
	if c.Stale == 0 {
		c.Stale = 2 * time.Minute
	}
	if c.Tick == 0 {
		c.Tick = time.Minute
	}
	return c
}

// Service runs the prompt-test use cases.
type Service struct {
	repo      Repository
	gateways  Gateways
	directory Directory
	now       func() time.Time
	sleep     func(context.Context, time.Duration) error
	// spawn runs a Prompt run in the background; tests run it in place.
	spawn  func(func())
	onFail func(uuid.UUID, error)
	cfg    Config
	// review is the language model's review of the candidates; its reviewers are nil until WithReviewers.
	review review
	// extractors read the answers instead of OpenRouter when they hand one out; nil until WithExtractors.
	extractors Extractors
}

// Option changes how a Service waits and runs work in the background; tests use them.
type Option func(*Service)

// WithSleep replaces the pause between calls.
func WithSleep(sleep func(context.Context, time.Duration) error) Option {
	return func(s *Service) { s.sleep = sleep }
}

// WithReviewers lets a language model review the competitor candidates, reading their home pages
// through pages; onError hears why a review failed, which only the logs can report.
func WithReviewers(r Reviewers, pages HomePages, onError func(error)) Option {
	return func(s *Service) { s.review.reviewers, s.review.pages, s.review.onError = r, pages, onError }
}

// WithExtractors lets Claude Haiku on an Anthropic key read the answers while one works; without
// one, OpenRouter reads them with domain.ExtractorModel.
func WithExtractors(e Extractors) Option {
	return func(s *Service) { s.extractors = e }
}

// WithSpawn replaces how a run is started in the background.
func WithSpawn(spawn func(func())) Option { return func(s *Service) { s.spawn = spawn } }

// New returns a Service. onFail hears about a run that could not be recorded, which only the logs
// can report.
func New(repo Repository, gateways Gateways, directory Directory, now func() time.Time, cfg Config,
	onFail func(uuid.UUID, error), opts ...Option,
) *Service {
	s := &Service{
		repo: repo, gateways: gateways, directory: directory, now: now, cfg: cfg.withDefaults(), onFail: onFail,
		sleep: sleep, spawn: func(f func()) { go f() },
	}
	for _, o := range opts {
		o(s)
	}
	return s
}

func sleep(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// View is everything the settings show.
type View struct {
	Settings       domain.Settings
	PromptSet      domain.PromptSet
	ExtractorModel string
	SpentThisMonth domain.Micros
	// Models are the models the latest run asked, each provider's with its fallback; none before a run.
	Models []domain.ModelChoice
	// KeyWorks is true when the chosen OpenRouter key can be read and used.
	KeyWorks bool
	Prices   domain.Prices
}

// Settings returns the settings, the prompt set, this month's spend, the models the latest run
// asked, whether the key works, and what runs cost.
func (s *Service) Settings(ctx context.Context) (View, error) {
	settings, err := s.settings(ctx)
	if err != nil {
		return View{}, err
	}
	spent, err := s.repo.SpentSince(ctx, domain.MonthStart(s.now()))
	if err != nil {
		return View{}, fmt.Errorf("this month's spend: %w", err)
	}
	reader, err := s.extractorModel(ctx)
	if err != nil {
		return View{}, err
	}
	v := View{
		Settings: settings, PromptSet: domain.CurrentPromptSet(), ExtractorModel: reader, SpentThisMonth: spent,
		Models: []domain.ModelChoice{}, KeyWorks: settings.CredentialID != "" && s.gateways.Check(ctx, settings.CredentialID) == nil,
	}
	if err := s.price(ctx, &v); err != nil {
		return View{}, err
	}
	return v, nil
}

// price fills in the latest run's models and the price overview for the providers set now: each
// one the latest run asked by what its answers cost, and the others, before any run or added
// since, by OpenRouter's price list when the key works.
func (s *Service) price(ctx context.Context, v *View) error {
	questions := len(v.PromptSet.Prompts)
	cadence, budget, providers := v.Settings.Cadence, v.Settings.MonthlyBudget, v.Settings.Providers
	runs, err := s.repo.Runs(ctx, 1)
	if err != nil {
		return fmt.Errorf("the latest run: %w", err)
	}
	v.Prices = domain.NoPrices(cadence, budget)
	missing := providers
	if len(runs) > 0 {
		v.Models = runs[0].Models
		answers, err := s.repo.Answers(ctx, runs[0].ID, false)
		if err != nil {
			return fmt.Errorf("the latest run's answers: %w", err)
		}
		if len(answers) > 0 {
			var kept []domain.ModelChoice
			kept, missing = domain.Asked(runs[0].Models, providers)
			v.Prices = domain.PricesFromRun(answers, kept, questions, cadence, budget)
		}
	}
	if len(missing) == 0 || !v.KeyWorks {
		return nil
	}
	catalogue := s.catalogue(ctx, v.Settings.CredentialID)
	if catalogue == nil {
		return nil
	}
	estimate := domain.PricesFromCatalogue(catalogue, domain.ChooseModels(catalogue, missing), questions, cadence, budget)
	if v.Prices.Basis == domain.BasisRun {
		v.Prices = v.Prices.With(estimate.Providers, questions, cadence)
		return nil
	}
	v.Prices = estimate
	return nil
}

// catalogue is OpenRouter's price list, or nil when it cannot be had: the overview then has
// nothing to go by for the providers it would price.
func (s *Service) catalogue(ctx context.Context, credentialID string) []domain.CatalogueModel {
	gw, err := s.gateways.Gateway(ctx, credentialID)
	if err != nil {
		return nil
	}
	models, err := gw.Models(ctx)
	if err != nil {
		return nil
	}
	return models
}

// extractor is what reads the answers, and its model: Claude Haiku while an Anthropic key works,
// else OpenRouter through gw with domain.ExtractorModel.
func (s *Service) extractor(ctx context.Context, gw Gateway) (Extractor, string, error) {
	if s.extractors != nil {
		x, model, err := s.extractors.Current(ctx)
		if err != nil {
			return nil, "", fmt.Errorf("the extractor: %w", err)
		}
		if x != nil {
			return x, model, nil
		}
	}
	return openRouterExtractor{gw: gw}, domain.ExtractorModel, nil
}

// extractorModel is the model that reads the answers now, as the settings show it.
func (s *Service) extractorModel(ctx context.Context) (string, error) {
	_, model, err := s.extractor(ctx, nil) // the gateway is not asked
	return model, err
}

// openRouterExtractor reads the answers on OpenRouter with domain.ExtractorModel.
type openRouterExtractor struct {
	gw Gateway
}

func (o openRouterExtractor) Extract(ctx context.Context, in ExtractInput) (domain.Extracted, domain.Reply, error) {
	return o.gw.Extract(ctx, domain.ExtractorModel, in)
}

func (s *Service) settings(ctx context.Context) (domain.Settings, error) {
	settings, ok, err := s.repo.Settings(ctx)
	if err != nil {
		return domain.Settings{}, fmt.Errorf("read the settings: %w", err)
	}
	if !ok {
		return domain.DefaultSettings(s.now()), nil
	}
	return settings, nil
}

// Change is what an administrator changes in the settings; a nil field stays as it is.
type Change struct {
	CredentialID *string
	Providers    []string
	Budget       *domain.Micros
	Cadence      *string
	FactSheet    *string
}

// Update applies ch to the settings. A new cadence starts counting from now; a new fact sheet is
// a new version.
func (s *Service) Update(ctx context.Context, actor string, ch Change) (View, error) {
	settings, err := s.settings(ctx)
	if err != nil {
		return View{}, err
	}
	if settings, err = s.apply(ctx, settings, ch, actor); err != nil {
		return View{}, err
	}
	if err := s.repo.SaveSettings(ctx, settings, actor, s.now()); err != nil {
		return View{}, fmt.Errorf("save the settings: %w", err)
	}
	return s.Settings(ctx)
}

func (s *Service) apply(ctx context.Context, settings domain.Settings, ch Change, actor string) (domain.Settings, error) {
	settings, err := s.applyRuns(ctx, settings, ch)
	if err != nil {
		return domain.Settings{}, err
	}
	if ch.Cadence != nil {
		if settings.Cadence, err = domain.CheckCadence(*ch.Cadence); err != nil {
			return domain.Settings{}, err
		}
		settings.NextDue = domain.NextDue(s.now(), settings.Cadence)
	}
	if ch.FactSheet != nil {
		text, err := domain.CheckFactSheet(*ch.FactSheet)
		if err != nil {
			return domain.Settings{}, err
		}
		if text != settings.FactSheet.Text {
			settings.FactSheet = domain.FactSheet{Text: text, Version: settings.FactSheet.Version + 1, UpdatedBy: actor, UpdatedAt: s.now()}
		}
	}
	return settings, nil
}

// applyRuns applies what runs ask with: the key, the providers and the budget.
func (s *Service) applyRuns(ctx context.Context, settings domain.Settings, ch Change) (domain.Settings, error) {
	var err error
	if ch.CredentialID != nil {
		if err := s.gateways.Check(ctx, *ch.CredentialID); err != nil {
			return domain.Settings{}, err
		}
		settings.CredentialID = *ch.CredentialID
	}
	if ch.Providers != nil {
		if settings.Providers, err = domain.CheckProviders(ch.Providers); err != nil {
			return domain.Settings{}, err
		}
	}
	if ch.Budget != nil {
		if settings.MonthlyBudget, err = domain.CheckBudget(*ch.Budget); err != nil {
			return domain.Settings{}, err
		}
	}
	return settings, nil
}

// IsInputError reports whether err is the caller's to fix (422) rather than the service's.
func IsInputError(err error) bool {
	return errors.Is(err, domain.ErrInvalid) || errors.Is(err, domain.ErrNoKey) || errors.Is(err, domain.ErrNoModels)
}

// Start starts a Prompt run now, unless one is running already (domain.ErrRunActive). It needs
// an OpenRouter key and at least one provider with a model on OpenRouter's list. The run goes on
// in the background.
func (s *Service) Start(ctx context.Context, actor string) (domain.Run, error) {
	run, err := s.prepare(ctx, domain.TriggerManual, actor)
	if err != nil {
		return domain.Run{}, err
	}
	if err := s.repo.CreateRun(ctx, run, s.cfg.Owner, s.now()); err != nil {
		if errors.Is(err, domain.ErrRunActive) {
			return domain.Run{}, err
		}
		return domain.Run{}, fmt.Errorf("store the run: %w", err)
	}
	s.background(run)
	return run, nil
}

// prepare makes the run the settings ask for: the cheapest model of each provider now.
func (s *Service) prepare(ctx context.Context, trigger domain.Trigger, actor string) (domain.Run, error) {
	settings, err := s.settings(ctx)
	if err != nil {
		return domain.Run{}, err
	}
	gw, err := s.gateways.Gateway(ctx, settings.CredentialID)
	if err != nil {
		return domain.Run{}, err
	}
	catalogue, err := gw.Models(ctx)
	if err != nil {
		return domain.Run{}, fmt.Errorf("OpenRouter's price list: %w", err)
	}
	models := domain.ChooseModels(catalogue, settings.Providers)
	if len(models) == 0 {
		return domain.Run{}, domain.ErrNoModels
	}
	set := domain.CurrentPromptSet()
	return domain.Run{
		ID: uuid.New(), PromptSetVersion: set.Version, FactSheetVersion: settings.FactSheet.Version, FactSheet: settings.FactSheet.Text,
		Trigger: trigger, StartedBy: actor, StartedAt: s.now(), Status: domain.RunRunning, Models: models,
		Tasks: len(domain.Tasks(set, models)),
	}, nil
}

// background runs r in the background, and reports a failure to record it.
func (s *Service) background(r domain.Run) {
	s.spawn(func() {
		if err := s.Execute(context.WithoutCancel(context.Background()), r); err != nil {
			s.onFail(r.ID, err)
		}
	})
}

// Tick takes over a run whose holder went quiet (a restart, or the other copy of the app during
// a rollout), and starts the scheduled run when it is due. Only one instance starts a due run:
// the repository claims it in one transaction.
func (s *Service) Tick(ctx context.Context) error {
	now := s.now()
	stale, err := s.repo.AdoptStale(ctx, s.cfg.Owner, now.Add(-s.cfg.Stale), now)
	if err != nil {
		return fmt.Errorf("take over a stale run: %w", err)
	}
	if stale != nil {
		s.background(*stale)
	}
	settings, err := s.settings(ctx)
	if err != nil {
		return err
	}
	// Without a key a due run waits for one; it starts at the first tick after a key is chosen.
	if now.Before(settings.NextDue) || settings.CredentialID == "" {
		return nil
	}
	run, err := s.prepare(ctx, domain.TriggerSchedule, "schedule")
	if err != nil {
		return fmt.Errorf("prepare the scheduled run: %w", err)
	}
	claimed, err := s.repo.ClaimDue(ctx, now, domain.NextDue(now, settings.Cadence), run, s.cfg.Owner)
	if err != nil {
		return fmt.Errorf("claim the scheduled run: %w", err)
	}
	if claimed {
		s.background(run)
	}
	return nil
}

// Loop calls Tick every Config.Tick until ctx ends; onError hears about each failed tick.
func (s *Service) Loop(ctx context.Context, onError func(error)) {
	for {
		if err := s.Tick(ctx); err != nil && ctx.Err() == nil {
			onError(err)
		}
		if s.sleep(ctx, s.cfg.Tick) != nil {
			return
		}
	}
}
