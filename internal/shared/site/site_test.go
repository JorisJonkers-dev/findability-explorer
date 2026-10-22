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
