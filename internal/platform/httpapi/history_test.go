package httpapi_test

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"

	analysisapp "github.com/JorisJonkers-dev/findability-explorer/internal/analysis/app"
	"github.com/JorisJonkers-dev/findability-explorer/internal/analysis/domain"
	companiesdomain "github.com/JorisJonkers-dev/findability-explorer/internal/companies/domain"
	terminologydomain "github.com/JorisJonkers-dev/findability-explorer/internal/terminology/domain"
)

func (r *companyRepo) Update(_ context.Context, c companiesdomain.Company) error {
	for i := range r.added {
		if r.added[i].ID == c.ID {
			r.added[i] = c
		}
	}
	return nil
}

func (r *companyRepo) Delete(_ context.Context, id string) error {
	for i := range r.added {
		if r.added[i].ID == id {
			r.added = append(r.added[:i], r.added[i+1:]...)
			return nil
		}
	}
	return companiesdomain.ErrNotFound
}

func (r *companyRepo) Overrides(context.Context) ([]companiesdomain.Override, error) {
	return r.overrides, nil
}

func (r *companyRepo) SaveOverride(_ context.Context, o companiesdomain.Override) error {
	r.overrides = []companiesdomain.Override{o}
	return nil
}

// Active reports a running scan of Märtens.
func (s companyScans) Active(_ context.Context, id string) (int, error) {
	if id == "martens-transportbander" {
		return 1, nil
	}
	return 0, nil
}

func (s companyScans) DeleteByCompany(context.Context, string) error { return nil }

func (a *analyses) Rescan(_ context.Context, _ string, pick analysisapp.Pick, maxPages int) ([]domain.Analysis, []analysisapp.Skipped, error) {
	a.pick = pick
	if a.rescanErr != nil {
		return nil, nil, a.rescanErr
	}
	queued := []domain.Analysis{{
		ID: uuid.New(), URL: "https://www.twentebelt.com/", CompetitorID: "twentebelt", Number: 3, MaxPages: maxPages,
		Status: domain.StatusQueued, CreatedAt: time.Now().UTC(), MatchingModel: "test: m",
	}}
	return queued, []analysisapp.Skipped{{CompetitorID: "martens", Name: "Märtens", Reason: "A scan of it is still queued or running."}}, nil
}

func (a *analyses) Profile(context.Context, string) (domain.Profile, error) {
	return a.profile, a.profileErr
}

func (a *analyses) Compare(_ context.Context, _ string, _, _ uuid.UUID) (domain.Comparison, error) {
	if a.compareErr != nil {
		return domain.Comparison{}, a.compareErr
	}
	older, newer := done(), done()
	older.Number, newer.Number = 1, 2
	groups := domain.Compare(
		domain.Analysis{Keywords: []domain.Keyword{
			{Phrase: "gone", Language: "en", Score: 3, Pages: 1, Classification: domain.Classification{Tier: "broad"}},
			{Phrase: "eyelink belt", Language: "en", Score: 3, Pages: 1, Classification: domain.Classification{Tier: "product", Family: "eyelink"}},
		}},
		domain.Analysis{Keywords: []domain.Keyword{
			{Phrase: "new", Language: "de", Score: 9, Pages: 2, Classification: domain.Classification{Tier: "spec", Family: "spec_all"}},
			{Phrase: "eyelink belt", Language: "en", Score: 30, Pages: 5, Classification: domain.Classification{Tier: "product", Family: "eyelink"}},
		}},
	)
	return domain.Comparison{From: older, To: newer, Groups: groups, Readiness: a.readinessChanges}, nil
}

// reviewStore is an in-memory terminology Reviews.
type reviewStore struct{ decisions []terminologydomain.Decision }

func (r *reviewStore) Decisions(context.Context) ([]terminologydomain.Decision, error) {
	return r.decisions, nil
}

func (r *reviewStore) Decide(_ context.Context, d terminologydomain.Decision) error {
	r.decisions = append(r.decisions, d)
	return nil
}

