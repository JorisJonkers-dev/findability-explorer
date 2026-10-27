package site_test

import (
	"net/netip"
	"net/url"
	"testing"

	"github.com/JorisJonkers-dev/findability-explorer/internal/shared/site"
)

func TestPublicHostAndAddr(t *testing.T) {
	for host, want := range map[string]bool{
		"example.com": true, "8.8.8.8": true, "2606:4700:4700::1111": true,
		"localhost": false, "intranet": false, "a.local": false, "a.internal": false, "x.localhost": false,
		"10.0.0.1": false, "100.64.0.1": false, "::1": false, "": false,
	} {
		if got := site.PublicHost(host); got != want {
			t.Errorf("PublicHost(%q) = %v, want %v", host, got, want)
		}
	}
	if site.PublicAddr(netip.MustParseAddr("::ffff:192.168.0.1")) {
		t.Error("a mapped private address counted as public")
	}
}

func TestHost(t *testing.T) {
	for raw, want := range map[string]string{
		"https://WWW.Twentebelt.com/x": "twentebelt.com",
		"http://wirebelt.de:8080/":     "wirebelt.de",
		"https://www2.example.com":     "www2.example.com",
	} {
		u, _ := url.Parse(raw)
		if got := site.Host(u); got != want {
			t.Errorf("Host(%q) = %q, want %q", raw, got, want)
		}
	}
}

func TestDomain(t *testing.T) {
	for raw, want := range map[string]string{
		"https://www.Ashworth.com/products":         "ashworth.com",
		"jpconveyors.com/":                          "jpconveyors.com",
		"www.oramachines.com":                       "oramachines.com",
		"https://products.bandogrp.com/en/product/": "bandogrp.com",
		"https://www.dunlopbelting.co.uk/":          "dunlopbelting.co.uk",
		"https://de.globalmeshbelt.com/spiral":      "globalmeshbelt.com",
		"http://metiz.tw/product_conveyor_belt.htm": "metiz.tw",
		"https://tribelt.jorisjonkers.dev/":         "jorisjonkers.dev",
		"":                                          "",
		"no address":                                "",
		"localhost":                                 "",
		"https://co.uk/":                            "",
	} {
		if got := site.Domain(raw); got != want {
			t.Errorf("Domain(%q) = %q, want %q", raw, got, want)
		}
	}
}

func TestCompanyDomainLeavesOutPlatforms(t *testing.T) {
	for raw, want := range map[string]string{
		"https://www.ashworth.com/":                                     "ashworth.com",
		"https://www.europages.nl/ondernemingen/rothstein":              "",
		"https://facebook.com/1427195200825384":                         "",
		"https://m.made-in-china.com/product/Wire-Mesh.html":            "",
		"https://www.wlw.de/de/firma/industrieofen-und-anlagenbau-gmbh": "",
		"https://www.linkedin.com/company/x":                            "",
		"https://en.wikipedia.org/wiki/Conveyor_belt":                   "",
	} {
		if got := site.CompanyDomain(raw); got != want {
			t.Errorf("CompanyDomain(%q) = %q, want %q", raw, got, want)
		}
	}
}

func TestRoot(t *testing.T) {
	for raw, want := range map[string]string{
		"https://www.ashworth.com/products?x=1": "https://www.ashworth.com/",
		"jpconveyors.com/":                      "https://jpconveyors.com/",
		"http://www.tkf.nl/":                    "http://www.tkf.nl/",
		"":                                      "",
	} {
		if got := site.Root(raw); got != want {
			t.Errorf("Root(%q) = %q, want %q", raw, got, want)
		}
	}
}
