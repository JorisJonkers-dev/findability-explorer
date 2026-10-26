package app_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/app"
	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain"
)

var errBoom = errors.New("boom")

// repo is an in-memory Repository with the run lock and the lease of the real one; failOn names
// the methods that fail.
type repo struct {
	mu        sync.Mutex
	settings  *domain.Settings
	runs      []*storedRun
	answers   []domain.Answer
	decisions []domain.CandidateDecision
	failOn    map[string]bool
	// heartbeatsLeft, when not negative, is how many heartbeats succeed before the run is
	// taken over by another instance.
	heartbeatsLeft int
	beats          int
}

type storedRun struct {
	run       domain.Run
	owner     string
	heartbeat time.Time
}

func newRepo(failOn ...string) *repo {
	r := &repo{failOn: map[string]bool{}, heartbeatsLeft: -1}
	for _, f := range failOn {
		r.failOn[f] = true
	}
	return r
}

func (r *repo) err(op string) error {
	if r.failOn[op] {
		return errBoom
	}
	return nil
}

func (r *repo) Settings(context.Context) (domain.Settings, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.settings == nil {
		return domain.Settings{}, false, r.err("Settings")
	}
	return *r.settings, true, r.err("Settings")
}

func (r *repo) SaveSettings(_ context.Context, s domain.Settings, _ string, _ time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.err("SaveSettings"); err != nil {
		return err
	}
	r.settings = &s
	return nil
}

func (r *repo) running() *storedRun {
	for _, s := range r.runs {
		if s.run.Status == domain.RunRunning {
			return s
		}
	}
	return nil
}

func (r *repo) CreateRun(_ context.Context, run domain.Run, owner string, at time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.err("CreateRun"); err != nil {
		return err
	}
	if r.running() != nil {
		return domain.ErrRunActive
	}
	r.runs = append(r.runs, &storedRun{run: run, owner: owner, heartbeat: at})
	return nil
}

func (r *repo) ClaimDue(_ context.Context, now, next time.Time, run domain.Run, owner string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.err("ClaimDue"); err != nil {
		return false, err
	}
	if r.settings == nil || now.Before(r.settings.NextDue) || r.running() != nil {
		return false, nil
	}
	r.settings.NextDue = next
	r.runs = append(r.runs, &storedRun{run: run, owner: owner, heartbeat: now})
	return true, nil
}

func (r *repo) AdoptStale(_ context.Context, owner string, staleBefore, now time.Time) (*domain.Run, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.err("AdoptStale"); err != nil {
		return nil, err
	}
	s := r.running()
	if s == nil || !s.heartbeat.Before(staleBefore) {
		return nil, nil //nolint:nilnil // none to adopt
	}
	s.owner, s.heartbeat = owner, now
	run := s.run
	return &run, nil
}

func (r *repo) Heartbeat(_ context.Context, id uuid.UUID, owner string, at time.Time) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.err("Heartbeat"); err != nil {
		return false, err
	}
	r.beats++
	if r.heartbeatsLeft == 0 {
		return false, nil
	}
	if r.heartbeatsLeft > 0 {
		r.heartbeatsLeft--
	}
	for _, s := range r.runs {
		if s.run.ID == id && s.owner == owner {
			s.heartbeat = at
			return true, nil
		}
	}
	return false, nil
}

func (r *repo) FinishRun(_ context.Context, id uuid.UUID, status domain.RunStatus, failure string, at time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.err("FinishRun"); err != nil {
		return err
	}
	for _, s := range r.runs {
		if s.run.ID == id {
			s.run.Status, s.run.Failure, s.run.FinishedAt = status, failure, at
		}
	}
	return nil
}

func (r *repo) Run(_ context.Context, id uuid.UUID) (domain.Run, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, s := range r.runs {
		if s.run.ID == id {
			return s.run, r.err("Run")
		}
	}
	return domain.Run{}, domain.ErrNotFound
}

