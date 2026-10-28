package main

import (
	"testing"
	"time"

	analysisdomain "github.com/JorisJonkers-dev/findability-explorer/internal/analysis/domain"
	insightdomain "github.com/JorisJonkers-dev/findability-explorer/internal/insight/domain"
	prompttestdomain "github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain"
)

func TestNamedFirstIsTheCompanyOnTheListAtTheFirstPlace(t *testing.T) {
	for _, tc := range []struct {
		name      string
		companies []prompttestdomain.Company
		want      string
	}{
		{"first", []prompttestdomain.Company{{CompetitorID: "a", Position: 1}, {CompetitorID: "b", Position: 2}}, "a"},
		{"off the list", []prompttestdomain.Company{{Name: "Other", Position: 1}, {CompetitorID: "b", Position: 2}}, ""},
		{"named again", []prompttestdomain.Company{{CompetitorID: "a", Position: 2}, {CompetitorID: "a", Position: 1}}, ""},
		{"none", nil, ""},
	} {
		if got := namedFirst(tc.companies); got != tc.want {
			t.Errorf("%s: %q, want %q", tc.name, got, tc.want)
		}
	}
}

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
		Robots: analysisdomain.RobotsFound, Bots: rules("CCBot"), Sitemap: analysisdomain.SitemapSummary{Files: 2, Addresses: 124, Coverage: &analysisdomain.SitemapCoverage{Pages: 40, Listed: 30}},
		LLMs: analysisdomain.LLMsFacts{Present: true}, Markdown: true,
	}, at)
	if !got.Sitemap || got.SitemapCoverage == nil || *got.SitemapCoverage != 0.75 || !got.LLMs || got.LLMsFull || !got.Markdown || len(got.Blocked) != 1 || got.Blocked[0] != "CCBot" ||
		got.Bots["CCBot"] != insightdomain.BotBlocked || got.Bots["GPTBot"] != insightdomain.BotAllowed {
		t.Errorf("readiness = %+v", got)
	}
}
