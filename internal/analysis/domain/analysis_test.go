package domain_test

import (
	"errors"
	"net/netip"
	"strings"
	"testing"

	"github.com/JorisJonkers-dev/findability-explorer/internal/analysis/domain"
)

func TestNewTargetAcceptsPublicWebAddresses(t *testing.T) {
	cases := map[string]string{
		"https://www.twentebelt.com":            "https://www.twentebelt.com/",
		"http://example.com/products#top":       "http://example.com/products",
		"https://93.184.216.34/":                "https://93.184.216.34/",
		"https://EXAMPLE.org/belts?lang=de":     "https://EXAMPLE.org/belts?lang=de",
		"https://[2606:4700:4700::1111]/robots": "https://[2606:4700:4700::1111]/robots",
	}
	for in, want := range cases {
		target, err := domain.NewTarget(in)
		if err != nil {
			t.Errorf("NewTarget(%q) error = %v", in, err)
			continue
		}
		if target.String() != want {
			t.Errorf("NewTarget(%q) = %q, want %q", in, target.String(), want)
		}
		u := target.URL()
		u.Path = "/changed"
		if target.String() != want {
			t.Errorf("changing URL() changed the target")
		}
	}
}

func TestNewTargetRefusesEverythingElse(t *testing.T) {
	for _, in := range []string{
		"ftp://example.com/",
		"file:///etc/passwd",
		"https://user:pass@example.com/",
		"https://localhost/",
		"https://intranet/",
		"https://printer.local/",
		"https://db.internal/",
		"https://app.localhost/",
		"https://127.0.0.1/",
		"https://10.0.0.8/",
		"https://192.168.1.1/",
		"https://169.254.169.254/latest/meta-data",
		"https://100.64.1.1/",
		"https://[::1]/",
		"https://[fd00::1]/",
		"https:// bad host/",
		"https://%zz/",
		"https://example.com/" + strings.Repeat("a", 500),
	} {
		if _, err := domain.NewTarget(in); !errors.Is(err, domain.ErrInvalidTarget) {
			t.Errorf("NewTarget(%q) error = %v, want ErrInvalidTarget", in, err)
		}
	}
}

func TestPublicAddr(t *testing.T) {
	for addr, want := range map[string]bool{
		"8.8.8.8": true, "2606:4700:4700::1111": true, "::ffff:8.8.8.8": true,
		"127.0.0.1": false, "10.1.2.3": false, "172.16.0.1": false, "100.100.0.1": false,
		"0.0.0.0": false, "224.0.0.1": false, "fe80::1": false, "::ffff:192.168.0.1": false,
	} {
		if got := domain.PublicAddr(netip.MustParseAddr(addr)); got != want {
			t.Errorf("PublicAddr(%s) = %v, want %v", addr, got, want)
		}
	}
}

func TestCheckMaxPages(t *testing.T) {
	for in, want := range map[int]int{0: domain.DefaultMaxPages, 1: 1, 30: 30, 100: 100} {
		if got, err := domain.CheckMaxPages(in); err != nil || got != want {
			t.Errorf("CheckMaxPages(%d) = %d, %v", in, got, err)
		}
	}
	for _, in := range []int{-1, 101} {
		if _, err := domain.CheckMaxPages(in); !errors.Is(err, domain.ErrInvalidMaxPages) {
			t.Errorf("CheckMaxPages(%d) error = %v", in, err)
		}
	}
}

func TestUsageTotal(t *testing.T) {
	if got := (domain.Usage{InputTokens: 1200, OutputTokens: 34}).Total(); got != 1234 {
		t.Fatalf("Total = %d", got)
	}
}

func TestCheckEquivalent(t *testing.T) {
	ok := domain.Equivalent{Product: "TwenteFlex", Family: "tri_flex_turn_curve"}
	if got, err := domain.CheckEquivalent(ok); err != nil || got.Product != "TwenteFlex" {
		t.Errorf("CheckEquivalent(ok) = %+v, %v", got, err)
	}
	for _, bad := range []domain.Equivalent{{Product: "", Family: "eyelink"}, {Product: "Belt", Family: "plastic"}, {Product: "Belt", Family: "all"}} {
		if _, err := domain.CheckEquivalent(bad); !errors.Is(err, domain.ErrUnknownFamily) {
			t.Errorf("CheckEquivalent(%+v) error = %v", bad, err)
		}
	}
	if n := len(domain.TribeltFamilies()); n != 5 {
		t.Errorf("%d Tribelt families, want 5", n)
	}
}
