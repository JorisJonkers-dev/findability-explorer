// Package research hands the desk research's companies, which the terminology context's seed
// holds, to the companies context: an anti-corruption adapter between the two.
package research

import (
	companiesdomain "github.com/JorisJonkers-dev/findability-explorer/internal/companies/domain"
	"github.com/JorisJonkers-dev/findability-explorer/internal/shared/site"
	terminologydomain "github.com/JorisJonkers-dev/findability-explorer/internal/terminology/domain"
)

// Companies maps the terminology seed's companies to the companies context's, by value.
func Companies(companies []terminologydomain.Company) []companiesdomain.Company {
	out := make([]companiesdomain.Company, 0, len(companies))
	for _, c := range companies {
		names := make([]companiesdomain.BeltName, 0, len(c.BeltNames))
		for _, b := range c.BeltNames {
			names = append(names, companiesdomain.BeltName{Name: b.Name, Family: string(b.Family)})
		}
		website := companiesdomain.ResearchWebsite(c.Website)
		host := ""
		if website != nil {
			host = site.Host(website)
		}
		out = append(out, companiesdomain.Company{
			ID: companiesdomain.ID(c.Name), Source: companiesdomain.SourceResearch, Name: c.Name,
			Country: c.Country, Website: c.Website, URL: website, Host: host, Languages: c.Languages,
			GroupCompany: c.GroupCompany, BeltNames: names, HomepageTitle: c.HomepageTitle,
		})
	}
	return out
}
