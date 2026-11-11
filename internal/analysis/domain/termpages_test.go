package domain_test

import (
	"strings"
	"testing"

	"github.com/JorisJonkers-dev/findability-explorer/internal/analysis/domain"
)

func TestTermPagesOfGroupsThePagesByCompany(t *testing.T) {
	companies := []domain.Company{
		{ID: "twentebelt", Name: "Twentebelt", Role: domain.RoleCompetitor},
		{ID: "esfo", Name: "ESFO", Role: domain.RoleCompetitor},
		{ID: "tribelt", Name: "Tribelt B.V.", Role: domain.RoleTribelt},
		{ID: "tribelt-mirror", Name: "Mirror of tribelt.nl", Role: domain.RoleMirror},
		{ID: "gone", Name: "Gone", Role: domain.RoleCompetitor, Archived: true},
	}
	rows := []domain.TermPageRow{
		{CompetitorID: "esfo", Position: -1},
		{CompetitorID: "gone", Position: 0, URL: "https://gone.test/"},
		{CompetitorID: "tribelt", Position: 2, URL: "https://www.tribelt.nl/en/eyelink", Title: "Eyelink belts"},
		{CompetitorID: "tribelt", Position: 2, URL: "https://www.tribelt.nl/en/eyelink", Title: "Eyelink belts"},
		{CompetitorID: "tribelt", Position: 0, URL: "https://www.tribelt.nl/"},
		{CompetitorID: "tribelt-mirror", Position: 0, URL: "https://mirror.test/"},
		{CompetitorID: "twentebelt", Position: 1, URL: "https://www.twentebelt.com/belts", Title: "Belts"},
		{CompetitorID: "unknown", Position: 0, URL: "https://unknown.test/"},
	}
	got := domain.TermPagesOf(rows, companies)
	var names []string
	for _, c := range got {
		names = append(names, c.Company.Name)
	}
	if strings.Join(names, ", ") != "Tribelt B.V., ESFO, Twentebelt" {
		t.Fatalf("companies = %v, want Tribelt first, then by name; the mirror, an archived and an unknown company left out", names)
	}
	if p := got[0].Pages; len(p) != 2 || p[0].URL != "https://www.tribelt.nl/en/eyelink" || p[0].Title != "Eyelink belts" || p[1].URL != "https://www.tribelt.nl/" {
		t.Errorf("Tribelt's pages = %+v, each once, in the scan's order", p)
	}
	if len(got[1].Pages) != 0 {
		t.Errorf("a scan from before pages were recorded lists none: %+v", got[1].Pages)
	}
}

func TestTermPagesOfKeepsAtMostMaxTermPagesACompany(t *testing.T) {
	var rows []domain.TermPageRow
	for i := range domain.MaxTermPages + 3 {
		rows = append(rows, domain.TermPageRow{CompetitorID: "x", Position: i, URL: "https://x.test/" + string(rune('a'+i))})
	}
	got := domain.TermPagesOf(rows, []domain.Company{{ID: "x", Name: "X", Role: domain.RoleGroup}})
	if len(got) != 1 || len(got[0].Pages) != domain.MaxTermPages {
		t.Fatalf("got %+v", got)
	}
}

func TestKeptPageTextIsCappedLikeTheFiles(t *testing.T) {
	if got := domain.KeptPageText(domain.Page{Text: "Eyelink belts.\x00"}); got != "Eyelink belts." {
		t.Errorf("text = %q", got)
	}
	if got := domain.KeptPageText(domain.Page{Text: strings.Repeat("é", domain.MaxRawFile)}); len(got) > domain.MaxRawFile || len(got) < domain.MaxRawFile-1 {
		t.Errorf("a long page keeps %d bytes", len(got))
	}
}
