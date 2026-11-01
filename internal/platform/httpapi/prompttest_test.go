package httpapi_test

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/app"
	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain"
	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain/domaintest"
)

// promptTests is a stand-in for the prompt-test use cases, so these tests exercise the contract.
type promptTests struct {
	err        error
	retryErr   error
	retried    *domain.TaskKey
	reread     *domain.TaskKey
	planErr    error
	startErr   error
	updateErr  error
	decideErr  error
	decided    domain.CandidateStatus
	reviewing  bool
	checking   bool
	problem    app.ReviewProblem
	change     app.Change
	actor      string
	filter     app.AnswerFilter
	stopErr    error
	stopped    uuid.UUID
	historyErr error
	setsErr    error
	catalogErr error
	// workStops are the reviews and checks stopped; workStopErr makes their stops fail.
	workStops   []string
	workStopErr error
}

func (a *analyses) prompts() *promptTests {
	if a.promptTest == nil {
		a.promptTest = &promptTests{}
	}
	return a.promptTest
}

var (
	runID    = uuid.MustParse("5b0c1a9e-2d3f-4e5a-8b6c-7d8e9f0a1b2c")
	answered = time.Date(2026, 11, 1, 6, 10, 0, 0, time.UTC)
)

func promptRun() domain.Run {
	return domain.Run{
		ID: runID, PromptSetVersion: 1, FactSheetVersion: 2, Trigger: domain.TriggerSchedule, StartedBy: "schedule",
		StartedAt: answered.Add(-10 * time.Minute), FinishedAt: answered, Status: domain.RunStopped, Failure: "Budget.",
		Models: []domain.ModelChoice{{Provider: "google", Model: "google/gemma:free", Free: true, Fallback: "google/gemini"}, {Provider: "openai", Model: "openai/gpt-oss-20b"}},
		Tasks:  360,
	}
}

func promptAnswers() []domain.Answer {
	set := domaintest.Version2().PromptSet()
	return []domain.Answer{
		{
			ID: uuid.New(), RunID: runID, Prompt: set.Prompts[0], PromptSetVersion: 1, Provider: "google", Model: "google/gemini", Mode: domain.ModeWeb,
			Reply: domain.Reply{Text: "Tribelt and Twentebelt.", Citations: []string{"https://www.tribelt.nl/"}, Cost: 1_500_000, InputTokens: 100, OutputTokens: 40}, At: answered,
			Extraction: &domain.Extraction{
				Companies:    []domain.Company{{Name: "Tribelt", CompetitorID: "tribelt", Position: 1}, {Name: "Ashworth", URL: "https://ashworth.com", Position: 2}},
				Addresses:    []string{"https://www.tribelt.nl/"},
				NamesTribelt: true, CitesTribelt: true, Correctness: domain.CorrectnessWrong, WrongClaims: []string{"Tribelt is in Hengelo."}, Cost: 500_000,
				LeftOut: []string{"Tribelt makes eyelink belts to measure."}, InputTokens: 300, OutputTokens: 20,
			},
		},
		{ID: uuid.New(), RunID: runID, Prompt: set.Prompts[1], PromptSetVersion: 1, Provider: "openai", Model: "openai/gpt-oss-20b", Mode: domain.ModeKnowledge, Failure: "Rate limited.", At: answered},
	}
}

func (p *promptTests) Results(context.Context) (app.Results, error) {
	answers := promptAnswers()
	return app.Results{
		PromptSetVersion: 3,
		Runs:             []app.RunResult{{Run: promptRun(), Progress: domain.Progressed(answers, promptRun().Models), Cells: domain.Cells(answers)}},
		Mentions:         []app.NamedMention{{Mention: domain.Mentions(answers)[0], Name: "Tribelt B.V.", Role: domain.RoleTribelt, Share: 0.5}},
		WrongClaims:      domain.WrongClaims(answers),
	}, p.err
}

func (p *promptTests) Answers(_ context.Context, id uuid.UUID, filter app.AnswerFilter) (app.RunResult, []domain.Answer, error) {
	if id != runID {
		return app.RunResult{}, nil, domain.ErrNotFound
	}
	p.filter = filter
	answers := promptAnswers()
	return app.RunResult{Run: promptRun(), Progress: domain.Progressed(answers, promptRun().Models), Cells: domain.Cells(answers)}, answers, p.err
}

func (p *promptTests) Start(_ context.Context, actor string) (domain.Run, error) {
	p.actor = actor
	r := promptRun()
	r.Status, r.FinishedAt, r.Failure = domain.RunRunning, time.Time{}, ""
	return r, p.startErr
}

func (p *promptTests) Retry(_ context.Context, id uuid.UUID, task *domain.TaskKey) (app.Retry, error) {
	if id != runID {
		return app.Retry{}, domain.ErrNotFound
	}
	p.retried = task
	r := promptRun()
	r.Status, r.FinishedAt = domain.RunRunning, time.Time{}
	return app.Retry{Run: r, Tasks: 7, Estimate: 84_000}, p.retryErr
}

func (p *promptTests) Reread(_ context.Context, id uuid.UUID, task *domain.TaskKey) (app.Retry, error) {
	if id != runID {
		return app.Retry{}, domain.ErrNotFound
	}
	p.reread = task
	r := promptRun()
	r.Status, r.FinishedAt = domain.RunRunning, time.Time{}
	return app.Retry{Run: r, Tasks: 2, Estimate: 300}, p.retryErr
}

func (p *promptTests) RetryPlan(_ context.Context, id uuid.UUID) (app.Plan, error) {
	if id != runID {
		return app.Plan{}, domain.ErrNotFound
	}
	return app.Plan{
		Ask: 7, AskCost: 84_000, Unknown: 3, Read: 2, ReadCost: 300, Reading: 150,
		Costs: []domain.ModeCost{{Provider: "openai", Mode: domain.ModeWeb, Cost: 20_150}},
	}, p.planErr
}

