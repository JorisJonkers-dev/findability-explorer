package domain

import (
	"context"
	"fmt"
)

// TribeltFamily is one of Tribelt's belt families, described for the language model that matches
// a site's products to them.
type TribeltFamily struct {
	Family      string
	Name        string
	Description string
}

// TribeltFamilies are the families the matcher may answer with. Their ids are the terminology
// map's family values.
func TribeltFamilies() []TribeltFamily {
	return []TribeltFamily{
		{
			"eyelink", "Eyelink belt (draadogenband, Drahtösengliederband)",
			"Straight-running, positively driven belt of wire eyelinks joined by cross rods; hygienic, open, for ovens, tunnels and food lines.",
		},
		{
			"open_spiral", "Open spiral / balanced weave belt (open spiraalband, Weitspiralband)",
			"Woven belt of alternating left and right wire spirals joined by cross rods, with an open mesh; high temperature, ovens, drying, cooling.",
		},
		{
			"closed_spiral", "Closed spiral / compound balanced belt (gesloten spiraalband, Engspiralband)",
			"Tightly woven spiral belt with a dense, flat surface for small parts and baking; high heat transfer, wear resistant.",
		},
		{
			"tri_flex_turn_curve", "Tri-Flex (curve-going and spiral-tower belt)",
			"Stainless belt that runs through curves and in spiral towers for freezing, cooling and proofing.",
		},
		{
			"other", "Other metal belts and drives",
			"Rod, chain, honeycomb, flat wire, Z and woven wire belts, and the sprockets and drums that drive them.",
		},
	}
}

// SearchTerm is a term people may search with, in one language.
type SearchTerm struct {
	Term     string
	Language string
}

// Equivalent is a product on a site matched to a Tribelt family.
type Equivalent struct {
	Product     string
	Family      string
	Reason      string
	SearchTerms []SearchTerm
}

// ErrUnknownFamily is returned when the matcher answers with a family Tribelt does not have.
var ErrUnknownFamily = fmt.Errorf("unknown belt family")

// CheckEquivalent returns e when its family is one of TribeltFamilies and it names a product.
func CheckEquivalent(e Equivalent) (Equivalent, error) {
	if e.Product == "" {
		return Equivalent{}, fmt.Errorf("%w: no product named", ErrUnknownFamily)
	}
	for _, f := range TribeltFamilies() {
		if f.Family == e.Family {
			return e, nil
		}
	}
	return Equivalent{}, fmt.Errorf("%w: %q", ErrUnknownFamily, e.Family)
}

// Matcher is the port the language-model adapter implements: it reads a site's text and names the
// site's products that correspond to Tribelt's families, with the terms the site uses for them.
type Matcher interface {
	Match(ctx context.Context, siteText string, truncated bool, families []TribeltFamily) ([]Equivalent, Usage, error)
}

// Matchers is the port to the language-model settings: the matcher to use now, and a label naming
// its provider and model. A nil Matcher means matching is off.
type Matchers interface {
	Current(ctx context.Context) (Matcher, string, error)
}

// Crawler is the port the crawler adapter implements. It fetches at most maxPages pages on the
// target's site, following robots.txt, the pages scorer ranks highest first, and calls onPage
// after each page in crawl order.
type Crawler interface {
	Crawl(ctx context.Context, target Target, maxPages int, scorer LinkScorer, onPage func(Page) error) error
	// Readiness fetches the site's readiness files; it fails only when ctx ends.
	Readiness(ctx context.Context, target Target) (ReadinessFiles, error)
}
