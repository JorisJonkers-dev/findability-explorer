package httpapi_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"

	analysisapp "github.com/JorisJonkers-dev/findability-explorer/internal/analysis/app"
	"github.com/JorisJonkers-dev/findability-explorer/internal/analysis/domain"
)

func (a *analyses) StartReadinessCheck(context.Context) (analysisapp.ReadinessCheck, error) {
	if a.checkErr != nil {
		return analysisapp.ReadinessCheck{}, a.checkErr
	}
	a.check = analysisapp.ReadinessCheck{Running: true, Total: 40}
	return a.check, nil
}

func (a *analyses) ReadinessCheckState() analysisapp.ReadinessCheck { return a.check }

func TestStartReadinessCheck(t *testing.T) {
	a := &analyses{}
	srv := serve(t, a)
	if r := call(t, srv, http.MethodPost, "/api/v1/admin/readiness-checks", "", true); r.code != http.StatusForbidden {
		t.Fatalf("a non-admin checking: status %d", r.code)
	}
	r := callAs(t, srv, http.MethodPost, "/api/v1/admin/readiness-checks", "", "admin")
	if r.code != http.StatusAccepted || r.body["running"] != true || r.body["total"].(float64) != 40 {
		t.Fatalf("check = %d %v", r.code, r.body)
	}
	a.check = analysisapp.ReadinessCheck{Checked: 39, Failed: 1, Total: 40}
	table := call(t, srv, http.MethodGet, "/api/v1/ai-readiness", "", true)
	if c := table.body["check"].(map[string]any); c["running"] != false || c["checked"].(float64) != 39 || c["failed"].(float64) != 1 {
		t.Errorf("table's check = %v", c)
	}
	a.checkErr = errors.New("database down")
	if r := callAs(t, srv, http.MethodPost, "/api/v1/admin/readiness-checks", "", "admin"); r.code != http.StatusInternalServerError {
		t.Errorf("failing check: status %d", r.code)
	}
}

func (a *analyses) ReadinessTable(context.Context) ([]domain.ReadinessRow, error) {
	return a.readinessRows, a.readinessErr
}

func (a *analyses) ReadinessFiles(_ context.Context, id uuid.UUID) (domain.RawFiles, error) {
	raw, ok := a.rawFiles[id]
	if !ok {
		return domain.RawFiles{}, domain.ErrNoReadiness
	}
	return raw, a.readinessErr
}

func readiness() domain.Readiness {
	return domain.AssessReadiness(domain.ReadinessFiles{
		Robots: domain.File{Status: 200, ContentType: "text/plain", Body: "User-agent: GPTBot\nDisallow: /\n"},
		Home: domain.File{
			Status: 200, ContentType: "text/html", Body: `<meta name="generator" content="WordPress">`,
			Header: map[string]string{"cf-ray": "1"},
		},
	})
}

