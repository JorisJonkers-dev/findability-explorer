package httpapi_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/app"
	"github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain"
)

// promptTests is a stand-in for the prompt-test use cases, so these tests exercise the contract.
type promptTests struct {
	err       error
	startErr  error
	updateErr error
	decideErr error
	change    app.Change
	actor     string
	filter    app.AnswerFilter
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
	set := domain.CurrentPromptSet()
	return []domain.Answer{
		{
			ID: uuid.New(), RunID: runID, Prompt: set.Prompts[0], PromptSetVersion: 1, Provider: "google", Model: "google/gemini", Mode: domain.ModeWeb,
			Reply: domain.Reply{Text: "Tribelt and Twentebelt.", Citations: []string{"https://www.tribelt.nl/"}, Cost: 1_500_000, InputTokens: 100, OutputTokens: 40}, At: answered,
			Extraction: &domain.Extraction{
				Companies:    []domain.Company{{Name: "Tribelt", CompetitorID: "tribelt", Position: 1}, {Name: "Ashworth", URL: "https://ashworth.com", Position: 2}},
				Addresses:    []string{"https://www.tribelt.nl/"},
				NamesTribelt: true, CitesTribelt: true, Correctness: domain.CorrectnessWrong, WrongClaims: []string{"Tribelt is in Hengelo."}, Cost: 500_000,
				InputTokens: 300, OutputTokens: 20,
			},
		},
		{ID: uuid.New(), RunID: runID, Prompt: set.Prompts[1], PromptSetVersion: 1, Provider: "openai", Model: "openai/gpt-oss-20b", Mode: domain.ModeKnowledge, Failure: "Rate limited.", At: answered},
	}
}

func (p *promptTests) Results(context.Context) (app.Results, error) {
	answers := promptAnswers()
	return app.Results{
		Runs:        []app.RunResult{{Run: promptRun(), Progress: domain.Progressed(answers, promptRun().Models), Cells: domain.Cells(answers)}},
		Mentions:    []app.NamedMention{{Mention: domain.Mentions(answers)[0], Name: "Tribelt B.V.", Role: domain.RoleTribelt}},
		WrongClaims: domain.WrongClaims(answers),
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

func view() app.View {
	s := domain.DefaultSettings(answered)
	s.CredentialID = "3f2a9c4e-7d1b-4c6a-9e8f-1a2b3c4d5e6f"
	s.FactSheet.UpdatedBy, s.FactSheet.UpdatedAt = "joris", answered
	return app.View{Settings: s, PromptSet: domain.CurrentPromptSet(), ExtractorModel: domain.ExtractorModel, SpentThisMonth: 2_500_000}
}

func (p *promptTests) Settings(context.Context) (app.View, error) { return view(), p.err }

func (p *promptTests) Update(_ context.Context, actor string, ch app.Change) (app.View, error) {
	p.actor, p.change = actor, ch
	return view(), p.updateErr
}

func candidate() domain.Candidate {
	return domain.Candidate{ID: "3a7bd3e2360a9f01", Name: "Ashworth", Answers: 12, Providers: []string{"openai"}, Addresses: []string{"https://ashworth.com"}, LastSeen: answered}
}

func (p *promptTests) Candidates(context.Context, domain.CandidateStatus) ([]domain.Candidate, map[domain.CandidateStatus]int, error) {
	return []domain.Candidate{candidate()}, map[domain.CandidateStatus]int{domain.CandidatePending: 1, domain.CandidateDismissed: 4}, p.err
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
	if r.code != http.StatusOK || len(runs) != 1 || r.body["promptSetVersion"].(float64) != domain.PromptSetVersion {
		t.Fatalf("results = %d %v", r.code, r.body)
	}
	run := runs[0].(map[string]any)
	if run["status"] != "stopped" || run["costUsd"].(float64) != 2 || run["answered"].(float64) != 1 || run["failed"].(float64) != 1 ||
		len(run["cells"].([]any)) != 1 || run["models"].([]any)[0].(map[string]any)["fallback"] != "google/gemini" || run["finishedAt"] == nil {
		t.Errorf("run = %v", run)
	}
	mention := r.body["mentions"].([]any)[0].(map[string]any)
	if mention["name"] != "Tribelt B.V." || mention["role"] != "tribelt" || mention["averagePosition"].(float64) != 1 {
		t.Errorf("mention = %v", mention)
	}
	if claim := r.body["wrongClaims"].([]any)[0].(map[string]any); claim["claim"] != "Tribelt is in Hengelo." || claim["correctness"] != "wrong" {
		t.Errorf("claim = %v", claim)
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
		r.body["credentialId"] != "3f2a9c4e-7d1b-4c6a-9e8f-1a2b3c4d5e6f" {
		t.Fatalf("settings = %d %v", r.code, r.body)
	}
	body := `{"credentialId":"3f2a9c4e-7d1b-4c6a-9e8f-1a2b3c4d5e6f","providers":["openai"],"monthlyBudgetUsd":12.5,"cadence":"weekly","factSheet":"Tribelt makes belts in Haaksbergen."}`
	if r := callAs(t, srv, http.MethodPatch, "/api/v1/admin/prompt-test", body, "admin"); r.code != http.StatusOK {
		t.Fatalf("update = %d %v", r.code, r.body)
	}
	ch := a.prompts().change
	if *ch.CredentialID != "3f2a9c4e-7d1b-4c6a-9e8f-1a2b3c4d5e6f" || *ch.Budget != 12_500_000 || *ch.Cadence != "weekly" || *ch.FactSheet == "" || ch.Providers[0] != "openai" {
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
