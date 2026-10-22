// Package site holds the rules every context applies to a website address: which hosts may be
// reached, and the host by which two addresses count as one site.
package site

import (
	"net/netip"
	"net/url"
	"strings"
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