func (r *repo) Runs(_ context.Context, limit int) ([]domain.Run, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []domain.Run{}
	for i := len(r.runs) - 1; i >= 0 && len(out) < limit; i-- {
		out = append(out, r.runs[i].run)
	}
	return out, r.err("Runs")
}

func (r *repo) AddAnswer(_ context.Context, a domain.Answer) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.err("AddAnswer"); err != nil {
		return err
	}
	r.answers = append(r.answers, a)
	return nil
}

func (r *repo) SetExtraction(_ context.Context, id uuid.UUID, x domain.Extraction, _ time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.err("SetExtraction"); err != nil {
		return err
	}
	for i := range r.answers {
		if r.answers[i].ID == id {
			r.answers[i].Extraction = &x
		}
	}
	return nil
}

func (r *repo) Answers(_ context.Context, runID uuid.UUID, withText bool) ([]domain.Answer, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.err("Answers"); err != nil {
		return nil, err
	}
	var out []domain.Answer
	for _, a := range r.answers {
		if a.RunID == runID {
			if !withText {
				a.Text = ""
			}
			out = append(out, a)
		}
	}
	return out, nil
}

func (r *repo) SpentSince(context.Context, time.Time) (domain.Micros, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var total domain.Micros
	for _, a := range r.answers {
		total += a.Cost
		if a.Extraction != nil {
			total += a.Extraction.Cost
		}
	}
	return total, r.err("SpentSince")
}

func (r *repo) Sightings(context.Context) ([]domain.Sighting, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []domain.Sighting
	for _, a := range r.answers {
		if a.Extraction == nil {
			continue
		}
		for _, c := range a.Extraction.Companies {
			if c.CompetitorID == "" {
				out = append(out, domain.Sighting{Name: c.Name, URL: c.URL, Provider: a.Provider, Model: a.Model, At: a.At})
			}
		}
	}
	return out, r.err("Sightings")
}

func (r *repo) CandidateDecisions(context.Context) ([]domain.CandidateDecision, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.decisions), r.err("CandidateDecisions")
}

func (r *repo) Decide(_ context.Context, d domain.CandidateDecision) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.err("Decide"); err != nil {
		return err
	}
	r.decisions = append(r.decisions, d)
	return nil
}

func (r *repo) Undecide(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.err("Undecide"); err != nil {
		return err
	}
	r.decisions = slices.DeleteFunc(r.decisions, func(d domain.CandidateDecision) bool { return d.ID == id })
	return nil
}

// gateway answers every prompt: each model's answer names Twentebelt and, for Dutch prompts,
// Tribelt; a model in limited is always rate limited; extraction fails failExtract times first.
type gateway struct {
	mu          sync.Mutex
	catalogue   []domain.CatalogueModel
	modelsErr   error
	limited     map[string]bool
	askErr      error
	failExtract int
	asked       map[string]int
	extracted   int
	// slow is how long each call to Ask takes.
	slow time.Duration
}

func newGateway() *gateway {
	return &gateway{
		catalogue: []domain.CatalogueModel{
			{ID: "openai/gpt-oss-20b", PromptPrice: 0.0000001, CompletionPrice: 0.0000002, InputModalities: []string{"text"}, OutputModalities: []string{"text"}},
			{ID: "google/gemma:free", InputModalities: []string{"text"}, OutputModalities: []string{"text"}},
			{ID: "google/gemini-flash-lite", PromptPrice: 0.0000001, CompletionPrice: 0.0000004, InputModalities: []string{"text"}, OutputModalities: []string{"text"}},
		},
		limited: map[string]bool{}, asked: map[string]int{},
	}
}

func (g *gateway) Models(context.Context) ([]domain.CatalogueModel, error) {
	return g.catalogue, g.modelsErr
}

