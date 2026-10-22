package seed

import (
	"errors"
	"strings"
	"testing"

	"github.com/JorisJonkers-dev/findability-explorer/internal/terminology/domain"
)

func TestLoadTheEmbeddedMap(t *testing.T) {
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Terms()) != 146 || len(c.Companies()) != 28 {
		t.Fatalf("loaded %d terms and %d companies, want 146 and 28", len(c.Terms()), len(c.Companies()))
	}
	var twentebelt domain.Company
	for _, co := range c.Companies() {
		if co.Name == "Twentebelt" {
			twentebelt = co
		}
	}
	families := map[domain.Family]bool{}
	for _, b := range twentebelt.BeltNames {
		families[b.Family] = true
	}
	for _, f := range []domain.Family{domain.FamilyEyelink, domain.FamilyOpenSpiral, domain.FamilyClosedSpiral, domain.FamilyTriFlex, domain.FamilyOther} {
		if !families[f] {
			t.Errorf("Twentebelt has no belt name in family %s: %+v", f, twentebelt.BeltNames)
		}
	}
	// The e2e suite searches for TwenteFlex, which the map holds as a user of "spiral belt".
	if got := domain.Search(c.Terms(), domain.Query{Text: "TwenteFlex", Limit: 5}); got.Total == 0 {
		t.Error("searching the map for TwenteFlex finds nothing; the e2e suite relies on it")
	}
}

func TestParseRefusesBadData(t *testing.T) {
	good := `[{"term":"x","language":"en","belt_family":"other","intent_tier":"product","used_by":["A"],"source":"s"}]`
	cases := map[string][2]string{
		"terms not JSON":     {"{", "[]"},
		"companies not JSON": {good, "{"},
		"unknown language":   {strings.Replace(good, `"en"`, `"xx"`, 1), "[]"},
		"unknown family":     {strings.Replace(good, `"other"`, `"plastic"`, 1), "[]"},
		"unknown tier":       {strings.Replace(good, `"product"`, `"vague"`, 1), "[]"},
	}
	for name, c := range cases {
		if _, err := parse([]byte(c[0]), []byte(c[1])); err == nil {
			t.Errorf("%s: parse succeeded", name)
		}
	}
	if _, err := parse([]byte(strings.Replace(good, `"en"`, `"xx"`, 1)), []byte("[]")); !errors.Is(err, domain.ErrUnknownValue) {
		t.Errorf("unknown language error = %v", err)
	}
}

func TestCompaniesWithoutTitleAreUnverified(t *testing.T) {
	c, err := parse([]byte("[]"), []byte(`[{"name":"X","belt_names":[{"competitor_name":"Y","tribelt_equivalent":"Other belts"}]}]`))
	if err != nil {
		t.Fatal(err)
	}
	if co := c.Companies()[0]; co.HomepageTitle != "unverified" || co.BeltNames[0].Family != domain.FamilyOther {
		t.Fatalf("company = %+v", co)
	}
}

func TestFamilyOf(t *testing.T) {
	cases := map[string]domain.Family{
		"Eyelink belt (draadogenband)":            domain.FamilyEyelink,
		"Eye-Link":                                domain.FamilyEyelink,
		"Open spiral / balanced weave":            domain.FamilyOpenSpiral,
		"Balanced weave":                          domain.FamilyOpenSpiral,
		"Closed spiral / compound balanced":       domain.FamilyClosedSpiral,
		"Compound":                                domain.FamilyClosedSpiral,
		"Tri-Flex / turn-curve":                   domain.FamilyTriFlex,
		"turn-curve":                              domain.FamilyTriFlex,
		"Spiral tower belt":                       domain.FamilyTriFlex,
		"Other belts (rod, flat wire, honeycomb)": domain.FamilyOther,
	}
	for in, want := range cases {
		if got := familyOf(in); got != want {
			t.Errorf("familyOf(%q) = %s, want %s", in, got, want)
		}
	}
}