func (p *promptTests) StopRun(_ context.Context, actor string, id uuid.UUID) error {
	if id != runID {
		return domain.ErrNotFound
	}
	p.actor, p.stopped = actor, id
	return p.stopErr
}

// partialID is the run the history gives as partial.
var partialID = uuid.MustParse("0c9d8e7f-6a5b-4c3d-9e2f-1a0b9c8d7e6f")

func (p *promptTests) PromptSets(context.Context) (app.PromptSets, error) {
	v1 := domain.PublishedSet{
		Version: 1, PublishedAt: time.Date(2026, 10, 26, 15, 4, 51, 0, time.UTC), PublishedBy: "import", Note: "The first.",
		Questions: []domain.Question{{
			ID: "eyelink-supplier-germany", Family: domain.FamilyEyelink, Intent: domain.IntentProduct,
			Scope: domaintest.Scope(domain.ScopeGermany), Text: "Welche Firmen liefern Drahtösengliederbänder?",
		}},
	}
	return app.PromptSets{Sets: []domain.PublishedSet{domaintest.Version2(), v1}, Scopes: domaintest.Scopes()}, p.setsErr
}

func (p *promptTests) History(context.Context) (app.HistoryView, error) {
	share := func(v float64) *float64 { return &v }
	v2 := domaintest.Version2()
	return app.HistoryView{
		Versions: domain.VersionsOf([]domain.PublishedSet{v2, {Version: 1, PublishedAt: v2.PublishedAt.Add(-time.Hour), Note: "The first.", Questions: make([]domain.Question, 90)}}),
		Runs: []domain.RunFigures{
			{RunID: partialID, StartedAt: answered.Add(-48 * time.Hour), Version: 1, Tasks: 1260, Answered: 62, Read: 47, Partial: true, All: share(0.4), Dutch: share(0.4)},
			{
				RunID: runID, StartedAt: answered, Version: 2, Tasks: 200, Answered: 196, Read: 194, Models: []string{"openai/gpt-oss-20b"},
				All: share(0.25), Dutch: share(0.3), German: share(0.2), English: share(0.1), Twente: share(0.5), Citation: share(0.35),
			},
		},
		FailureOf: map[uuid.UUID]string{partialID: "OpenRouter answered 401: User not found."},
	}, p.historyErr
}

func view() app.View {
	s := domain.DefaultSettings(answered)
	s.CredentialID = "3f2a9c4e-7d1b-4c6a-9e8f-1a2b3c4d5e6f"
	s.FactSheet.UpdatedBy, s.FactSheet.UpdatedAt = "joris", answered
	s.Models = []string{"google/gemma:free"}
	models := []domain.ModelChoice{{Provider: "google", Model: "google/gemma:free", Free: true, Fallback: "google/gemini"}}
	return app.View{
		Settings: s, PromptSet: domaintest.Version2().PromptSet(), ExtractorModel: domain.ExtractorModel, SpentThisMonth: 2_500_000,
		Models: models, KeyWorks: true,
		Prices: domain.PricesFromRun([]domain.Answer{{Provider: "google", Mode: domain.ModeKnowledge, Reply: domain.Reply{Cost: 1000}}}, models, 20, domain.CadenceWeekly, 10_000_000, nil),
	}
}

func (p *promptTests) Settings(context.Context) (app.View, error) { return view(), p.err }

func (p *promptTests) Update(_ context.Context, actor string, ch app.Change) (app.View, error) {
	p.actor, p.change = actor, ch
	return view(), p.updateErr
}

func (p *promptTests) Catalogue(context.Context) (app.CatalogueView, error) {
	listed := domain.CatalogueModel{ID: "openai/gpt-oss-20b", Name: "OpenAI: gpt-oss-20b", Created: answered, PromptPrice: 0.00000003, CompletionPrice: 0.00000014, InputModalities: []string{"text"}, OutputModalities: []string{"text"}}
	unlisted := domain.CatalogueModel{ID: "qwen/qwen3:free", Name: "Qwen3 (free)", InputModalities: []string{"text"}, OutputModalities: []string{"text"}}
	return app.CatalogueView{ReadAt: answered, Questions: 20, Entries: domain.Catalogue([]domain.CatalogueModel{unlisted, listed}, nil)}, p.catalogErr
}

func TestThePromptModelCatalogueIsForAdministrators(t *testing.T) {
	a := &analyses{}
	srv := serve(t, a)
	const path = "/api/v1/admin/prompt-test/catalogue"
	if r := callAs(t, srv, http.MethodGet, path, "", "tester"); r.code != http.StatusForbidden {
		t.Errorf("a viewer read the catalogue: %d", r.code)
	}
	r := callAs(t, srv, http.MethodGet, path, "", "admin")
	if r.code != http.StatusOK || r.body["readAt"] != "2026-11-01T06:10:00Z" || r.body["questions"].(float64) != 20 {
		t.Fatalf("catalogue = %d %v", r.code, r.body)
	}
	models := r.body["models"].([]any)
	first, second := models[0].(map[string]any), models[1].(map[string]any)
	if first["id"] != "openai/gpt-oss-20b" || first["recommended"] != true || first["provider"] != "openai" || first["listedAt"] != "2026-11-01T06:10:00Z" ||
		first["basis"] != "catalogue" || first["free"] != false || first["knowledgeUsd"].(float64) != 0.000135 || first["webUsd"].(float64) != 0.020135 {
		t.Errorf("first = %v", first)
	}
	if second["id"] != "qwen/qwen3:free" || second["recommended"] != false || second["free"] != true || second["listedAt"] != nil {
		t.Errorf("second = %v", second)
	}
	a.prompts().catalogErr = domain.ErrNoKey
	if r := callAs(t, srv, http.MethodGet, path, "", "admin"); r.code != http.StatusUnprocessableEntity {
		t.Errorf("no key: %d %v", r.code, r.body)
	}
	a.prompts().catalogErr = errors.New("down")
	if r := callAs(t, srv, http.MethodGet, path, "", "admin"); r.code != http.StatusInternalServerError {
		t.Errorf("failing: %d", r.code)
	}
}

