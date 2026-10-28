# AI readiness is scored out of eight checks

The readiness score was out of five: every AI crawler let in (half when some are kept out), a
sitemap, llms.txt, llms-full.txt and Markdown pages. The glossary defines AI readiness as more than
that, and the app already read the rest (Organization markup, the language versions and the
sitemap's languages) but showed it beside the score. A site missing all three still scored five of
five, so the score did not say whether the site was ready.

**The score counts what the glossary calls AI readiness: eight checks** (`ScoredChecks` in
`internal/insight/domain`). The five above, plus:

- **Structured data**: the home page has Organization markup. Product markup stays on the pages as
  a check beside the score, and is not scored.
- **Language versions**: the site offers its pages in Dutch, German and English (`languages`).
- **Sitemap in every language**: the sitemaps list pages in all three languages (`sitemapLanguages`).

The crawler check is unchanged. It covers the search crawlers and the training crawlers (GPTBot,
ClaudeBot, Google-Extended, and CCBot), and its label now says so.

- **The total has one source.** `ReadinessChecks()` in Go is sent as `readinessChecks` on the
  standings, the plan and the AI search summary, and the web reads it from there. No page writes
  the number.
- **Scores change without a rescan**, since the facts they need are already stored with each scan.
- **The plan target** `readinessChecks` is every check, so its default moves from five to eight. It
  is not stored unless an administrator set it, and one who set it keeps their value.

# Web citations are a share of a run's web answers

The plan's target `citations` counted a run's web answers that cite tribelt.nl (28 of 50), which
cannot be compared between runs that read different numbers of web answers. The standings already
showed `citationShare`, a share of the same answers.

**The number to watch is the share of a run's web answers that cite tribelt.nl, with a target of
50%.** The target is `citationShare`, from 0 to 1, like the other shares. The migration
`00026_citation_share_target.sql` turns an existing target of 50 into 50%.
