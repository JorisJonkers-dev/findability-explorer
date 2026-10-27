// Package site holds the rules every context applies to a website address: which hosts may be
// reached, and the host by which two addresses count as one site.
package site

import (
	"net/netip"
	"net/url"
	"slices"
	"strings"

	"golang.org/x/net/publicsuffix"
)

// PublicHost reports whether host may be crawled by name or literal: not empty, not a local or
// internal name, and, when it is an IP literal, a public address.
func PublicHost(host string) bool {
	if addr, err := netip.ParseAddr(host); err == nil {
		return PublicAddr(addr)
	}
	return strings.Contains(host, ".") && !strings.HasSuffix(host, ".localhost") &&
		!strings.HasSuffix(host, ".local") && !strings.HasSuffix(host, ".internal")
}

// PublicAddr reports whether addr is a globally routable unicast address. The crawler refuses to
// connect anywhere else, so a public name that resolves to a private address cannot be used to
// reach the service's own network.
func PublicAddr(addr netip.Addr) bool {
	addr = addr.Unmap()
	cgnat := netip.MustParsePrefix("100.64.0.0/10")
	return addr.IsGlobalUnicast() && !addr.IsPrivate() && !cgnat.Contains(addr)
}

// Host is u's host in lower case without a leading "www.", so www.example.com and example.com
// are one site.
func Host(u *url.URL) string {
	return strings.TrimPrefix(strings.ToLower(u.Hostname()), "www.")
}

// parse reads an address as the answers give it: with or without a scheme ("jpconveyors.com/").
func parse(raw string) *url.URL {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || !strings.Contains(u.Hostname(), ".") || strings.ContainsAny(u.Hostname(), " ") {
		return nil
	}
	return u
}

// Domain is the registered domain an address belongs to, in lower case: "products.bandogrp.com"
// and "bandogrp.com" are one domain, as are "www.x.co.uk" and "x.co.uk". "" for no address.
func Domain(raw string) string {
	u := parse(raw)
	if u == nil {
		return ""
	}
	d, err := publicsuffix.EffectiveTLDPlusOne(strings.ToLower(u.Hostname()))
	if err != nil {
		return ""
	}
	return d
}

// platforms are the names of sites on which many companies have a page — marketplaces,
// directories and social networks — so an address there is no company's own website.
var platforms = []string{ //nolint:gochecknoglobals // a constant list
	"alibaba", "aliexpress", "amazon", "blogspot", "ebay", "europages", "facebook", "github",
	"globalsources", "google", "indiamart", "instagram", "issuu", "kompass", "linkedin",
	"made-in-china", "medium", "pinterest", "reddit", "scribd", "thomasnet", "tradeindia",
	"twitter", "wikipedia", "wlw", "wordpress", "yellowpages", "youtube",
}

// CompanyDomain is the domain of a company's own website the address belongs to, or "" when it
// is on a platform many companies share, or no address at all.
func CompanyDomain(raw string) string {
	d := Domain(raw)
	name, _, _ := strings.Cut(d, ".")
	if d == "" || slices.Contains(platforms, name) {
		return ""
	}
	return d
}

// Root is the address of the website's home page: its scheme (https when none) and host.
func Root(raw string) string {
	u := parse(raw)
	if u == nil {
		return ""
	}
	return u.Scheme + "://" + strings.ToLower(u.Host) + "/"
}