func candidate() domain.Candidate {
	return domain.Candidate{
		ID: "3a7bd3e2360a9f01", Name: "Ashworth", Answers: 12, Providers: []string{"openai"}, Addresses: []string{"https://ashworth.com"}, LastSeen: answered,
		Sites: []string{"ashworth.com"}, Website: "https://ashworth.com/",
		Members: []domain.Member{
			{ID: "3a7bd3e2360a9f01", Name: "Ashworth", Answers: 10},
			{ID: "3a7bd3e2360a9f02", Name: "Ashworth Bros", Answers: 2, Joined: domain.JoinedByModel},
		},
		Verdict: &domain.CompanyVerdict{Competitor: true, Sure: true, Reason: "Makes metal belts.", Model: "anthropic: claude-haiku-5-5", At: answered},
	}
}

func (p *promptTests) Candidates(context.Context, domain.CandidateStatus) ([]domain.Candidate, app.CandidateCounts, error) {
	return []domain.Candidate{candidate()}, app.CandidateCounts{Pending: 1, Dismissed: 4, Unchecked: 1}, p.err
}

func (p *promptTests) DecideMany(_ context.Context, actor string, status domain.CandidateStatus, ids []string) ([]app.Decided, error) {
	p.actor, p.decided = actor, status
	return []app.Decided{
		{ID: ids[0], Name: "Ashworth", Outcome: app.OutcomeDone},
		{ID: "3a7bd3e2360a9f02", Outcome: app.OutcomeSkipped, Reason: "the answers gave no website for it"},
	}, p.decideErr
}

func (p *promptTests) Split(context.Context, string, string) error { return p.decideErr }

func (p *promptTests) StartReview(context.Context) bool {
	p.reviewing = true
	return true
}

func (p *promptTests) ReviewState() app.ReviewState {
	if p.problem == "" {
		return app.ReviewState{Running: p.reviewing, Problem: app.ReviewProblemNoKey}
	}
	return app.ReviewState{Running: p.reviewing, Problem: p.problem}
}

func (p *promptTests) ListChecks(context.Context) ([]domain.ListCheck, int, error) {
	return []domain.ListCheck{
		{CompanyID: "fenner-dunlop", Sure: true, Reason: "Makes rubber belting for quarries.", Model: "anthropic: claude-haiku-5-5", Version: domain.CheckVersion, At: answered},
		{CompanyID: "intralox", Competitor: true, Model: "anthropic: claude-haiku-5-5", Version: domain.CheckVersion, At: answered},
	}, 3, p.err
}

func (p *promptTests) StartListCheck(context.Context) bool {
	p.checking = true
	return true
}

func (p *promptTests) ListCheckState() app.ReviewState {
	if p.checking {
		return app.ReviewState{Running: true}
	}
	return app.ReviewState{Problem: app.ReviewProblemFails}
}

func (p *promptTests) AddCandidate(_ context.Context, actor, _, _, _ string) (domain.Candidate, error) {
	p.actor = actor
	c := candidate()
	c.Decision = &domain.CandidateDecision{Status: domain.CandidateAdded, CompetitorID: "ashworth", By: actor, At: answered}
	return c, p.decideErr
}

func (p *promptTests) DismissCandidate(_ context.Context, actor, _ string) (domain.Candidate, error) {
	c := candidate()
	c.Decision = &domain.CandidateDecision{Status: domain.CandidateDismissed, By: actor, At: answered}
	return c, p.decideErr
}

func (p *promptTests) UndoCandidate(context.Context, string, string) error { return p.decideErr }

func TestPromptTestResultsAreForEveryoneSignedIn(t *testing.T) {
	a := &analyses{}
	srv := serve(t, a)
	r := callAs(t, srv, http.MethodGet, "/api/v1/prompt-test", "", "tester")
	runs, _ := r.body["runs"].([]any)
	if r.code != http.StatusOK || len(runs) != 1 || r.body["promptSetVersion"].(float64) != 3 {
		t.Fatalf("results = %d %v", r.code, r.body)
	}
	run := runs[0].(map[string]any)
	if run["status"] != "stopped" || run["costUsd"].(float64) != 2 || run["answered"].(float64) != 1 || run["failed"].(float64) != 1 ||
		len(run["cells"].([]any)) != 1 || run["models"].([]any)[0].(map[string]any)["fallback"] != "google/gemini" || run["finishedAt"] == nil {
		t.Errorf("run = %v", run)
	}
	mention := r.body["mentions"].([]any)[0].(map[string]any)
	if mention["name"] != "Tribelt B.V." || mention["role"] != "tribelt" || mention["averagePosition"].(float64) != 1 || mention["share"].(float64) != 0.5 {
		t.Errorf("mention = %v", mention)
	}
	// Only the claims wrong about Tribelt are listed; the facts a response left out are not.
	if claims := r.body["wrongClaims"].([]any); len(claims) != 1 || claims[0].(map[string]any)["claim"] != "Tribelt is in Hengelo." ||
		claims[0].(map[string]any)["correctness"] != "wrong" {
		t.Errorf("claims = %v", claims)
	}
	a.prompts().err = errors.New("down")
	if r := callAs(t, srv, http.MethodGet, "/api/v1/prompt-test", "", "tester"); r.code != http.StatusInternalServerError {
		t.Errorf("failing results: %d", r.code)
	}
}

