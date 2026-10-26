package domain_test

import (
	"testing"

	"github.com/JorisJonkers-dev/findability-explorer/internal/analysis/domain"
)

// policyOf returns the rule ReadCrawlerPolicy finds for the crawler token.
func policyOf(t *testing.T, rules []domain.BotRule, token string) domain.BotRule {
	t.Helper()
	for _, r := range rules {
		if r.Bot == token {
			return r
		}
	}
	t.Fatalf("no rule for %s in %v", token, rules)
	return domain.BotRule{}
}

func TestCrawlerPolicyCoversEverySearchAndAICrawler(t *testing.T) {
	rules := domain.ReadCrawlerPolicy("")
	want := []string{
		"Googlebot", "Google-Extended", "Bingbot", "OAI-SearchBot", "ChatGPT-User", "GPTBot",
		"Claude-SearchBot", "ClaudeBot", "PerplexityBot", "CCBot",
	}
	if len(rules) != len(want) {
		t.Fatalf("got %d rules, want %d", len(rules), len(want))
	}
	for i, token := range want {
		if rules[i].Bot != token || rules[i].Access != domain.AccessAllowed || rules[i].Named {
			t.Errorf("rule %d = %+v, want %s allowed and not mentioned", i, rules[i], token)
		}
	}
}

func TestCrawlerPolicyReadsEachCrawlersOwnGroup(t *testing.T) {
	rules := domain.ReadCrawlerPolicy(`User-agent: *
Allow: /
Disallow: /privacyverklaring

# A group may name several crawlers.
User-agent: GPTBot
User-agent: ccbot
Disallow: /

User-agent: ClaudeBot
Disallow: /
Allow: /products

User-agent: OAI-SearchBot
Allow: /

User-agent: Google-Extended
Disallow: /
Allow: /
`)
	cases := []struct {
		bot    string
		access domain.Access
		named  bool
	}{
		{"GPTBot", domain.AccessBlocked, true},
		{"CCBot", domain.AccessBlocked, true}, // agent names match ignoring case
		{"ClaudeBot", domain.AccessPartly, true},
		{"OAI-SearchBot", domain.AccessAllowed, true},
		{"Google-Extended", domain.AccessAllowed, true}, // an allow rule as long as the disallow wins
		{"Googlebot", domain.AccessPartly, false},       // the * group, which keeps one page out
	}
	for _, c := range cases {
		got := policyOf(t, rules, c.bot)
		if got.Access != c.access || got.Named != c.named {
			t.Errorf("%s = %+v, want %s, named %v", c.bot, got, c.access, c.named)
		}
	}
}

func TestCrawlerPolicyAppliesWildcardsAndTheStarGroupsBlock(t *testing.T) {
	rules := domain.ReadCrawlerPolicy("User-agent: *\nDisallow: /\n\nUser-agent: Bingbot\nDisallow: /*.pdf$\nDisallow: /*?\n")
	if got := policyOf(t, rules, "Bingbot"); got.Access != domain.AccessPartly || !got.Named {
		t.Errorf("Bingbot = %+v, want partly blocked by its own wildcard rules", got)
	}
	if got := policyOf(t, rules, "PerplexityBot"); got.Access != domain.AccessBlocked || got.Named {
		t.Errorf("PerplexityBot = %+v, want blocked by the * group", got)
	}
}

func TestCrawlerFollowsTheAdviceUnlessBlocked(t *testing.T) {
	for access, want := range map[domain.Access]bool{
		domain.AccessAllowed: true, domain.AccessPartly: true, domain.AccessBlocked: false,
	} {
		if got := (domain.BotRule{Access: access}).FollowsAdvice(); got != want {
			t.Errorf("%s follows the advice = %v, want %v", access, got, want)
		}
	}
}