func (r *reviewStore) Undo(_ context.Context, id string) (bool, error) {
	for i, d := range r.decisions {
		if d.CandidateID == id {
			r.decisions = append(r.decisions[:i], r.decisions[i+1:]...)
			return true, nil
		}
	}
	return false, nil
}

func observations() []terminologydomain.Observation {
	return []terminologydomain.Observation{
		{
			ScanID: "s1", CompanyID: "twentebelt", CompanyName: "Twentebelt", ScanURL: "https://www.twentebelt.com/", StartedAt: time.Now().UTC(),
			Phrase: "spiral freezer", Language: "en", Score: 12, Pages: 3, Tier: "application", Family: "tri_flex_turn_curve", Closest: "spiral belt",
		},
		{
			ScanID: "s1", CompanyID: "twentebelt", CompanyName: "Twentebelt", StartedAt: time.Now().UTC(),
			Phrase: "belt news", Language: "en", Score: 1, Pages: 1, Tier: "broad",
		},
	}
}

func TestUpdateAndDeleteCompetitor(t *testing.T) {
	srv := serve(t, &analyses{})
	if r := call(t, srv, http.MethodPatch, "/api/v1/competitors/twentebelt", `{"archived":true}`, true); r.code != http.StatusForbidden {
		t.Fatalf("a non-admin archiving: status %d", r.code)
	}
	r := callAs(t, srv, http.MethodPatch, "/api/v1/competitors/twentebelt", `{"name":"Twentebelt B.V.","url":"https://www.twentebelt.nl","archived":true}`, "admin")
	if r.code != http.StatusOK || r.body["archived"] != true || r.body["host"] != "twentebelt.nl" || r.body["name"] != "Twentebelt B.V." {
		t.Fatalf("archive = %d %v", r.code, r.body)
	}
	added := callAs(t, srv, http.MethodPost, "/api/v1/competitors", `{"name":"Example","url":"https://example.com"}`, "admin")
	if added.code != http.StatusCreated {
		t.Fatalf("add = %d", added.code)
	}
	// An added company is archived and keeps its scans; restoring it keeps it on the list.
	r = callAs(t, srv, http.MethodPatch, "/api/v1/competitors/example", `{"archived":true}`, "admin")
	if r.code != http.StatusOK || r.body["archived"] != true || r.body["archivedBy"] != "admin-1" || r.body["archivedAt"] == nil || r.body["keptBy"] != nil {
		t.Fatalf("archive an added company = %d %v", r.code, r.body)
	}
	r = callAs(t, srv, http.MethodPatch, "/api/v1/competitors/example", `{"archived":false}`, "admin")
	if r.code != http.StatusOK || r.body["archived"] != false || r.body["archivedBy"] != nil || r.body["keptBy"] != "admin-1" {
		t.Fatalf("restore an added company = %d %v", r.code, r.body)
	}
	if r := callAs(t, srv, http.MethodPatch, "/api/v1/competitors/example", `{"kept":false}`, "admin"); r.code != http.StatusOK || r.body["keptBy"] != nil {
		t.Fatalf("unkeep = %d %v", r.code, r.body)
	}
	if r := callAs(t, srv, http.MethodPatch, "/api/v1/competitors/example", `{"kept":true}`, "admin"); r.code != http.StatusOK || r.body["keptBy"] != "admin-1" {
		t.Fatalf("keep = %d %v", r.code, r.body)
	}
	for body, want := range map[string]int{
		`{"url":"https://www.twentebelt.nl"}`: http.StatusConflict,
		`{"name":" "}`:                        http.StatusUnprocessableEntity,
		`{}`:                                  http.StatusBadRequest,
	} {
		if r := callAs(t, srv, http.MethodPatch, "/api/v1/competitors/example", body, "admin"); r.code != want {
			t.Errorf("patch %s: status %d, want %d (%v)", body, r.code, want, r.body)
		}
	}
	if r := callAs(t, srv, http.MethodPatch, "/api/v1/competitors/nobody", `{"name":"X"}`, "admin"); r.code != http.StatusNotFound {
		t.Errorf("patch nobody: status %d", r.code)
	}
	for path, want := range map[string]int{
		"/api/v1/competitors/example":                 http.StatusNoContent,
		"/api/v1/competitors/nobody":                  http.StatusNotFound,
		"/api/v1/competitors/martens-transportbander": http.StatusConflict,
	} {
		if r := callAs(t, srv, http.MethodDelete, path, "", "admin"); r.code != want {
			t.Errorf("delete %s: status %d, want %d", path, r.code, want)
		}
	}
}

