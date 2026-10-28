# One number per statistic, counted once on the server

The app showed the same idea as different numbers. The plan said tribelt.nl uses 45 search terms
from the map, the home page said 12. The plan counted the map terms tribelt.nl's newest scan found.
The home page counted the terms the desk research credits Tribelt with (`usedBy` naming
Tribelt), in the browser, and later the union of both. The Terms page and Tribelt's company page
counted the union in the browser too. A scan's known terms marked "Tribelt uses it too" by the
research's credit. The Prompt test page counted an answer as naming Tribelt by the reading's
verdict, as ADR-0004 says, while the home page, the plan, Catching up and AI search counted only
answers whose names the list matched to Tribelt. The plan counted citations by address only. And
the targets for search terms (35) and citations (20) were below where Tribelt already stood.

**Each statistic has one definition, computed in one place on the server. The browser only
writes it.**

- **Search terms tribelt.nl uses** are the map's terms its newest finished scan found, each in its
  own language (`lang.Same`, ADR-0008). Before tribelt.nl has a finished scan, the terms the desk
  research credits Tribelt with stand in, and every page says so. The rule is
  `SiteUse.TribeltUses` in `internal/terminology/domain`: the terminology context owns the terms,
  their credits and the scans' sightings. The term search carries it per term (`tribeltUses`,
  `tribeltFrom`), the scans' glossary carries it to their known terms, read again by the map as it
  is now, and the insight context counts it once (`TermMap.TermsUsed`) for the home page, Catching
  up, the plan and the products. Other companies' counts are the map terms their newest scan found.
  The scan is chosen over the research because the question is how many of the map's terms the site
  uses now: a scan measures that, the research was a reading of the sites in October 2026.
- **An answer names Tribelt** when its reading says so, whether or not a name in it matched Tribelt
  on the list (ADR-0004), and **cites tribelt.nl** likewise. A rematch to Tribelt names it too. The
  insight context takes Tribelt into an answer's named and cited companies that way, so the share
  on the home page, Catching up, the plan, AI search and the products is `ShareOfVoice` over the
  same answers of the newest Prompt run, and the Prompt test page's bars, headline and mention lines
  agree with it. The share is sent with the home page's figures rather than divided in the browser.
- **The Twente share** is `TwenteShare`, **citations** are `Citations` (the count) and
  `CitationShare` (its share of the web answers), **readiness checks** are `ReadinessScore`.
- **Competitors** are the competitors on the list now (see the amendment below), counted by
  `CompetitorsOf`.

**A target at or below the value now is met, not a goal.** `Watched.Met` says so; the plan shows
"Met: 45, target was 35", draws no target behind the bar, scales the bar past the value, and asks
for the target to be raised. Stored targets are not rewritten. The defaults, used until an
administrator sets a target, are raised above where Tribelt stood in October 2026: 90 search terms
(twice the 45 of tribelt.nl's scan, about three more terms per belt family in each of the three
languages) and 50 citing web answers (half the 100 web answers of a run). The shares and the
readiness checks stay at 40%, 75% and five of five.

## Consequences

- The new contract fields are additive and optional: `share`, `from`, `tiers` and `listed` on the
  overview, `met` on a watched number, `tribeltTermsFrom` on the standings, the plan and the
  products, and `tribeltUses` and `tribeltFrom` on the term search.
- Catching up's search terms measure now counts only terms the map still holds, and puts Tribelt
  in it before a scan of tribelt.nl, with the research's count and a note saying so.
- The Readiness page's table of eight checks counts the sites passing each check. It is a different
  statistic from the five readiness checks, and its heading says how many sites it counts.
- Tests in `internal/insight/app/agreement_test.go` read the same data through every use case and
  check the pages agree on each statistic.

## Amended, October 2026: every competitor figure counts the list as it is now

The home page and the Competitors page still said "The desk research found 44 competitors" after an
administrator, the candidate review and Haiku had grown the list to 69: `CompetitorsOf` counted only
the competitors the research found, and the families' bars only their belt names. Other pages
counted the list, so one idea was two numbers again.

**Every competitor figure is computed from the competitors on the list now: role competitor, not
archived, researched or added since.** No page keeps a count of the research's list.

- `CompetitorsOf` counts them (`makers` on the overview; `listed` is gone, as it is now the same
  number), and each family's makers by the belt names recorded for them or the products their
  newest scan matched (`AllBeltNames`). The home tile, the Competitors headline ("69 competitors are
  on our list. The same belts, other names.") and All competitors show it; the products' sellers, the
  standings' typical competitor and leader, the readiness ranking and the AI share rows already
  read the same list through the insight Directory.
- **A term's companies are its makers** (`SiteUse.Makers` in `internal/terminology/domain`): the
  competitors on the list that its credits name, by the name the map credits them under, or whose
  newest finished scan found it. A credit to a company no longer on the list, or to Tribelt, makes
  no maker. The term search sends them (`makers`), the products count them (`usedBy`), and the
  search terms page, the heatmap and the company pages list them, in place of the browser merging
  the credits with the scans.
- **The readiness checks are defined once** (`ScoredChecks`, `ReadinessChecks`, `Points` in
  `internal/insight/domain`). The standings, the plan and the AI search summary send how many there
  are (`readinessChecks`), so no page writes "five" itself; the AI search summary counts the sites
  passing each check (`checks`) for the readiness page; a plan target carries its most (`max`); and
  the plan's Product markup check reads the readiness page's rule.
- **A competitor's share of AI answers** on All competitors is the Prompt test mention's `share`,
  computed on the server over the same answers as `ShareOfVoice`.
- The sitemap is measured by its coverage (ADR-0013).
- `internal/insight/app/agreement_test.go` reads one field of Tribelt, its group, the mirror and
  three competitors, one added since, through every use case, and checks the competitor count, the
  family makers, the leaders, the typical competitor, the sitemap coverage, the readiness checks
  and the shares agree.
