package app

import (
	"context"
	"fmt"
	"slices"

	"github.com/JorisJonkers-dev/findability-explorer/internal/insight/domain"
)

// Limits of the AI search summary.
const (
	// TrendRuns is how many runs the trend goes back.
	TrendRuns = 12
	// TrendCompetitors is how many of the most named competitors the trend follows beside Tribelt.
	TrendCompetitors = 3
	// RankingRows is how many companies the readiness ranking shows.
	RankingRows = 7
)

// CrawlerRow is a crawler with how far tribelt.nl lets it in; Access is empty when Tribelt's
// robots.txt was not read.
type CrawlerRow struct {
	domain.Crawler
	Access domain.BotAccess
}

// AiSearch is the AI search summary.
type AiSearch struct {
	// Goal is the share of answers Tribelt aims to be named in.
	Goal float64
	// Followed are the companies the trend follows: Tribelt, then the most named competitors.
	Followed   []domain.Company
	Trend      []domain.TrendPoint
	Crawlers   []CrawlerRow
	KeepingOut []domain.KeepingOut
	Ranking    []domain.Ranked
	// Sites is how many sites have a recorded AI readiness.
	Sites int
	// Questions and Providers are what the newest run asked; zero before any run.
	Questions int
	Providers int
}

// AiSearch summarises AI search: Tribelt's share of answers over the newest runs against the most
// named competitors, which crawlers tribelt.nl lets in, who keeps them out, and the readiness ranking.
func (s *Service) AiSearch(ctx context.Context) (AiSearch, error) {
	companies, err := s.ports.Directory.Companies(ctx)
	if err != nil {
		return AiSearch{}, fmt.Errorf("list the companies: %w", err)
	}
	runs, err := s.ports.PromptRuns.Runs(ctx, TrendRuns)
	if err != nil {
		return AiSearch{}, fmt.Errorf("the Prompt runs: %w", err)
	}
	readiness, err := s.ports.Readiness.Readiness(ctx)
	if err != nil {
		return AiSearch{}, fmt.Errorf("the AI readiness: %w", err)
	}
	out := AiSearch{
		Goal: s.targets()[domain.TargetShareOfVoice], KeepingOut: domain.KeepOut(companies, readiness),
		Ranking: domain.Ranking(companies, readiness, RankingRows), Sites: len(readiness),
	}
	byID := map[string]domain.Company{}
	for _, c := range companies {
		byID[c.ID] = c
	}
	var ids []string
	if i := slices.IndexFunc(companies, func(c domain.Company) bool { return c.Role == domain.RoleTribelt }); i >= 0 {
		ids = append(ids, companies[i].ID)
	}
	if len(runs) > 0 {
		for _, m := range domain.MostNamed(runs[0], companies, TrendCompetitors) {
			ids = append(ids, m.CompanyID)
		}
		out.Questions, out.Providers = asked(runs[0])
	}
	for _, id := range ids {
		out.Followed = append(out.Followed, byID[id])
	}
	out.Trend = domain.Trend(runs, ids)
	tribelt, scanned := readiness[domain.TribeltID]
	for _, c := range domain.Crawlers() {
		row := CrawlerRow{Crawler: c}
		if scanned {
			row.Access = tribelt.Bots[c.Bot]
		}
		out.Crawlers = append(out.Crawlers, row)
	}
	return out, nil
}

// asked counts the questions and the providers a run's answers come from.
func asked(r domain.Run) (questions, providers int) {
	var q, p []string
	for _, a := range r.Answers {
		if !slices.Contains(q, a.PromptID) {
			q = append(q, a.PromptID)
		}
		if !slices.Contains(p, a.Provider) {
			p = append(p, a.Provider)
		}
	}
	return len(q), len(p)
}
