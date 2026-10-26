package app_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/JorisJonkers-dev/findability-explorer/internal/analysis/app"
	"github.com/JorisJonkers-dev/findability-explorer/internal/analysis/domain"
)

// readySite is a site's readiness files: robots.txt that blocks GPTBot, and llms.txt.
func readySite(gptbot string) domain.ReadinessFiles {
	return domain.ReadinessFiles{
		Robots: domain.File{Status: 200, ContentType: "text/plain", Body: "User-agent: GPTBot\n" + gptbot + "\n"},
		LLMs:   domain.File{Status: 200, ContentType: "text/plain", Body: "# X\n- [A](/a.md)\n"},
		Home:   domain.File{Status: 200, ContentType: "text/html", Body: "<html></html>"},
	}
}

func bot(r *domain.Readiness, token string) domain.BotRule {
	for _, b := range r.Bots {
		if b.Bot == token {
			return b
		}
	}
	return domain.BotRule{}
}

func TestRunRecordsTheSitesAIReadiness(t *testing.T) {
	r := newRepo()
	svc := service(r, crawler{files: readySite("Disallow: /"), pages: pages()}, nil, limits())
	a, err := svc.Start(t.Context(), "u", "https://x.test/", 1, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Run(t.Context(), a.ID); err != nil {
		t.Fatal(err)
	}
	got, err := svc.Get(t.Context(), a.ID)
	if err != nil || got.Readiness == nil {
		t.Fatalf("scan = %+v, %v", got, err)
	}
	if bot(got.Readiness, "GPTBot").Access != domain.AccessBlocked || !got.Readiness.LLMs.Present || !got.Readiness.Markdown {
		t.Errorf("readiness = %+v", got.Readiness)
	}
	raw, err := svc.ReadinessFiles(t.Context(), a.ID)
	if err != nil || !strings.Contains(raw.Robots, "GPTBot") || raw.LLMs == "" {
		t.Errorf("raw files = %+v, %v", raw, err)
	}
}

func TestRunRecordsReadinessWhenTheSiteBlocksTheCrawl(t *testing.T) {
	r := newRepo()
	blocked := crawler{files: readySite("Disallow: /"), err: domain.ErrBlockedBySite}
	svc := service(r, blocked, nil, limits())
	a, _ := svc.Start(t.Context(), "u", "https://x.test/", 1, "")
	if err := svc.Run(t.Context(), a.ID); err != nil {
		t.Fatal(err)
	}
	got, _ := svc.Get(t.Context(), a.ID)
	if !got.Blocked() || got.Readiness == nil || bot(got.Readiness, "GPTBot").Access != domain.AccessBlocked {
		t.Fatalf("a blocked scan keeps its crawler policy: %+v", got)
	}
}

func TestRunWithoutReadinessWhenItCannotBeRead(t *testing.T) {
	r := newRepo()
	svc := service(r, crawler{readinessErr: context.Canceled, pages: pages()}, nil, limits())
	a, _ := svc.Start(t.Context(), "u", "https://x.test/", 1, "")
	if err := svc.Run(t.Context(), a.ID); err != nil {
		t.Fatal(err)
	}
	got, _ := svc.Get(t.Context(), a.ID)
	if got.Readiness != nil || got.Status != domain.StatusDone {
		t.Fatalf("scan = %+v", got)
	}
	if _, err := svc.ReadinessFiles(t.Context(), a.ID); !errors.Is(err, domain.ErrNoReadiness) {
		t.Fatalf("raw files of a scan without readiness = %v", err)
	}
	failing := newRepo("SaveReadiness")
	svc = service(failing, crawler{files: readySite("Allow: /")}, nil, limits())
	a, _ = svc.Start(t.Context(), "u", "https://x.test/", 1, "")
	if err := svc.Run(t.Context(), a.ID); !errors.Is(err, errBoom) {
		t.Fatalf("Run = %v, want the store's failure", err)
	}
	if failing.analyses[a.ID].Status != domain.StatusFailed {
		t.Fatal("a scan whose readiness could not be stored stays running")
	}
}

func TestReadinessTableBandsTheCompanies(t *testing.T) {
	r := newRepo()
	dir := &directory{companies: []domain.Company{
		{ID: "x", Name: "X Belts", Role: domain.RoleCompetitor},
		{ID: "tribelt", Name: "Tribelt", Role: domain.RoleTribelt},
	}}
	svc := app.New(r, dir, crawler{files: readySite("Disallow: /")}, source{}, glossary, func() time.Time { return now }, limits())
	a, _ := svc.Start(t.Context(), "u", "https://x.test/", 1, "")
	_ = svc.Run(t.Context(), a.ID)
	rows, err := svc.ReadinessTable(t.Context())
	if err != nil || len(rows) != 2 || rows[0].Company.ID != "tribelt" || rows[0].Scan != nil || rows[1].Scan == nil {
		t.Fatalf("rows = %+v, %v", rows, err)
	}
	for name, s := range map[string]*app.Service{
		"companies": app.New(r, &directory{companiesErr: errBoom}, crawler{}, source{}, glossary, func() time.Time { return now }, limits()),
		"readiness": app.New(newRepo("LatestReadiness"), dir, crawler{}, source{}, glossary, func() time.Time { return now }, limits()),
	} {
		if _, err := s.ReadinessTable(t.Context()); !errors.Is(err, errBoom) {
			t.Errorf("%s failing: %v", name, err)
		}
	}
}

func TestCompareListsReadinessChanges(t *testing.T) {
	r := newRepo()
	dir := &directory{companies: companies()}
	var ids []uuid.UUID
	for _, gptbot := range []string{"Allow: /", "Disallow: /"} {
		svc := app.New(r, dir, crawler{files: readySite(gptbot), pages: pages()}, source{}, glossary, func() time.Time { return now }, limits())
		a, err := svc.Start(t.Context(), "u", "https://x.test/", 1, "")
		if err != nil {
			t.Fatal(err)
		}
		_ = svc.Run(t.Context(), a.ID)
		ids = append(ids, a.ID)
	}
	svc := app.New(r, dir, crawler{}, source{}, glossary, func() time.Time { return now }, limits())
	c, err := svc.Compare(t.Context(), "x", uuid.Nil, uuid.Nil)
	if err != nil || c.From.ID != ids[0] || c.To.ID != ids[1] {
		t.Fatalf("Compare = %+v, %v", c, err)
	}
	if len(c.Readiness) != 1 || c.Readiness[0] != (domain.ReadinessChange{Field: "bot:GPTBot", From: "allowed", To: "blocked"}) {
		t.Errorf("readiness changes = %+v", c.Readiness)
	}
}
