package domain_test

import (
	"testing"

	"github.com/JorisJonkers-dev/findability-explorer/internal/analysis/domain"
)

func TestReadinessTablePinsTribeltAndTheMirrorAboveTheGroupAndTheCompetitors(t *testing.T) {
	companies := []domain.Company{
		{ID: "twentebelt", Name: "Twentebelt", Role: domain.RoleCompetitor},
		{ID: "esfo", Name: "ESFO"},
		{ID: "de-spiraal", Name: "De Spiraal", Role: domain.RoleGroup},
		{ID: "old", Name: "Archived Belts", Role: domain.RoleCompetitor, Archived: true},
		{ID: "tribelt-mirror", Name: "Mirror of tribelt.nl", Role: domain.RoleMirror},
		{ID: "lesjofors", Name: "Lesjöfors", Role: domain.RoleGroup},
		{ID: "tribelt", Name: "Tribelt B.V.", Role: domain.RoleTribelt},
	}
	r := domain.AssessReadiness(domain.ReadinessFiles{})
	rows := domain.ReadinessTable(companies, map[string]domain.Analysis{"esfo": {Number: 2, Readiness: &r}})
	want := []string{"tribelt", "tribelt-mirror", "de-spiraal", "lesjofors", "esfo", "twentebelt"}
	if len(rows) != len(want) {
		t.Fatalf("rows = %+v", rows)
	}
	for i, id := range want {
		if rows[i].Company.ID != id {
			t.Errorf("row %d = %s, want %s", i, rows[i].Company.ID, id)
		}
	}
	if rows[4].Scan == nil || rows[4].Scan.Number != 2 || rows[5].Scan != nil {
		t.Errorf("scans = %+v, %+v", rows[4].Scan, rows[5].Scan)
	}
}
