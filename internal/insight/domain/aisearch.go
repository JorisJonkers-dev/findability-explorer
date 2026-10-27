package domain

import (
	"cmp"
	"slices"
	"time"
)

// Feeds is what a crawler's visits feed: answers AI assistants give today, or what future models know.
type Feeds string

const (
	FeedsToday  Feeds = "today"
	FeedsFuture Feeds = "future"
)

// Crawler is one search or AI crawler a scan records: its robots.txt token, the lab behind it,
// what letting it in gets Tribelt, and what it feeds.
type Crawler struct {
	Bot   string
	Lab   string
	Gets  string
	Feeds Feeds
}

// Crawlers are the crawlers a scan records, those that feed answers today first.
func Crawlers() []Crawler {
	return []Crawler{
		{"Googlebot", "Google", "Google Search, AI Overviews and Gemini", FeedsToday},
		{"Bingbot", "Microsoft", "Bing, Copilot and part of ChatGPT search", FeedsToday},
		{"OAI-SearchBot", "OpenAI", "Links in ChatGPT search answers", FeedsToday},
		{"ChatGPT-User", "OpenAI", "Pages ChatGPT opens while it answers", FeedsToday},
		{"Claude-SearchBot", "Anthropic", "Links in Claude answers", FeedsToday},
		{"PerplexityBot", "Perplexity", "Sources in Perplexity answers", FeedsToday},
		{"GPTBot", "OpenAI", "What future ChatGPT models know", FeedsFuture},
		{"ClaudeBot", "Anthropic", "What future Claude models know", FeedsFuture},
		{"Google-Extended", "Google", "What future Gemini models know", FeedsFuture},
		{"CCBot", "Common Crawl", "Open data many models learn from", FeedsFuture},
	}
}

// CompanyShare is one company's share of a run's answers.
type CompanyShare struct {
	CompanyID string
	Share     float64
}

// TrendPoint is one run: when it started, and the share of answers naming each company followed.
type TrendPoint struct {
	RunID     string
	StartedAt time.Time
	Shares    []CompanyShare
}

// MostNamed are the at most n competitors with the highest share of the run's answers, highest
// first, then by name; a competitor no answer names is left out.
func MostNamed(r Run, companies []Company, n int) []CompanyShare {
	var out []CompanyShare
	for _, c := range companies {
		if c.Role != RoleCompetitor {
			continue
		}
		if v, _ := ShareOfVoice(r.Answers, c.ID); v > 0 {
			out = append(out, CompanyShare{CompanyID: c.ID, Share: v})
		}
	}
	names := nameOf(companies)
	slices.SortStableFunc(out, func(a, b CompanyShare) int {
		return cmp.Or(cmp.Compare(b.Share, a.Share), byName(names[a.CompanyID], names[b.CompanyID]))
	})
	return out[:min(n, len(out))]
}

// Trend is each run's share of answers naming each of ids, oldest run first.
func Trend(runs []Run, ids []string) []TrendPoint {
	out := make([]TrendPoint, 0, len(runs))
	for i := len(runs) - 1; i >= 0; i-- {
		p := TrendPoint{RunID: runs[i].ID, StartedAt: runs[i].StartedAt}
		for _, id := range ids {
			v, _ := ShareOfVoice(runs[i].Answers, id)
			p.Shares = append(p.Shares, CompanyShare{CompanyID: id, Share: v})
		}
		out = append(out, p)
	}
	return out
}

// KeepingOut is a competitor whose robots.txt keeps crawlers out of the whole site.
type KeepingOut struct {
	CompanyID string
	Name      string
	Blocked   []string
}

// KeepOut lists the competitors that keep any crawler out, most crawlers first, then by name.
func KeepOut(companies []Company, readiness map[string]Readiness) []KeepingOut {
	var out []KeepingOut
	for _, c := range companies {
		if r, ok := readiness[c.ID]; ok && c.Role == RoleCompetitor && len(r.Blocked) > 0 {
			out = append(out, KeepingOut{CompanyID: c.ID, Name: c.Name, Blocked: r.Blocked})
		}
	}
	slices.SortStableFunc(out, func(a, b KeepingOut) int {
		return cmp.Or(cmp.Compare(len(b.Blocked), len(a.Blocked)), byName(a.Name, b.Name))
	})
	return out
}

// Ranked is a company in the readiness ranking.
type Ranked struct {
	CompanyID string
	Name      string
	Role      Role
	Score     float64
}

// Ranking is Tribelt among the best-prepared competitors, at most n rows, highest score first,
// then by name: Tribelt takes the last row when its score does not reach the first n.
func Ranking(companies []Company, readiness map[string]Readiness, n int) []Ranked {
	var all []Ranked
	for _, c := range companies {
		if r, ok := readiness[c.ID]; ok && (c.Role == RoleCompetitor || c.Role == RoleTribelt) {
			all = append(all, Ranked{CompanyID: c.ID, Name: c.Name, Role: c.Role, Score: r.ReadinessScore()})
		}
	}
	slices.SortStableFunc(all, func(a, b Ranked) int { return cmp.Or(cmp.Compare(b.Score, a.Score), byName(a.Name, b.Name)) })
	if len(all) <= n {
		return all
	}
	top := slices.Clone(all[:n])
	if !slices.ContainsFunc(top, func(r Ranked) bool { return r.Role == RoleTribelt }) {
		if i := slices.IndexFunc(all, func(r Ranked) bool { return r.Role == RoleTribelt }); i >= 0 {
			top[n-1] = all[i]
		}
	}
	return top
}

func nameOf(companies []Company) map[string]string {
	out := make(map[string]string, len(companies))
	for _, c := range companies {
		out[c.ID] = c.Name
	}
	return out
}
