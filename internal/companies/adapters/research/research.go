// Package research hands the desk research's companies, which the terminology context's seed
// holds, to the companies context: an anti-corruption adapter between the two.
package research

import (
	"slices"

	companiesdomain "github.com/JorisJonkers-dev/findability-explorer/internal/companies/domain"
	"github.com/JorisJonkers-dev/findability-explorer/internal/shared/lang"
	"github.com/JorisJonkers-dev/findability-explorer/internal/shared/site"
	terminologydomain "github.com/JorisJonkers-dev/findability-explorer/internal/terminology/domain"
)

// Companies maps the terminology seed's companies to the companies context's, by value.
func Companies(companies []terminologydomain.Company) []companiesdomain.Company {
	out := make([]companiesdomain.Company, 0, len(companies))
	for _, c := range companies {
		// Names in Dutch, German or English come first; the others stay, as they still show which
		// families the company offers.
		beltNames := slices.Clone(c.BeltNames)
		slices.SortStableFunc(beltNames, func(a, b terminologydomain.BeltName) int {
			return boolOrder(lang.InFocus(b.Language), lang.InFocus(a.Language))
		})
		names := make([]companiesdomain.BeltName, 0, len(beltNames))
		for _, b := range beltNames {
			names = append(names, companiesdomain.BeltName{Name: b.Name, Family: string(b.Family)})
		}
		// The website's Dutch, German and English versions, once each: en-GB is English.
		var languages []string
		for _, l := range c.Languages {
			if p := lang.Primary(l); lang.InFocus(p) && !slices.Contains(languages, p) {
				languages = append(languages, p)
			}
		}
		website := companiesdomain.ResearchWebsite(c.Website)
		host := ""
		if website != nil {
			host = site.Host(website)
		}
		out = append(out, companiesdomain.Company{
			ID: companiesdomain.ID(c.Name), Source: companiesdomain.SourceResearch, Name: c.Name,
			Country: c.Country, Website: c.Website, URL: website, Host: host, Languages: languages,
			GroupCompany: c.GroupCompany, BeltNames: names, HomepageTitle: c.HomepageTitle,
		})
	}
	return out
}

// boolOrder orders true before false: -1 when a is true and b is not, 1 the other way round.
func boolOrder(a, b bool) int {
	switch {
	case a == b:
		return 0
	case a:
		return 1
	default:
		return -1
	}
}
