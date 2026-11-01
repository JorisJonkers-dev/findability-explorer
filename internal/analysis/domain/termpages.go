package domain

import (
	"cmp"
	"slices"
)

// TribeltID is Tribelt's own id among the companies; only a scan of tribelt.nl keeps its pages' text.
const TribeltID = "tribelt"

// TermPageRow is one page of a company's newest finished scan that uses a term, by its position
// among the scan's pages; Position is -1 for a scan that found the term before scans recorded
// its pages.
type TermPageRow struct {
	CompetitorID string
	Position     int
	URL          string
	Title        string
}

// TermPage is a page that uses a term.
type TermPage struct {
	URL   string
	Title string
}

// CompanyTermPages are one company's pages that use a term; none for a scan from before scans
// recorded them.
type CompanyTermPages struct {
	Company Company
	Pages   []TermPage
}

// TermPagesOf groups the rows by company: Tribelt first, then the competitors and the group by
// name, each page once and at most MaxTermPages; the mirror, an archived company and one the
// directory does not know are left out.
func TermPagesOf(rows []TermPageRow, companies []Company) []CompanyTermPages {
	byID := map[string]*CompanyTermPages{}
	var out []*CompanyTermPages
	for _, row := range rows {
		c, ok := byID[row.CompetitorID]
		if !ok {
			i := slices.IndexFunc(companies, func(c Company) bool { return c.ID == row.CompetitorID })
			if i < 0 || companies[i].Role == RoleMirror || companies[i].Archived {
				continue
			}
			c = &CompanyTermPages{Company: companies[i], Pages: []TermPage{}}
			byID[row.CompetitorID] = c
			out = append(out, c)
		}
		seen := slices.ContainsFunc(c.Pages, func(p TermPage) bool { return p.URL == row.URL })
		if row.Position >= 0 && !seen && len(c.Pages) < MaxTermPages {
			c.Pages = append(c.Pages, TermPage{URL: row.URL, Title: row.Title})
		}
	}
	slices.SortStableFunc(out, func(a, b *CompanyTermPages) int {
		return cmp.Or(cmp.Compare(rank(a.Company.Role), rank(b.Company.Role)), cmp.Compare(a.Company.Name, b.Company.Name))
	})
	result := make([]CompanyTermPages, 0, len(out))
	for _, c := range out {
		result = append(result, *c)
	}
	return result
}

// rank puts Tribelt before every other company.
func rank(r Role) int {
	if r == RoleTribelt {
		return 0
	}
	return 1
}

// KeptPageText is the text a scan of tribelt.nl keeps of a page: its visible text, capped at
// MaxRawFile bytes as the raw files are.
func KeptPageText(p Page) string {
	text, _ := rawText(p.Text)
	return text
}
