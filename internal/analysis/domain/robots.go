package domain

import (
	"strings"
)

// Robots is the part of a site's robots.txt that applies to one crawler. The zero value allows
// every path, which is the rule for a site without a robots.txt.
type Robots struct {
	rules    []robotsRule
	sitemaps []string
}

// Sitemaps are the sitemap addresses robots.txt lists; they apply to every crawler.
func (r Robots) Sitemaps() []string { return r.sitemaps }

type robotsRule struct {
	allow   bool
	pattern string // the rule's path, with * for any run of characters and a final $ for the end
}

// ParseRobots reads robots.txt and keeps the group for agent, or the * group when no group names
// agent. Agent matching ignores case, and rules may use * and a final $, as the standard
// (RFC 9309) defines.
func ParseRobots(body, agent string) Robots {
	p := robotsParser{groups: map[string][]robotsRule{}}
	for _, line := range strings.Split(body, "\n") {
		p.line(line)
	}
	if rules, ok := p.groups[strings.ToLower(agent)]; ok {
		return Robots{rules: rules, sitemaps: p.sitemaps}
	}
	return Robots{rules: p.groups["*"], sitemaps: p.sitemaps}
}

// robotsParser collects the rules of every group: consecutive user-agent lines open a group,
// and the allow and disallow lines after them belong to every agent the group names.
type robotsParser struct {
	groups   map[string][]robotsRule
	current  []string
	inRules  bool
	sitemaps []string
}

func (p *robotsParser) line(line string) {
	if i := strings.IndexByte(line, '#'); i >= 0 {
		line = line[:i]
	}
	key, value, ok := strings.Cut(line, ":")
	if !ok {
		return
	}
	key, value = strings.ToLower(strings.TrimSpace(key)), strings.TrimSpace(value)
	switch key {
	case "sitemap":
		if value != "" {
			p.sitemaps = append(p.sitemaps, value)
		}
	case "user-agent":
		if p.inRules {
			p.current, p.inRules = nil, false
		}
		p.current = append(p.current, strings.ToLower(value))
	case "allow", "disallow":
		p.inRules = true
		if value == "" {
			return // an empty rule matches nothing
		}
		for _, a := range p.current {
			p.groups[a] = append(p.groups[a], robotsRule{allow: key == "allow", pattern: value})
		}
	}
}

// Allowed reports whether path (with its query, if any) may be fetched: the matching rule with the
// longest pattern wins, and on a tie the allow rule.
func (r Robots) Allowed(path string) bool {
	best, allowed := -1, true
	for _, rule := range r.rules {
		if !matches(rule.pattern, path) {
			continue
		}
		if n := len(rule.pattern); n > best || (n == best && rule.allow) {
			best, allowed = n, rule.allow
		}
	}
	return allowed
}

// matches reports whether path starts with pattern, where * matches any run of characters and a
// final $ means the path ends there.
func matches(pattern, path string) bool {
	anchored := strings.HasSuffix(pattern, "$")
	parts := strings.Split(strings.TrimSuffix(pattern, "$"), "*")
	if !strings.HasPrefix(path, parts[0]) {
		return false
	}
	rest := path[len(parts[0]):]
	for _, part := range parts[1:] {
		i := strings.Index(rest, part)
		if i < 0 {
			return false
		}
		rest = rest[i+len(part):]
	}
	if !anchored {
		return true
	}
	if len(parts) == 1 {
		return rest == ""
	}
	// With a wildcard before the anchor, the last part must end the path.
	return strings.HasSuffix(path, parts[len(parts)-1])
}