func TestPromptAnswersFilterAndRead(t *testing.T) {
	a := &analyses{}
	srv := serve(t, a)
	r := callAs(t, srv, http.MethodGet, "/api/v1/prompt-runs/"+runID.String()+"/answers", "", "tester")
	items, _ := r.body["items"].([]any)
	if r.code != http.StatusOK || len(items) != 2 || r.body["run"].(map[string]any)["id"] != runID.String() {
		t.Fatalf("answers = %d %v", r.code, r.body)
	}
	first := items[0].(map[string]any)
	x := first["extraction"].(map[string]any)
	if first["costUsd"].(float64) != 2 || first["inputTokens"].(float64) != 400 || first["outputTokens"].(float64) != 60 || first["kind"] != "application" || x["correctness"] != "wrong" ||
		x["companies"].([]any)[0].(map[string]any)["competitorId"] != "tribelt" || x["companies"].([]any)[1].(map[string]any)["url"] != "https://ashworth.com" {
		t.Errorf("answer = %v", first)
	}
	if second := items[1].(map[string]any); second["failure"] != "Rate limited." || second["extraction"] != nil {
		t.Errorf("failed answer = %v", second)
	}
	// One response shows what it got wrong about Tribelt apart from what it left out.
	if wrong, left := x["wrongClaims"].([]any), x["leftOut"].([]any); len(wrong) != 1 || wrong[0] != "Tribelt is in Hengelo." ||
		len(left) != 1 || left[0] != "Tribelt makes eyelink belts to measure." {
		t.Errorf("wrong %v, left out %v", x["wrongClaims"], x["leftOut"])
	}
	callAs(t, srv, http.MethodGet, "/api/v1/prompt-runs/"+runID.String()+"/answers?provider=google&scope=twente&family=eyelink&mode=knowledge", "", "tester")
	if f := a.prompts().filter; f != (app.AnswerFilter{Provider: "google", Scope: domain.ScopeTwente, Family: domain.FamilyEyelink, Mode: domain.ModeKnowledge}) {
		t.Errorf("filter = %+v", f)
	}
	if r := callAs(t, srv, http.MethodGet, "/api/v1/prompt-runs/"+uuid.NewString()+"/answers", "", "tester"); r.code != http.StatusNotFound {
		t.Errorf("no run: %d", r.code)
	}
	a.prompts().err = errors.New("down")
	if r := callAs(t, srv, http.MethodGet, "/api/v1/prompt-runs/"+runID.String()+"/answers", "", "tester"); r.code != http.StatusInternalServerError {
		t.Errorf("failing answers: %d", r.code)
	}
}

func TestStartingAPromptRunIsForAdministrators(t *testing.T) {
	a := &analyses{}
	srv := serve(t, a)
	if r := callAs(t, srv, http.MethodPost, "/api/v1/admin/prompt-runs", "", "tester"); r.code != http.StatusForbidden {
		t.Errorf("a viewer started a run: %d", r.code)
	}
	r := callAs(t, srv, http.MethodPost, "/api/v1/admin/prompt-runs", "", "admin")
	if r.code != http.StatusAccepted || r.body["status"] != "running" || r.body["finishedAt"] != nil || a.prompts().actor != "Admin" {
		t.Fatalf("start = %d %v", r.code, r.body)
	}
	for err, want := range map[error]int{
		domain.ErrRunActive:  http.StatusConflict,
		domain.ErrNoKey:      http.StatusUnprocessableEntity,
		domain.ErrNoModels:   http.StatusUnprocessableEntity,
		errors.New("broken"): http.StatusInternalServerError,
	} {
		a.prompts().startErr = err
		if r := callAs(t, srv, http.MethodPost, "/api/v1/admin/prompt-runs", "", "admin"); r.code != want {
			t.Errorf("%v: %d", err, r.code)
		}
	}
}