func (g *gateway) Ask(_ context.Context, model, prompt string, web bool) (domain.Reply, error) {
	time.Sleep(g.slow)
	g.mu.Lock()
	defer g.mu.Unlock()
	g.asked[model]++
	if g.limited[model] {
		return domain.Reply{Cost: 1}, domain.ErrRateLimited
	}
	if g.askErr != nil {
		return domain.Reply{}, g.askErr
	}
	text := "Twentebelt makes these belts."
	if strings.HasPrefix(prompt, "Welke") {
		text += " Tribelt in Haaksbergen too."
	}
	r := domain.Reply{Text: text, InputTokens: 20, OutputTokens: 30, Cost: 10}
	if web {
		r.Citations = []string{"https://www.tribelt.nl/"}
	}
	return r, nil
}

func (g *gateway) Extract(_ context.Context, model string, in app.ExtractInput) (domain.Extracted, domain.Reply, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if model != domain.ExtractorModel || in.FactSheet == "" || in.Prompt == "" {
		return domain.Extracted{}, domain.Reply{}, errors.New("the extractor got the wrong input")
	}
	if g.failExtract > 0 {
		g.failExtract--
		return domain.Extracted{}, domain.Reply{Cost: 1}, errBoom
	}
	g.extracted++
	x := domain.Extracted{Companies: []domain.Named{{Name: "Twentebelt"}, {Name: "Ashworth", URL: "https://ashworth.com"}}}
	if strings.Contains(in.Answer, "Tribelt") {
		x.Companies = append([]domain.Named{{Name: "Tribelt"}}, x.Companies...)
		x.NamesTribelt, x.Correctness, x.WrongClaims = true, domain.CorrectnessPartly, []string{"Tribelt is in Hengelo."}
	}
	return x, domain.Reply{InputTokens: 50, OutputTokens: 20, Cost: 2}, nil
}

// gateways hands out the one gateway for the credential "key".
type gateways struct {
	gw       *gateway
	checkErr error
}

func (g gateways) Check(_ context.Context, id string) error {
	if g.checkErr != nil {
		return g.checkErr
	}
	if id != "key" {
		return domain.ErrNoKey
	}
	return nil
}

func (g gateways) Gateway(_ context.Context, id string) (app.Gateway, error) {
	if id != "key" {
		return nil, domain.ErrNoKey
	}
	return g.gw, nil
}

// directory is the company list: Tribelt and Twentebelt, and the companies added since.
type directory struct {
	mu        sync.Mutex
	added     []domain.DirectoryCompany
	err       error
	addErr    error
	removeErr error
}

func (d *directory) Companies(context.Context) ([]domain.DirectoryCompany, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]domain.DirectoryCompany{
		{ID: "tribelt", Name: "Tribelt B.V.", Host: "tribelt.nl", Role: domain.RoleTribelt},
		{ID: "twentebelt", Name: "Twentebelt", Host: "twentebelt.com", Role: domain.RoleCompetitor},
	}, d.added...), d.err
}

func (d *directory) Add(_ context.Context, _, name, website string) (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.addErr != nil {
		return "", d.addErr
	}
	id := strings.ToLower(strings.Fields(name)[0])
	d.added = append(d.added, domain.DirectoryCompany{ID: id, Name: name, Host: strings.TrimPrefix(website, "https://"), Role: domain.RoleCompetitor})
	return id, nil
}

func (d *directory) Remove(_ context.Context, _, id string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.removeErr != nil {
		return d.removeErr
	}
	d.added = slices.DeleteFunc(d.added, func(c domain.DirectoryCompany) bool { return c.ID == id })
	return nil
}

// clock is a time a test moves on.
type clock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *clock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *clock) set(t time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = t
}

// sleeps counts the pauses a run took, and never waits.
type sleeps struct {
	mu    sync.Mutex
	total time.Duration
	n     int
	err   error
}

func (s *sleeps) Sleep(ctx context.Context, d time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.total += d
	s.n++
	if s.err != nil {
		return s.err
	}
	return ctx.Err()
}