func TestStartRescans(t *testing.T) {
	a := &analyses{}
	srv := serve(t, a)
	if r := call(t, srv, http.MethodPost, "/api/v1/admin/rescans", `{"all":true}`, true); r.code != http.StatusForbidden {
		t.Fatalf("a non-admin re-scanning: status %d", r.code)
	}
	r := callAs(t, srv, http.MethodPost, "/api/v1/admin/rescans", `{"competitorIds":["twentebelt","martens"],"maxPages":80}`, "admin")
	queued, _ := r.body["queued"].([]any)
	skipped, _ := r.body["skipped"].([]any)
	if r.code != http.StatusAccepted || len(queued) != 1 || len(skipped) != 1 || queued[0].(map[string]any)["number"].(float64) != 3 ||
		queued[0].(map[string]any)["matchingModel"] != "test: m" {
		t.Fatalf("rescan = %d %v", r.code, r.body)
	}
	if !slices.Equal(a.pick.IDs, []string{"twentebelt", "martens"}) || a.pick.All || a.pick.Unscanned {
		t.Fatalf("pick = %+v", a.pick)
	}
	if r := callAs(t, srv, http.MethodPost, "/api/v1/admin/rescans", `{"unscanned":true}`, "admin"); r.code != http.StatusAccepted || !a.pick.Unscanned {
		t.Fatalf("unscanned: %d, pick %+v", r.code, a.pick)
	}
	for err, want := range map[error]int{
		analysisapp.ErrInvalidRescan: http.StatusUnprocessableEntity,
		domain.ErrInvalidMaxPages:    http.StatusUnprocessableEntity,
		errors.New("database down"):  http.StatusInternalServerError,
	} {
		a.rescanErr = err
		if r := callAs(t, srv, http.MethodPost, "/api/v1/admin/rescans", `{"all":true}`, "admin"); r.code != want {
			t.Errorf("%v: status %d, want %d", err, r.code, want)
		}
	}
}