func TestPromptTestSettings(t *testing.T) {
	a := &analyses{}
	srv := serve(t, a)
	if r := callAs(t, srv, http.MethodGet, "/api/v1/admin/prompt-test", "", "tester"); r.code != http.StatusForbidden {
		t.Errorf("a viewer read the settings: %d", r.code)
	}
	r := callAs(t, srv, http.MethodGet, "/api/v1/admin/prompt-test", "", "admin")
	if r.code != http.StatusOK || r.body["monthlyBudgetUsd"].(float64) != 10 || r.body["spentThisMonthUsd"].(float64) != 2.5 ||
		len(r.body["promptSet"].(map[string]any)["prompts"].([]any)) != 20 || r.body["factSheet"].(map[string]any)["updatedBy"] != "joris" ||
		r.body["credentialId"] != "3f2a9c4e-7d1b-4c6a-9e8f-1a2b3c4d5e6f" || r.body["keyWorks"] != true ||
		r.body["models"].([]any)[0].(map[string]any)["fallback"] != "google/gemini" || r.body["modelList"].([]any)[0] != "google/gemma:free" {
		t.Fatalf("settings = %d %v", r.code, r.body)
	}
	prices := r.body["prices"].(map[string]any)
	provider := prices["providers"].([]any)[0].(map[string]any)
	if prices["basis"] != "run" || prices["runUsd"].(float64) != 0.02 || prices["overBudget"] != false || prices["budgetUsd"].(float64) != 10 ||
		provider["knowledgeUsd"].(float64) != 0.001 || provider["fallback"] != "google/gemini" {
		t.Errorf("prices = %v", prices)
	}
	body := `{"credentialId":"3f2a9c4e-7d1b-4c6a-9e8f-1a2b3c4d5e6f","providers":["openai"],"models":["openai/gpt-oss-20b"],"monthlyBudgetUsd":12.5,"cadence":"weekly","factSheet":"Tribelt makes belts in Haaksbergen."}`
	if r := callAs(t, srv, http.MethodPatch, "/api/v1/admin/prompt-test", body, "admin"); r.code != http.StatusOK {
		t.Fatalf("update = %d %v", r.code, r.body)
	}
	ch := a.prompts().change
	if *ch.CredentialID != "3f2a9c4e-7d1b-4c6a-9e8f-1a2b3c4d5e6f" || *ch.Budget != 12_500_000 || *ch.Cadence != "weekly" || *ch.FactSheet == "" || ch.Providers[0] != "openai" ||
		!slices.Equal(ch.Models, []string{"openai/gpt-oss-20b"}) {
		t.Errorf("change = %+v", ch)
	}
	a.prompts().updateErr = errors.Join(domain.ErrInvalid, errors.New("invalid: a month's budget is more than $0"))
	if r := callAs(t, srv, http.MethodPatch, "/api/v1/admin/prompt-test", `{"cadence":"monthly"}`, "admin"); r.code != http.StatusUnprocessableEntity {
		t.Errorf("an invalid change: %d %v", r.code, r.body)
	}
	a.prompts().updateErr = errors.New("down")
	if r := callAs(t, srv, http.MethodPatch, "/api/v1/admin/prompt-test", `{"cadence":"monthly"}`, "admin"); r.code != http.StatusInternalServerError {
		t.Errorf("a failing change: %d", r.code)
	}
	a.prompts().err = errors.New("down")
	if r := callAs(t, srv, http.MethodGet, "/api/v1/admin/prompt-test", "", "admin"); r.code != http.StatusInternalServerError {
		t.Errorf("failing settings: %d", r.code)
	}
}

func TestCompetitorCandidates(t *testing.T) {
	a := &analyses{}
	srv := serve(t, a)
	base := "/api/v1/admin/competitor-candidates"
	if r := callAs(t, srv, http.MethodGet, base, "", "tester"); r.code != http.StatusForbidden {
		t.Errorf("a viewer listed candidates: %d", r.code)
	}
	r := callAs(t, srv, http.MethodGet, base+"?status=pending", "", "admin")
	if r.code != http.StatusOK || len(r.body["items"].([]any)) != 1 || r.body["counts"].(map[string]any)["dismissed"].(float64) != 4 {
		t.Fatalf("candidates = %d %v", r.code, r.body)
	}
	id := "3a7bd3e2360a9f01"
	r = callAs(t, srv, http.MethodPost, base+"/"+id+"/add", `{"name":"Ashworth","url":"https://www.ashworth.com"}`, "admin")
	if r.code != http.StatusOK || r.body["status"] != "added" || r.body["decision"].(map[string]any)["competitorId"] != "ashworth" {
		t.Fatalf("add = %d %v", r.code, r.body)
	}
	if r := callAs(t, srv, http.MethodPost, base+"/"+id+"/dismiss", "", "admin"); r.code != http.StatusOK || r.body["status"] != "dismissed" {
		t.Fatalf("dismiss = %d %v", r.code, r.body)
	}
	if r := callAs(t, srv, http.MethodDelete, base+"/"+id+"/decision", "", "admin"); r.code != http.StatusNoContent {
		t.Fatalf("undo = %d", r.code)
	}
	cases := []struct {
		err                error
		add, dismiss, undo int
	}{
		{domain.ErrNotFound, 404, 404, 404},
		{app.ErrDecided, 409, 409, 409},
		{domain.ErrCompanyKnown, 409, 500, 500},
		{domain.ErrInvalid, 422, 500, 409},
		{errors.New("down"), 500, 500, 500},
	}
	for _, c := range cases {
		a.prompts().decideErr = c.err
		if r := callAs(t, srv, http.MethodPost, base+"/"+id+"/add", `{"name":"A","url":"https://a.test"}`, "admin"); r.code != c.add {
			t.Errorf("add with %v: %d", c.err, r.code)
		}
		if r := callAs(t, srv, http.MethodPost, base+"/"+id+"/dismiss", "", "admin"); r.code != c.dismiss {
			t.Errorf("dismiss with %v: %d", c.err, r.code)
		}
		if r := callAs(t, srv, http.MethodDelete, base+"/"+id+"/decision", "", "admin"); r.code != c.undo {
			t.Errorf("undo with %v: %d", c.err, r.code)
		}
	}
	a.prompts().err = errors.New("down")
	if r := callAs(t, srv, http.MethodGet, base, "", "admin"); r.code != http.StatusInternalServerError {
		t.Errorf("failing list: %d", r.code)
	}
}

