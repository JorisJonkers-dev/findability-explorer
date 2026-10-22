package domain_test

import (
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/JorisJonkers-dev/findability-explorer/internal/companies/domain"
)

func TestID(t *testing.T) {
	for name, want := range map[string]string{
		"Twentebelt":  "twentebelt",
		"PELGER GmbH": "pelger",
		"Märtens Transportbänder GmbH (MARTENS)": "martens-transportbander",
		"Wire Belt Company Ltd":                  "wire-belt-company",
		"Wire Belt Company of America":           "wire-belt-company-of-america",
		"Tribelt B.V.":                           "tribelt",
		"Fratelli Mariani S.p.A.":                "fratelli-mariani",
		"HEIN, LEHMANN GmbH":                     "hein-lehmann",
		"GKD – Gebr. Kufferath AG":               "gkd-gebr-kufferath",
		"GmbH":                                   "gmbh",
		"Łódź Belts sp":                          "lodz-belts-sp",
		"!!!":                                    "",
		strings.Repeat("belt ", 40):              strings.TrimRight(strings.Repeat("belt-", 16), "-"),
	} {
		if got := domain.ID(name); got != want {
			t.Errorf("ID(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestCheckName(t *testing.T) {
	if got, err := domain.CheckName("  Example Belts "); err != nil || got != "Example Belts" {
		t.Errorf("CheckName = %q, %v", got, err)
	}
	for _, bad := range []string{"", "   ", strings.Repeat("x", domain.MaxNameLength+1)} {
		if _, err := domain.CheckName(bad); !errors.Is(err, domain.ErrInvalidName) {
			t.Errorf("CheckName(%q) error = %v", bad, err)
		}
	}
}

func TestNewWebsite(t *testing.T) {
	if u, err := domain.NewWebsite(" https://www.example-belts.com#top "); err != nil || u.String() != "https://www.example-belts.com/" {
		t.Errorf("NewWebsite = %v, %v", u, err)
	}
	for _, bad := range []string{"ftp://example.com", "https://u:p@example.com", "https://localhost/", "https://10.0.0.1/", "https://%zz", "https://example.com/" + strings.Repeat("a", 500)} {
		if _, err := domain.NewWebsite(bad); !errors.Is(err, domain.ErrInvalidWebsite) {
			t.Errorf("NewWebsite(%q) error = %v", bad, err)
		}
	}
}

func TestResearchWebsite(t *testing.T) {
	for note, want := range map[string]string{
		"https://www.twentebelt.com": "https://www.twentebelt.com/",
		"https://www.cambridge-es.com (blocked by Cloudflare 403 on 2026-10-22)": "https://www.cambridge-es.com/",
		"unverified (no official site found)":                                    "",
		"unverified (geppert-band.de did not resolve on 2026-10-22)":             "",
		"see http://[::1 and https://ok.example.com":                             "https://ok.example.com/",
	} {
		got := domain.ResearchWebsite(note)
		if (want == "" && got != nil) || (want != "" && (got == nil || got.String() != want)) {
			t.Errorf("ResearchWebsite(%q) = %v, want %q", note, got, want)
		}
	}
}

func mustURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func directory(t *testing.T) domain.Directory {
	t.Helper()
	return domain.NewDirectory(
		[]domain.Company{
			{ID: "tribelt", Name: "Tribelt B.V.", Host: "tribelt.nl", GroupCompany: true, Source: domain.SourceResearch},
			{ID: "de-spiraal", Name: "De Spiraal B.V.", Host: "tribelt.nl", GroupCompany: true, Source: domain.SourceResearch},
			{ID: "twentebelt", Name: "Twentebelt", Host: "twentebelt.com", Source: domain.SourceResearch},
			{ID: "ntm", Name: "NTM", Source: domain.SourceResearch},
			{ID: "ashworth", Name: "ashworth", Host: "ashworth.com", Source: domain.SourceResearch},
		},
		[]domain.Company{{ID: "example-belts", Name: "Example Belts", Host: "example-belts.com", Source: domain.SourceAdded}},
	)
}

func TestDirectoryFinds(t *testing.T) {
	d := directory(t)
	if c, ok := d.ByID("example-belts"); !ok || c.Name != "Example Belts" {
		t.Errorf("ByID = %+v, %v", c, ok)
	}
	if _, ok := d.ByID("nobody"); ok {
		t.Error("ByID found nobody")
	}
	if c, ok := d.ByHost("tribelt.nl"); !ok || c.ID != "tribelt" {
		t.Errorf("ByHost(tribelt.nl) = %+v: the first company on a shared host wins", c)
	}
	if _, ok := d.ByHost(""); ok {
		t.Error("ByHost found a company without a website")
	}
}

func TestNewCompany(t *testing.T) {
	d := directory(t)
	at := time.Date(2026, 10, 22, 9, 0, 0, 0, time.UTC)
	c, err := d.NewCompany("Twentebelt", mustURL(t, "https://www.twentebelt.org/"), "joris", at)
	if err != nil || c.ID != "twentebelt-2" || c.Host != "twentebelt.org" || c.Source != domain.SourceAdded ||
		c.AddedBy != "joris" || !c.AddedAt.Equal(at) || c.Website != "https://www.twentebelt.org/" {
		t.Errorf("NewCompany = %+v, %v", c, err)
	}
	c, err = d.NewCompany("!!!", mustURL(t, "https://new-belts.example/"), "joris", at)
	if err != nil || c.ID != "new-belts-example" {
		t.Errorf("NewCompany without a usable name = %+v, %v", c, err)
	}
	var known domain.AlreadyKnownError
	if _, err := d.NewCompany("Anything", mustURL(t, "https://WWW.TwenteBelt.com/x"), "joris", at); !errors.As(err, &known) ||
		known.ID != "twentebelt" || err.Error() != "Twentebelt already has this website" {
		t.Errorf("NewCompany on a known host error = %v", err)
	}
}

func TestSorted(t *testing.T) {
	scan := domain.Scan{URL: "https://www.twentebelt.com/"}
	got := directory(t).Sorted(map[string]domain.Stats{"twentebelt": {Count: 2, Last: &scan}})
	var ids []string
	for _, l := range got {
		ids = append(ids, l.ID)
	}
	if strings.Join(ids, ",") != "ashworth,ntm,twentebelt,example-belts,de-spiraal,tribelt" {
		t.Fatalf("Sorted = %v", ids)
	}
	if got[2].Stats.Count != 2 || got[2].Stats.Last != &scan || got[0].Stats.Count != 0 {
		t.Fatalf("stats not attached: %+v", got[2].Stats)
	}
}