func TestProfileAndCompare(t *testing.T) {
	scan := done()
	scan.Number, scan.Keywords = 2, nil
	a := &analyses{profile: domain.BuildProfile(done(), 2, nil)}
	a.profile.Scan.Number = 2
	srv := serve(t, a)
	r := call(t, srv, http.MethodGet, "/api/v1/competitors/twentebelt/profile", "", true)
	groups, _ := r.body["groups"].([]any)
	if r.code != http.StatusOK || len(groups) != 7 || r.body["scan"].(map[string]any)["number"].(float64) != 2 || r.body["finishedScans"].(float64) != 2 {
		t.Fatalf("profile = %d %v", r.code, r.body)
	}
	broad := groups[5].(map[string]any)["terms"].([]any)
	if len(broad) != 2 || broad[0].(map[string]any)["closestTerm"] != "metal conveyor belt(s)" || broad[0].(map[string]any)["family"] != "all" {
		t.Fatalf("broad terms = %v", broad)
	}
	a.profile = domain.Profile{Languages: []string{}, Groups: []domain.GroupTerms{}}
	if r := call(t, srv, http.MethodGet, "/api/v1/competitors/twentebelt/profile", "", true); r.code != http.StatusOK || r.body["scan"] != nil {
		t.Fatalf("empty profile = %d %v", r.code, r.body)
	}
	for err, want := range map[error]int{analysisapp.ErrUnknownCompany: http.StatusNotFound, errors.New("down"): http.StatusInternalServerError} {
		a.profileErr = err
		if r := call(t, srv, http.MethodGet, "/api/v1/competitors/twentebelt/profile", "", true); r.code != want {
			t.Errorf("profile %v: status %d", err, r.code)
		}
	}

	c := call(t, srv, http.MethodGet, "/api/v1/competitors/twentebelt/scans/compare?from="+uuid.NewString()+"&to="+uuid.NewString(), "", true)
	cg, _ := c.body["groups"].([]any)
	if c.code != http.StatusOK || len(cg) != 7 || c.body["from"].(map[string]any)["number"].(float64) != 1 {
		t.Fatalf("compare = %d %v", c.code, c.body)
	}
	moved := cg[0].(map[string]any)["moved"].([]any)[0].(map[string]any)
	if moved["fromScore"].(float64) != 3 || moved["toPages"].(float64) != 5 || moved["family"] != "eyelink" {
		t.Fatalf("moved = %v", moved)
	}
	if added := cg[6].(map[string]any)["added"].([]any)[0].(map[string]any); added["fromScore"] != nil || added["toScore"].(float64) != 9 {
		t.Fatalf("added = %v", added)
	}
	for err, want := range map[error]int{
		analysisapp.ErrUnknownCompany: http.StatusNotFound,
		domain.ErrNotComparable:       http.StatusUnprocessableEntity,
		errors.New("down"):            http.StatusInternalServerError,
	} {
		a.compareErr = err
		if r := call(t, srv, http.MethodGet, "/api/v1/competitors/twentebelt/scans/compare", "", true); r.code != want {
			t.Errorf("compare %v: status %d", err, r.code)
		}
	}
}