func TestCandidatesFoldReviewAndAreDecidedTogether(t *testing.T) {
	a := &analyses{}
	srv := serve(t, a)
	base := "/api/v1/admin/competitor-candidates"
	r := callAs(t, srv, http.MethodGet, base, "", "admin")
	item := r.body["items"].([]any)[0].(map[string]any)
	names := item["names"].([]any)
	if r.code != http.StatusOK || item["website"] != "https://ashworth.com/" || item["sites"].([]any)[0] != "ashworth.com" || len(names) != 2 ||
		names[0].(map[string]any)["joined"] != "lead" || names[1].(map[string]any)["joined"] != "model" ||
		item["check"].(map[string]any)["reason"] != "Makes metal belts." || r.body["counts"].(map[string]any)["unchecked"] != float64(1) ||
		r.body["review"].(map[string]any)["problem"] != "noKey" {
		t.Fatalf("list = %d %v", r.code, r.body)
	}
	a.prompts().problem = app.ReviewProblemBudget
	if r := callAs(t, srv, http.MethodGet, base, "", "admin"); r.body["review"].(map[string]any)["problem"] != "budget" {
		t.Errorf("a spent budget = %v", r.body["review"])
	}
	a.prompts().problem = ""
	for _, path := range []string{base + "/review", base + "/decisions", base + "/3a7bd3e2360a9f02/split"} {
		if r := callAs(t, srv, http.MethodPost, path, `{"action":"add","ids":["3a7bd3e2360a9f01"]}`, "tester"); r.code != http.StatusForbidden {
			t.Errorf("a viewer on %s: %d", path, r.code)
		}
	}
	if r := callAs(t, srv, http.MethodPost, base+"/review", "", "admin"); r.code != http.StatusAccepted || r.body["running"] != true {
		t.Errorf("review = %d %v", r.code, r.body)
	}
	r = callAs(t, srv, http.MethodPost, base+"/decisions", `{"action":"add","ids":["3a7bd3e2360a9f01","3a7bd3e2360a9f02"]}`, "admin")
	items := r.body["items"].([]any)
	if r.code != http.StatusOK || a.prompts().decided != domain.CandidateAdded || len(items) != 2 ||
		items[0].(map[string]any)["outcome"] != "done" || items[1].(map[string]any)["reason"] != "The answers gave no website for it." {
		t.Fatalf("decide = %d %v", r.code, r.body)
	}
	if r := callAs(t, srv, http.MethodPost, base+"/decisions", `{"action":"dismiss","ids":["3a7bd3e2360a9f01"]}`, "admin"); r.code != http.StatusOK || a.prompts().decided != domain.CandidateDismissed {
		t.Errorf("dismiss several = %d", r.code)
	}
	if r := callAs(t, srv, http.MethodPost, base+"/decisions", `{"action":"keep","ids":["3a7bd3e2360a9f01"]}`, "admin"); r.code != http.StatusBadRequest {
		t.Errorf("an unknown action = %d", r.code)
	}
	if r := callAs(t, srv, http.MethodPost, base+"/3a7bd3e2360a9f02/split", "", "admin"); r.code != http.StatusNoContent {
		t.Errorf("split = %d", r.code)
	}
	for err, want := range map[error]int{domain.ErrNotFound: 404, app.ErrNotMerged: 409, errors.New("down"): 500} {
		a.prompts().decideErr = err
		if r := callAs(t, srv, http.MethodPost, base+"/3a7bd3e2360a9f02/split", "", "admin"); r.code != want {
			t.Errorf("split with %v = %d, want %d", err, r.code, want)
		}
	}
	if r := callAs(t, srv, http.MethodPost, base+"/decisions", `{"action":"dismiss","ids":["3a7bd3e2360a9f01"]}`, "admin"); r.code != http.StatusInternalServerError {
		t.Errorf("decide failing = %d", r.code)
	}
}

func TestAskingAgainIsForAdministrators(t *testing.T) {
	a := &analyses{}
	srv := serve(t, a)
	path := "/api/v1/admin/prompt-runs/" + runID.String() + "/retry"
	if r := callAs(t, srv, http.MethodPost, path, `{}`, "tester"); r.code != http.StatusForbidden {
		t.Errorf("a viewer asked again: %d", r.code)
	}
	r := callAs(t, srv, http.MethodPost, path, `{}`, "admin")
	if r.code != http.StatusAccepted || r.body["tasks"].(float64) != 7 || r.body["estimatedCostUsd"].(float64) != 0.084 ||
		r.body["run"].(map[string]any)["status"] != "running" || a.prompts().retried != nil {
		t.Fatalf("retry all = %d %v", r.code, r.body)
	}
	r = callAs(t, srv, http.MethodPost, path, `{"promptId":"eyelink-application-twente","provider":"openai","mode":"web"}`, "admin")
	if want := (domain.TaskKey{PromptID: "eyelink-application-twente", Provider: "openai", Mode: domain.ModeWeb}); r.code != http.StatusAccepted || *a.prompts().retried != want {
		t.Errorf("retry one = %d, %+v", r.code, a.prompts().retried)
	}
	if r := callAs(t, srv, http.MethodPost, path, `{"provider":"openai"}`, "admin"); r.code != http.StatusUnprocessableEntity {
		t.Errorf("half a task = %d", r.code)
	}
	if r := callAs(t, srv, http.MethodPost, "/api/v1/admin/prompt-runs/"+uuid.NewString()+"/retry", `{}`, "admin"); r.code != http.StatusNotFound {
		t.Errorf("no such run = %d", r.code)
	}
	for err, want := range map[error]int{
		domain.ErrRunActive:     http.StatusConflict,
		domain.ErrBudgetReached: http.StatusConflict,
		app.ErrNothingFailed:    http.StatusUnprocessableEntity,
		errors.New("broken"):    http.StatusInternalServerError,
	} {
		a.prompts().retryErr = err
		if r := callAs(t, srv, http.MethodPost, path, `{}`, "admin"); r.code != want {
			t.Errorf("%v: %d", err, r.code)
		}
	}
}

