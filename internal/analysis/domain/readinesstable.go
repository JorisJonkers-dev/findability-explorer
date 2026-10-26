package domain

import (
	"cmp"
	"slices"
	"strings"

	"github.com/JorisJonkers-dev/findability-explorer/internal/shared/fold"
)

// ReadinessRow is one company in the AI readiness table, with its newest scan that recorded its
// readiness; Scan is nil when no scan did.
type ReadinessRow struct {
	Company Company
	Scan    *Analysis
}

// ReadinessTable lists every company that is not archived with its latest readiness: Tribelt on
// top as the baseline, the mirror below it as the target, then Tribelt's group companies, then
// the competitors, each band by name.
func ReadinessTable(companies []Company, latest map[string]Analysis) []ReadinessRow {
	out := make([]ReadinessRow, 0, len(companies))
	for _, c := range companies {
		if c.Archived {
			continue
		}
		row := ReadinessRow{Company: c}
		if a, ok := latest[c.ID]; ok {
			row.Scan = &a
		}
		out = append(out, row)
	}
	// A company without a role is a competitor.
	bands := []Role{RoleTribelt, RoleMirror, RoleGroup}
	band := func(r Role) int {
		if i := slices.Index(bands, r); i >= 0 {
			return i
		}
		return len(bands)
	}
	slices.SortStableFunc(out, func(a, b ReadinessRow) int {
		return cmp.Or(cmp.Compare(band(a.Company.Role), band(b.Company.Role)),
			strings.Compare(fold.Fold(a.Company.Name), fold.Fold(b.Company.Name)))
	})
	return out
}