func TestTermReviewQueue(t *testing.T) {
	srv := serve(t, &analyses{})
	if r := call(t, srv, http.MethodGet, "/api/v1/admin/term-candidates", "", true); r.code != http.StatusForbidden {
		t.Fatalf("a non-admin listing candidates: status %d", r.code)
	}
	list := callAs(t, srv, http.MethodGet, "/api/v1/admin/term-candidates", "", "admin")
	items, _ := list.body["items"].([]any)
	if list.code != http.StatusOK || len(items) != 2 || list.body["counts"].(map[string]any)["pending"].(float64) != 2 {
		t.Fatalf("list = %d %v", list.code, list.body)
	}
	first := items[0].(map[string]any)
	id := first["id"].(string)
	if first["phrase"] != "spiral freezer" || first["suggestedFamily"] != "tri_flex_turn_curve" || first["closestTerm"] != "spiral belt" {
		t.Fatalf("first = %v", first)
	}
	accept := `{"term":"spiral freezer belt","language":"en","family":"tri_flex_turn_curve","tier":"application","usedBy":["Twentebelt"]}`
	if r := call(t, srv, http.MethodPost, "/api/v1/admin/term-candidates/"+id+"/accept", accept, true); r.code != http.StatusForbidden {
		t.Fatalf("a non-admin accepting: status %d", r.code)
	}
	r := callAs(t, srv, http.MethodPost, "/api/v1/admin/term-candidates/"+id+"/accept", accept, "admin")
	decision, _ := r.body["decision"].(map[string]any)
	if r.code != http.StatusOK || r.body["status"] != "accepted" || decision["by"] != "Admin" || decision["term"].(map[string]any)["origin"] != "review" {
		t.Fatalf("accept = %d %v", r.code, r.body)
	}
	if found := call(t, srv, http.MethodGet, "/api/v1/terms?q=spiral%20freezer", "", true); found.body["total"].(float64) != 1 {
		t.Fatalf("the accepted term is not in the map: %v", found.body)
	}
	other := items[1].(map[string]any)["id"].(string)
	for path, want := range map[string]int{
		"/api/v1/admin/term-candidates/" + id + "/accept":       http.StatusConflict,
		"/api/v1/admin/term-candidates/0000000000000000/accept": http.StatusNotFound,
	} {
		if r := callAs(t, srv, http.MethodPost, path, accept, "admin"); r.code != want || r.body["detail"] == nil {
			t.Errorf("%s: status %d, want %d", path, r.code, want)
		}
	}
	if r := callAs(t, srv, http.MethodPost, "/api/v1/admin/term-candidates/"+other+"/accept",
		`{"term":"spiral freezer belt","language":"en","family":"other","tier":"broad","usedBy":[]}`, "admin"); r.code != http.StatusConflict {
		t.Errorf("a term the map has: status %d", r.code)
	}
	if r := callAs(t, srv, http.MethodPost, "/api/v1/admin/term-candidates/"+other+"/accept",
		`{"term":"  ","language":"en","family":"other","tier":"broad","usedBy":[]}`, "admin"); r.code != http.StatusUnprocessableEntity {
		t.Errorf("an empty term: status %d", r.code)
	}
	steps := []struct {
		method, path string
		want         int
	}{
		{http.MethodPost, "/api/v1/admin/term-candidates/" + other + "/reject", http.StatusOK},
		{http.MethodPost, "/api/v1/admin/term-candidates/" + other + "/reject", http.StatusConflict},
		{http.MethodPost, "/api/v1/admin/term-candidates/0000000000000000/reject", http.StatusNotFound},
		{http.MethodGet, "/api/v1/admin/term-candidates?status=rejected", http.StatusOK},
		{http.MethodDelete, "/api/v1/admin/term-candidates/" + other + "/decision", http.StatusOK},
		{http.MethodDelete, "/api/v1/admin/term-candidates/" + other + "/decision", http.StatusConflict},
		{http.MethodDelete, "/api/v1/admin/term-candidates/0000000000000000/decision", http.StatusNotFound},
	}
	for _, step := range steps {
		r := callAs(t, srv, step.method, step.path, "", "admin")
		if r.code != step.want {
			t.Errorf("%s %s: status %d, want %d", step.method, step.path, r.code, step.want)
		}
		if items, ok := r.body["items"].([]any); ok && len(items) != 1 {
			t.Errorf("rejected = %v", r.body)
		}
	}
}

func TestTheLanguageModelsReview(t *testing.T) {
	srv := serve(t, &analyses{})
	list := callAs(t, srv, http.MethodGet, "/api/v1/admin/term-candidates", "", "admin")
	items, _ := list.body["items"].([]any)
	if list.code != http.StatusOK || len(items) != 2 {
		t.Fatalf("list = %d %v", list.code, list.body)
	}
	if review := list.body["review"].(map[string]any); review["running"] != false || review["waiting"] != float64(1) || review["problem"] != nil {
		t.Errorf("review = %v", review)
	}
	if v := items[0].(map[string]any)["verdict"]; v != nil {
		t.Errorf("a candidate the model has not seen has a verdict: %v", v)
	}
	v := items[1].(map[string]any)["verdict"].(map[string]any)
	if v["accept"] != true || v["sure"] != false || v["reason"] != "Could go either way." || v["model"] != "claude-haiku-5-5" ||
		v["term"].(map[string]any)["family"] != "all" {
		t.Errorf("verdict = %v", v)
	}
	if r := call(t, srv, http.MethodPost, "/api/v1/admin/term-candidates/review", "", true); r.code != http.StatusForbidden {
		t.Errorf("a viewer started a review: %d", r.code)
	}
	r := callAs(t, srv, http.MethodPost, "/api/v1/admin/term-candidates/review", "", "admin")
	if r.code != http.StatusAccepted || r.body["running"] != true || r.body["waiting"] != float64(1) {
		t.Errorf("start = %d %v", r.code, r.body)
	}
}
