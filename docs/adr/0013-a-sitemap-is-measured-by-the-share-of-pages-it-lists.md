# A sitemap is measured by the share of the site's pages it lists

Catching up, the plan and the AI readiness page compared sites on the number of addresses their
sitemaps list (`sitemapPages`). A higher number did not mean a better sitemap: a site with 9,000
pages and a sitemap that leaves half of them out beat a site with 120 pages and a sitemap that
lists all of them. What a buyer and a search engine need from a sitemap is that it lists the pages
there are, so the number says more about the size of the site than about its sitemap.

**A sitemap is measured by its coverage: of the different pages a scan read on the site, the share
its sitemaps list.** `CoverageOf` in `internal/analysis/domain` is the one place this rule lives.
It is a share from 0 to 1, and the measure is `sitemapCoverage`.

- **The denominator is the pages the scan read.** A scan reads at most its page limit, starting at
  the home page and following the site's links and the sitemaps' addresses, belt pages first. These
  are the site's pages the app knows to exist: each answered as an HTML page, after redirects.
  The site's true number of pages is unknown, so it cannot be the denominator. The sitemaps'
  address count cannot be either, as a share of itself says nothing.
- **The numerator is the pages read whose address a sitemap lists.** Addresses compare as the crawl
  compares them: "www." aside, without the query, the fragment or a trailing slash, with the path
  decoded. The sitemaps are those the readiness check reads, at most ten files of at most 5,000
  addresses each, so a very large sitemap may list a page past what is read.
- **The crawl also reads pages because the sitemaps list them**, so the pages read lean towards the
  ones a sitemap lists. The share is therefore an upper estimate for a site whose sitemap is
  incomplete. Counting only the pages a link on the site leads to was considered: it needs the
  links of every page kept with the scan and does not remove the lean, since the sitemaps' pages
  still take part of the page limit. Every site is measured by the same rule, so the shares can be
  compared, though none of them is exact.
- **A site without a sitemap lists none of its pages**: its coverage is 0, not unknown.

**It is computed when the pages are known.** A scan reads the readiness files before its crawl, so
a site that turns the crawl away still shows its crawler policy; the coverage is recorded on that
readiness once the crawl is done (`SaveSitemapCoverage`). A check of the AI readiness again reads
the sitemaps again and measures them against the pages the scan read, as it crawls none.

**A readiness recorded before this has no coverage, and no page shows one for it.** Migration 00024
adds `sitemap_pages_read` and `sitemap_pages_listed` to `analysis_readiness`, both null for those
and for a scan that read no page. The standings leave such a site out of the typical competitor and
the leader, and the pages say "Not measured yet. Scan the site again." A new scan, or a check of the
readiness again, measures it.

## Consequences

- The `StandingMeasure` value `sitemapPages` is now `sitemapCoverage`, a share written as a
  percentage. No target watches it, so no stored target moves.
- `SitemapSummary` in the contract gains an optional `coverage` (`pages`, `listed`, `share`). The
  number of addresses stays in the scan's readiness, as a fact about the sitemap, but nothing ranks
  or compares sites by it.
- The AI readiness table's Sitemap column shows the share and sorts by it.
- The plan's "A sitemap for each language" says how much of tribelt.nl its sitemaps list and who
  lists the most.
