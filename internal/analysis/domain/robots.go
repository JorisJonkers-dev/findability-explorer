package domain

import (
	"strings"
)

// Robots is the part of a site's robots.txt that applies to one crawler. The zero value allows
// every path, which is the rule for a site without a robots.txt.
type Robots struct {
	rules []robotsRule
}

type robotsRule struct {
	allow  bool
	prefix string
}

// ParseRobots reads robots.txt and keeps the group for agent, or the * group when no group names
// agent. Agent matching ignores case, as the standard (RFC 9309) requires. Wildcards inside a rule
// are not supported: such a rule is read up to its first *, which only ever blocks more.
func ParseRobots(body, agent string) Robots {
	p := robotsParser{groups: map[string][]robotsRule{}}
	for _, line := range strings.Split(body, "\n") {
		p.line(line)
	}
	if rules, ok := p.groups[strings.ToLower(agent)]; ok {
		return Robots{rules: rules}
	}
	return Robots{rules: p.groups["*"]}
}

// robotsParser collects the rules of every group: consecutive user-agent lines open a group,
// and the allow and disallow lines after them belong to every agent the group names.
type robotsParser struct {
	groups  map[string][]robotsRule
	current []string
	inRules bool
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
	case "user-agent":
		if p.inRules {
			p.current, p.inRules = nil, false
		}
		p.current = append(p.current, strings.ToLower(value))
	case "allow", "disallow":
		p.inRules = true
		prefix, _, _ := strings.Cut(value, "*")
		if prefix == "" && key == "disallow" {
			return // an empty Disallow allows everything
		}
		for _, a := range p.current {
			p.groups[a] = append(p.groups[a], robotsRule{allow: key == "allow", prefix: strings.TrimSuffix(prefix, "$")})
		}
	}
}

// Allowed reports whether path may be fetched: the longest matching rule wins, and on a tie the
// allow rule.
func (r Robots) Allowed(path string) bool {
	best, allowed := -1, true
	for _, rule := range r.rules {
		if !strings.HasPrefix(path, rule.prefix) {
			continue
		}
		if n := len(rule.prefix); n > best || (n == best && rule.allow) {
			best, allowed = n, rule.allow
		}
	}
	return allowed
}
