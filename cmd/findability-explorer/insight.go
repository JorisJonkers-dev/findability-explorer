package main

import (
	"context"
	"slices"
	"time"

	analysisapp "github.com/JorisJonkers-dev/findability-explorer/internal/analysis/app"
	analysisdomain "github.com/JorisJonkers-dev/findability-explorer/internal/analysis/domain"
	companiesapp "github.com/JorisJonkers-dev/findability-explorer/internal/companies/app"
	insightapp "github.com/JorisJonkers-dev/findability-explorer/internal/insight/app"
	insightdomain "github.com/JorisJonkers-dev/findability-explorer/internal/insight/domain"
	prompttestapp "github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/app"
	prompttestdomain "github.com/JorisJonkers-dev/findability-explorer/internal/prompttest/domain"
	terminologyapp "github.com/JorisJonkers-dev/findability-explorer/internal/terminology/app"
)

// newInsight builds the insight use cases on the other contexts, each reached through a port
// that hands its data over by value.
func newInsight(store insightapp.Store, companies *companiesapp.Service, prompts *prompttestapp.Service, analysis *analysisapp.Service,
	terminology *terminologyapp.Service, now func() time.Time,
) *insightapp.Service {
	return insightapp.New(insightapp.Ports{
		Directory:  insightDirectory{companies: companies},
		PromptRuns: promptRuns{prompts: prompts, directory: promptDirectory{companies: companies}},
		Readiness:  readiness{analysis: analysis},
		Terms:      termsUsed{terminology: terminology, analysis: analysis},
		Store:      store,
	}, now)
}

// insightDirectory hands the insight context the companies that are not archived.
type insightDirectory struct {
	companies *companiesapp.Service
}

func (d insightDirectory) Companies(ctx context.Context) ([]insightdomain.Company, error) {
	listed, err := d.companies.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]insightdomain.Company, 0, len(listed))
	for _, l := range listed {
		if l.Archived {
			continue
		}
		c := insightdomain.Company{ID: l.ID, Name: l.Name, Role: insightdomain.Role(l.Role())}
		if l.URL != nil {
			c.Website = l.URL.String()
		}
		for _, b := range l.AllBeltNames() {
			c.BeltNames = append(c.BeltNames, insightdomain.BeltName{Name: b.Name, Family: b.Family, FromScan: b.FromScan})
		}
		out = append(out, c)
	}
	return out, nil
}

// promptRuns hands the insight context the newest Prompt runs' answers that count, with the
// companies each names and cites.
type promptRuns struct {
	prompts   *prompttestapp.Service
	directory promptDirectory
}

func (p promptRuns) Runs(ctx context.Context, limit int) ([]insightdomain.Run, error) {
	runs, err := p.prompts.Counted(ctx, limit)
	if err != nil {
		return nil, err
	}
	directory, err := p.directory.Companies(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]insightdomain.Run, 0, len(runs))
	for _, r := range runs {
		run := insightdomain.Run{ID: r.Run.ID.String(), StartedAt: r.Run.StartedAt}
		for _, a := range r.Answers {
			run.Answers = append(run.Answers, insightdomain.Answer{
				PromptID: a.Prompt.ID, Provider: a.Provider, Mode: insightdomain.Mode(a.Mode), Scope: string(a.Prompt.Scope), Family: string(a.Prompt.Family),
				Named: withTribelt(a.Extraction.NamedCompanies(), a.Extraction.NamesTribelt), First: namedFirst(a.Extraction.Companies),
				Cited: withTribelt(prompttestdomain.CitedCompanies(a.Extraction.Addresses, directory), a.Extraction.CitesTribelt),
			})
		}
		out = append(out, run)
	}
	return out, nil
}

// withTribelt adds Tribelt to the companies an answer names or cites when its reading says it
// names or cites Tribelt, as ADR-0004 counts it: also when the extractor saw Tribelt without a
// name or an address the list matches. The Prompt test counts its share the same way.
func withTribelt(ids []string, tribelt bool) []string {
	if tribelt && !slices.Contains(ids, insightdomain.TribeltID) {
		return append(ids, insightdomain.TribeltID)
	}
	return ids
}

// namedFirst is the company on the list an answer names at its first place, as the Prompt test's
// mentions count it: each company once, where the answer first names it.
func namedFirst(companies []prompttestdomain.Company) string {
	seen := map[string]bool{}
	for _, c := range companies {
		if c.CompetitorID == "" || seen[c.CompetitorID] {
			continue
		}
		seen[c.CompetitorID] = true
		if c.Position == 1 {
			return c.CompetitorID
		}
	}
	return ""
}

// readiness hands the insight context each company's newest AI readiness.
type readiness struct {
	analysis *analysisapp.Service
}

func (r readiness) Readiness(ctx context.Context) (map[string]insightdomain.Readiness, error) {
	rows, err := r.analysis.ReadinessTable(ctx)
	if err != nil {
		return nil, err
	}
	out := map[string]insightdomain.Readiness{}
	for _, row := range rows {
		if row.Scan == nil || row.Scan.Readiness == nil {
			continue
		}
		out[row.Company.ID] = readinessOf(*row.Scan.Readiness, row.Scan.FinishedAt)
	}
	return out, nil
}

func readinessOf(r analysisdomain.Readiness, at time.Time) insightdomain.Readiness {
	out := insightdomain.Readiness{
		Sitemap: r.Sitemap.Files > 0,
		SitemapLanguages: r.Sitemap.Languages, LLMs: r.LLMs.Present, LLMsFull: r.LLMsFull, Markdown: r.Markdown,
		StructuredData: r.StructuredData, ScannedAt: at, Bots: map[string]insightdomain.BotAccess{},
	}
	if c := r.Sitemap.Coverage; c != nil {
		share := c.Share()
		out.SitemapCoverage = &share
	}
	for _, b := range r.Bots {
		out.Bots[b.Bot] = insightdomain.BotAccess(b.Access)
		if b.Access == analysisdomain.AccessBlocked {
			out.Blocked = append(out.Blocked, b.Bot)
		}
	}
	out.Crawlers = insightdomain.AccessOf(r.Robots != analysisdomain.RobotsUnreachable, len(out.Blocked), len(r.Bots))
	return out
}

// termsUsed hands the insight context the terminology map: each term with whether tribelt.nl
// uses it (terminology's SiteUse.TribeltUses) and the companies whose newest scan found it, and
// the companies a finished scan read.
type termsUsed struct {
	terminology *terminologyapp.Service
	analysis    *analysisapp.Service
}

func (t termsUsed) TermMap(ctx context.Context) (insightdomain.TermMap, error) {
	use, err := t.terminology.SiteUse(ctx)
	if err != nil {
		return insightdomain.TermMap{}, err
	}
	rows, err := t.analysis.ReadinessTable(ctx)
	if err != nil {
		return insightdomain.TermMap{}, err
	}
	out := insightdomain.TermMap{TribeltFrom: insightdomain.TermsSource(use.Source())}
	for _, row := range rows {
		if row.Scan != nil {
			out.Scanned = append(out.Scanned, row.Company.ID)
		}
	}
	for _, term := range t.terminology.Terms() {
		out.Terms = append(out.Terms, insightdomain.MapTerm{
			Text: term.Text, Language: string(term.Language), Family: string(term.Family), Tier: string(term.Tier),
			UsedBy: len(term.UsedBy), UsedByTribelt: use.TribeltUses(term), FoundBy: use.FoundBy[term.Key()],
		})
	}
	return out, nil
}