func TestHaikuChecksTheCompetitorList(t *testing.T) {
	a := &analyses{}
	srv := serve(t, a)
	base := "/api/v1/admin/competitor-checks"
	for method, want := range map[string]int{http.MethodGet: http.StatusForbidden, http.MethodPost: http.StatusForbidden} {
		if r := callAs(t, srv, method, base, "", "tester"); r.code != want {
			t.Errorf("a viewer's %s: %d", method, r.code)
		}
	}
	r := callAs(t, srv, http.MethodGet, base, "", "admin")
	items, _ := r.body["items"].([]any)
	if r.code != http.StatusOK || len(items) != 2 || r.body["unchecked"] != float64(3) || r.body["review"].(map[string]any)["problem"] != "failed" {
		t.Fatalf("list = %d %v", r.code, r.body)
	}
	first := items[0].(map[string]any)
	if first["competitorId"] != "fenner-dunlop" || first["sure"] != true || first["competitor"] != false ||
		first["reason"] != "Makes rubber belting for quarries." || first["at"] == nil {
		t.Errorf("first check = %v", first)
	}
	if r := callAs(t, srv, http.MethodPost, base, "", "admin"); r.code != http.StatusAccepted || r.body["running"] != true || !a.prompts().checking {
		t.Errorf("start = %d %v", r.code, r.body)
	}
	a.prompts().err = errors.New("down")
	if r := callAs(t, srv, http.MethodGet, base, "", "admin"); r.code != http.StatusInternalServerError {
		t.Errorf("failing list: %d", r.code)
	}
}

func TestReadingAgainIsForAdministrators(t *testing.T) {
	a := &analyses{}
	srv := serve(t, a)
	path := "/api/v1/admin/prompt-runs/" + runID.String() + "/reread"
	if r := callAs(t, srv, http.MethodPost, path, `{}`, "tester"); r.code != http.StatusForbidden {
		t.Errorf("a viewer read again: %d", r.code)
	}
	r := callAs(t, srv, http.MethodPost, path, `{}`, "admin")
	if r.code != http.StatusAccepted || r.body["tasks"].(float64) != 2 || r.body["estimatedCostUsd"].(float64) != 0.0003 ||
		r.body["run"].(map[string]any)["status"] != "running" || a.prompts().reread != nil {
		t.Fatalf("read all again = %d %v", r.code, r.body)
	}
	r = callAs(t, srv, http.MethodPost, path, `{"promptId":"eyelink-application-twente","provider":"openai","mode":"web"}`, "admin")
	if want := (domain.TaskKey{PromptID: "eyelink-application-twente", Provider: "openai", Mode: domain.ModeWeb}); r.code != http.StatusAccepted || *a.prompts().reread != want {
		t.Errorf("read one again = %d, %+v", r.code, a.prompts().reread)
	}
	if r := callAs(t, srv, http.MethodPost, path, `{"mode":"web"}`, "admin"); r.code != http.StatusUnprocessableEntity {
		t.Errorf("half an answer = %d", r.code)
	}
	if r := callAs(t, srv, http.MethodPost, "/api/v1/admin/prompt-runs/"+uuid.NewString()+"/reread", `{}`, "admin"); r.code != http.StatusNotFound {
		t.Errorf("no such run = %d", r.code)
	}
	for err, want := range map[error]int{
		domain.ErrRunActive:     http.StatusConflict,
		domain.ErrBudgetReached: http.StatusConflict,
		app.ErrNothingUnread:    http.StatusUnprocessableEntity,
		errors.New("broken"):    http.StatusInternalServerError,
	} {
		a.prompts().retryErr = err
		if r := callAs(t, srv, http.MethodPost, path, `{}`, "admin"); r.code != want {
			t.Errorf("%v: %d", err, r.code)
		}
	}
}

func TestThePlanOfAskingAgainIsForAdministrators(t *testing.T) {
	a := &analyses{}
	srv := serve(t, a)
	path := "/api/v1/admin/prompt-runs/" + runID.String() + "/retry-plan"
	if r := callAs(t, srv, http.MethodGet, path, "", "tester"); r.code != http.StatusForbidden {
		t.Errorf("a viewer read the plan: %d", r.code)
	}
	r := callAs(t, srv, http.MethodGet, path, "", "admin")
	costs, _ := r.body["answerCosts"].([]any)
	if r.code != http.StatusOK || r.body["askTasks"].(float64) != 7 || r.body["askCostUsd"].(float64) != 0.084 || r.body["unknownTasks"].(float64) != 3 ||
		r.body["readTasks"].(float64) != 2 || r.body["readCostUsd"].(float64) != 0.0003 || r.body["readingCostUsd"].(float64) != 0.00015 || len(costs) != 1 ||
		costs[0].(map[string]any)["mode"] != "web" || costs[0].(map[string]any)["costUsd"].(float64) != 0.02015 {
		t.Fatalf("plan = %d %v", r.code, r.body)
	}
	if r := callAs(t, srv, http.MethodGet, "/api/v1/admin/prompt-runs/"+uuid.NewString()+"/retry-plan", "", "admin"); r.code != http.StatusNotFound {
		t.Errorf("no such run = %d", r.code)
	}
	a.prompts().planErr = errors.New("broken")
	if r := callAs(t, srv, http.MethodGet, path, "", "admin"); r.code != http.StatusInternalServerError {
		t.Errorf("a failing plan = %d", r.code)
	}
}

func TestAnAdministratorStopsARunningRun(t *testing.T) {
	a := &analyses{}
	srv := serve(t, a)
	path := "/api/v1/admin/prompt-runs/" + runID.String() + "/stop"
	if r := callAs(t, srv, http.MethodPost, path, "", "tester"); r.code != http.StatusForbidden || a.prompts().stopped != uuid.Nil {
		t.Errorf("a viewer stopped the run: %d", r.code)
	}
	if r := callAs(t, srv, http.MethodPost, path, "", "admin"); r.code != http.StatusNoContent || a.prompts().stopped != runID || a.prompts().actor == "" {
		t.Errorf("stop = %d, stopped %v by %q", r.code, a.prompts().stopped, a.prompts().actor)
	}
	r := callAs(t, srv, http.MethodPost, "/api/v1/admin/prompt-runs/"+uuid.NewString()+"/stop", "", "admin")
	if r.code != http.StatusNotFound || r.body["detail"] != "No Prompt run with this id is running." {
		t.Errorf("stopping a run that is not running = %d %v", r.code, r.body)
	}
	a.prompts().stopErr = errors.New("broken")
	if r := callAs(t, srv, http.MethodPost, path, "", "admin"); r.code != http.StatusInternalServerError {
		t.Errorf("a failing stop = %d", r.code)
	}
}