func TestAIReadinessTable(t *testing.T) {
	scan := done()
	rd := readiness()
	rd.CheckedAt = time.Date(2026, 10, 27, 18, 30, 0, 0, time.UTC)
	rd.Sitemap.Coverage = &domain.SitemapCoverage{Pages: 40, Listed: 34}
	scan.Readiness = &rd
	a := &analyses{readinessRows: []domain.ReadinessRow{
		{Company: domain.Company{ID: "tribelt", Name: "Tribelt B.V.", Role: domain.RoleTribelt}, Scan: &scan},
		{Company: domain.Company{ID: "twentebelt", Name: "Twentebelt", Role: domain.RoleCompetitor}},
	}}
	srv := serve(t, a)
	r := call(t, srv, http.MethodGet, "/api/v1/ai-readiness", "", true)
	items, _ := r.body["items"].([]any)
	if r.code != http.StatusOK || len(items) != 2 {
		t.Fatalf("table = %d %v", r.code, r.body)
	}
	tribelt := items[0].(map[string]any)
	got := tribelt["readiness"].(map[string]any)
	bots := got["bots"].([]any)
	gpt := bots[5].(map[string]any)
	if tribelt["role"] != "tribelt" || got["robots"] != "found" || gpt["bot"] != "GPTBot" || gpt["access"] != "blocked" || gpt["followsAdvice"] != false {
		t.Errorf("tribelt = %v", tribelt)
	}
	if got["checkedAt"] != "2026-10-27T18:30:00Z" {
		t.Errorf("checkedAt = %v", got["checkedAt"])
	}
	if c := got["sitemap"].(map[string]any)["coverage"].(map[string]any); c["pages"].(float64) != 40 || c["listed"].(float64) != 34 || c["share"].(float64) != 0.85 {
		t.Errorf("coverage = %v", c)
	}
	if got["stack"].(map[string]any)["cms"] != "WordPress" || got["stack"].(map[string]any)["seoPlugin"] != nil || got["gaps"].(float64) != 8 {
		t.Errorf("stack and gaps = %v %v", got["stack"], got["gaps"])
	}
	brief := tribelt["scan"].(map[string]any)["aiReadiness"].(map[string]any)
	if brief["blockedBots"].([]any)[0] != "GPTBot" {
		t.Errorf("brief = %v", brief)
	}
	if twente := items[1].(map[string]any); twente["scan"] != nil || twente["readiness"] != nil || twente["role"] != "competitor" {
		t.Errorf("an unchecked company = %v", twente)
	}
	a.readinessErr = errors.New("down")
	if r := call(t, srv, http.MethodGet, "/api/v1/ai-readiness", "", true); r.code != http.StatusInternalServerError {
		t.Errorf("failing table: %d", r.code)
	}
	if r := call(t, srv, http.MethodGet, "/api/v1/ai-readiness", "", false); r.code != http.StatusUnauthorized {
		t.Errorf("anonymous table: %d", r.code)
	}
}

func TestAScanShowsItsReadiness(t *testing.T) {
	scan := done()
	rd := readiness()
	scan.Readiness = &rd
	srv := serve(t, &analyses{stored: scan})
	r := call(t, srv, http.MethodGet, "/api/v1/analyses/"+scan.ID.String(), "", true)
	got, _ := r.body["aiReadiness"].(map[string]any)
	if got != nil && got["sitemap"].(map[string]any)["coverage"] != nil {
		t.Errorf("a readiness without coverage sends %v", got["sitemap"])
	}
	if r.code != http.StatusOK || got == nil || len(got["advice"].([]any)) != 7 {
		t.Fatalf("scan = %d %v", r.code, r.body)
	}
}

func TestReadinessFilesAreForEveryoneSignedIn(t *testing.T) {
	id := uuid.New()
	a := &analyses{rawFiles: map[uuid.UUID]domain.RawFiles{id: {Robots: "User-agent: GPTBot\nDisallow: /", LLMsTruncated: true}}}
	srv := serve(t, a)
	path := "/api/v1/analyses/" + id.String() + "/readiness-files"
	r := callAs(t, srv, http.MethodGet, path, "", "tester")
	if r.code != http.StatusOK || r.body["robotsTxt"] != "User-agent: GPTBot\nDisallow: /" || r.body["llmsTruncated"] != true || r.body["llmsTxt"] != "" {
		t.Fatalf("files = %d %v", r.code, r.body)
	}
	if r := callAs(t, srv, http.MethodGet, "/api/v1/analyses/"+uuid.NewString()+"/readiness-files", "", "admin"); r.code != http.StatusNotFound {
		t.Errorf("files of a scan without readiness: %d", r.code)
	}
	a.readinessErr = errors.New("down")
	if r := callAs(t, srv, http.MethodGet, path, "", "admin"); r.code != http.StatusInternalServerError {
		t.Errorf("failing files: %d", r.code)
	}
}

func TestCompareShowsReadinessChanges(t *testing.T) {
	a := &analyses{readinessChanges: []domain.ReadinessChange{{Field: "llmsTxt", From: "no", To: "yes"}}}
	srv := serve(t, a)
	r := call(t, srv, http.MethodGet, "/api/v1/competitors/twentebelt/scans/compare", "", true)
	changes, _ := r.body["readiness"].([]any)
	if r.code != http.StatusOK || len(changes) != 1 || changes[0].(map[string]any)["to"] != "yes" {
		t.Fatalf("compare = %d %v", r.code, r.body)
	}
	a.readinessChanges = nil
	if r := call(t, srv, http.MethodGet, "/api/v1/competitors/twentebelt/scans/compare", "", true); r.body["readiness"] != nil {
		t.Errorf("a comparison without readiness lists %v", r.body["readiness"])
	}
}
