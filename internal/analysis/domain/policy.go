package domain

import "strings"

// Bots are the search and AI crawlers whose access a scan records, by their robots.txt token:
// the search indexes, the AI assistants' search and user fetchers, and the training crawlers.
// The report recommends allowing every one of them.
func Bots() []string {
	return []string{
		"Googlebot", "Google-Extended", "Bingbot", "OAI-SearchBot", "ChatGPT-User", "GPTBot",
		"Claude-SearchBot", "ClaudeBot", "PerplexityBot", "CCBot",
	}
}

// Access is how far robots.txt lets a crawler in.
type Access string

const (
	// AccessAllowed means no rule keeps the crawler out of any page.
	AccessAllowed Access = "allowed"
	// AccessPartly means some pages are kept from it, or only some are open to it.
	AccessPartly Access = "partly"
	// AccessBlocked means the whole site is kept from it.
	AccessBlocked Access = "blocked"
)

// BotRule is one crawler's access, and whether robots.txt names it in a group of its own; a
// crawler it does not name follows the * group.
type BotRule struct {
	Bot    string
	Access Access
	Named  bool
}

// FollowsAdvice reports whether the rule follows the report's advice, which is to let every
// search and AI crawler in: a crawler kept out of a few pages still reads the site.
func (r BotRule) FollowsAdvice() bool { return r.Access != AccessBlocked }

// ReadCrawlerPolicy reads robots.txt's rules for each of Bots, in that order. An empty body, as
// for a site without robots.txt, allows every crawler.
func ReadCrawlerPolicy(body string) []BotRule {
	p := robotsParser{groups: map[string][]robotsRule{}}
	for _, line := range strings.Split(body, "\n") {
		p.line(line)
	}
	out := make([]BotRule, 0, len(Bots()))
	for _, bot := range Bots() {
		rules, named := p.groups[strings.ToLower(bot)]
		if !named {
			rules = p.groups["*"]
		}
		out = append(out, BotRule{Bot: bot, Access: accessOf(Robots{rules: rules}), Named: named})
	}
	return out
}

// accessOf tells how far r lets a crawler in: blocked when the home page is closed and no allow
// rule opens another page, partly when a disallow rule closes a page or the home page is closed,
// allowed otherwise. A rule's own path, its wildcards filled in, stands for the pages it covers.
func accessOf(r Robots) Access {
	opens, closes := false, false
	for _, rule := range r.rules {
		allowed := r.Allowed(examplePath(rule.pattern))
		opens = opens || (rule.allow && allowed)
		closes = closes || (!rule.allow && !allowed)
	}
	switch {
	case !r.Allowed("/") && !opens:
		return AccessBlocked
	case !r.Allowed("/") || closes:
		return AccessPartly
	default:
		return AccessAllowed
	}
}

// examplePath is a path a rule's pattern matches: its wildcards filled in, without the end anchor.
func examplePath(pattern string) string {
	return strings.ReplaceAll(strings.TrimSuffix(pattern, "$"), "*", "x")
}