func TestEverySignedInAccountReadsThePromptTestHistory(t *testing.T) {
	a := &analyses{}
	srv := serve(t, a)
	if r := call(t, srv, http.MethodGet, "/api/v1/prompt-test/history", "", false); r.code != http.StatusUnauthorized {
		t.Errorf("without a session = %d", r.code)
	}
	r := call(t, srv, http.MethodGet, "/api/v1/prompt-test/history", "", true)
	versions, _ := r.body["versions"].([]any)
	runs, _ := r.body["runs"].([]any)
	if r.code != http.StatusOK || len(versions) != 2 || len(runs) != 2 {
		t.Fatalf("history = %d %v", r.code, r.body)
	}
	v := versions[1].(map[string]any)
	if v["version"].(float64) != 2 || v["publishedAt"] != "2026-10-27T06:35:33Z" || v["questions"].(float64) != 20 || v["note"] == "" {
		t.Errorf("version 2 = %v", v)
	}
	partial, full := runs[0].(map[string]any), runs[1].(map[string]any)
	if full["models"].([]any)[0] != "openai/gpt-oss-20b" || partial["models"] != nil {
		t.Errorf("models = %v, %v", full["models"], partial["models"])
	}
	if partial["runId"] != partialID.String() || partial["partial"] != true || partial["promptSetVersion"].(float64) != 1 || partial["tasks"].(float64) != 1260 ||
		partial["answered"].(float64) != 62 || partial["read"].(float64) != 47 || partial["failure"] != "OpenRouter answered 401: User not found." ||
		partial["all"].(float64) != 0.4 || partial["dutch"].(float64) != 0.4 {
		t.Errorf("partial run = %v", partial)
	}
	for _, k := range []string{"german", "english", "twente", "citation"} {
		if _, ok := partial[k]; ok {
			t.Errorf("the partial run has a %s share it had nothing to measure on", k)
		}
	}
	if full["runId"] != runID.String() || full["partial"] != false || full["citation"].(float64) != 0.35 || full["twente"].(float64) != 0.5 ||
		full["english"].(float64) != 0.1 || full["german"].(float64) != 0.2 || full["startedAt"] != "2026-11-01T06:10:00Z" {
		t.Errorf("full run = %v", full)
	}
	if _, ok := full["failure"]; ok {
		t.Errorf("a full run has a failure: %v", full)
	}
	a.prompts().historyErr = errors.New("broken")
	if r := call(t, srv, http.MethodGet, "/api/v1/prompt-test/history", "", true); r.code != http.StatusInternalServerError {
		t.Errorf("a failing history = %d", r.code)
	}
}

func TestEverySignedInAccountReadsThePublishedPromptSets(t *testing.T) {
	a := &analyses{}
	srv := serve(t, a)
	if r := call(t, srv, http.MethodGet, "/api/v1/prompt-test/prompt-sets", "", false); r.code != http.StatusUnauthorized {
		t.Errorf("without a session = %d", r.code)
	}
	r := call(t, srv, http.MethodGet, "/api/v1/prompt-test/prompt-sets", "", true)
	versions, _ := r.body["versions"].([]any)
	scopes, _ := r.body["scopes"].([]any)
	if r.code != http.StatusOK || len(versions) != 2 || len(scopes) != 6 {
		t.Fatalf("prompt sets = %d %v", r.code, r.body)
	}
	v2, v1 := versions[0].(map[string]any), versions[1].(map[string]any)
	questions, _ := v2["questions"].([]any)
	if v2["version"].(float64) != 2 || v2["publishedAt"] != "2026-10-27T06:35:33Z" || v2["note"] == "" || len(questions) != 20 {
		t.Fatalf("version 2 = %v", v2)
	}
	twente := questions[0].(map[string]any)
	if twente["id"] != "eyelink-application-twente" || twente["family"] != "eyelink" || twente["intent"] != "application" ||
		twente["scope"] != "twente" || twente["scopeName"] != "Twente (Dutch)" || twente["language"] != "nl" || twente["region"] != "Twente" ||
		!strings.HasPrefix(twente["text"].(string), "We bouwen") || !strings.HasPrefix(twente["english"].(string), "We are building") {
		t.Errorf("a Twente question = %v", twente)
	}
	english := questions[3].(map[string]any)
	if _, ok := english["english"]; ok || english["scope"] != "english" {
		t.Errorf("an English question = %v, with no translation", english)
	}
	if _, ok := english["region"]; ok {
		t.Errorf("a question with no region = %v", english)
	}
	product := v1["questions"].([]any)[0].(map[string]any)
	if product["intent"] != "product" || product["scope"] != "germany" || product["region"] != "Germany" || product["language"] != "de" {
		t.Errorf("a version 1 supplier question = %v", product)
	}
	first, dutch := scopes[0].(map[string]any), scopes[3].(map[string]any)
	if first["id"] != "twente" || first["name"] != "Twente (Dutch)" || first["language"] != "nl" || first["region"] != "Twente" {
		t.Errorf("the Twente scope = %v", first)
	}
	if _, ok := dutch["region"]; ok || dutch["id"] != "dutch" {
		t.Errorf("the Dutch scope = %v, with no region", dutch)
	}
	a.prompts().setsErr = errors.New("broken")
	if r := call(t, srv, http.MethodGet, "/api/v1/prompt-test/prompt-sets", "", true); r.code != http.StatusInternalServerError {
		t.Errorf("failing prompt sets = %d", r.code)
	}
}
