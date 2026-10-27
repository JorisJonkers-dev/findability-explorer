package main

import (
	"testing"
	"time"

	analysisdomain "github.com/JorisJonkers-dev/findability-explorer/internal/analysis/domain"
	insightdomain "github.com/JorisJonkers-dev/findability-explorer/internal/insight/domain"
)

func TestReadinessOfSaysHowFarTheCrawlersAreLetIn(t *testing.T) {
	at := time.Date(2026, 11, 2, 9, 0, 0, 0, time.UTC)
	rules := func(blocked ...string) []analysisdomain.BotRule {
		var out []analysisdomain.BotRule
		for _, bot := range analysisdomain.Bots() {
			access := analysisdomain.AccessAllowed
			for _, b := range blocked {
				if b == bot {
					access = analysisdomain.AccessBlocked
				}
			}
			out = append(out, analysisdomain.BotRule{Bot: bot, Access: access})
		}
		return out
	}
	for _, tc := range []struct {
		name string
		in   analysisdomain.Readiness
		want insightdomain.Access
	}{
		{"open", analysisdomain.Readiness{Robots: analysisdomain.RobotsFound, Bots: rules()}, insightdomain.AccessOpen},
		{"some", analysisdomain.Readiness{Robots: analysisdomain.RobotsFound, Bots: rules("GPTBot")}, insightdomain.AccessSome},
		{"closed", analysisdomain.Readiness{Robots: analysisdomain.RobotsFound, Bots: rules(analysisdomain.Bots()...)}, insightdomain.AccessClosed},
		{"unreadable", analysisdomain.Readiness{Robots: analysisdomain.RobotsUnreachable}, insightdomain.AccessUnknown},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := readinessOf(tc.in, at); got.Crawlers != tc.want || !got.ScannedAt.Equal(at) {
				t.Errorf("crawlers = %s, want %s", got.Crawlers, tc.want)
			}
		})
	}
	got := readinessOf(analysisdomain.Readiness{
		Robots: analysisdomain.RobotsFound, Bots: rules("CCBot"), Sitemap: analysisdomain.SitemapSummary{Files: 2, Addresses: 124},
		LLMs: analysisdomain.LLMsFacts{Present: true}, Markdown: true,
	}, at)
	if !got.Sitemap || got.SitemapPages != 124 || !got.LLMs || got.LLMsFull || !got.Markdown || len(got.Blocked) != 1 || got.Blocked[0] != "CCBot" {
		t.Errorf("readiness = %+v", got)
	}
}
