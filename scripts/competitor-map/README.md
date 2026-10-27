# Competitor map

`competitor-map.json` says, for every company in the production competitor review queue of
27 October 2026, whether it competes with Tribelt, with the evidence. `apply.ts` applies it to the
review through the admin API: it adds the competitors and dismisses the rest.

The map was made because Claude Haiku's check left most of the queue undecided. Its notes were used
as hints only, and many turned out wrong.

## Who counts as a competitor

The definition is Tribelt's target market (`TargetMarket` in `internal/prompttest/domain/market.go`,
ADR-0009). Tribelt makes metal conveyor belts and their drive elements in its own factory: eyelink,
open and closed spiral, Tri-Flex curve-going and spiral-tower, chain, ladder, plate link,
honeycomb, interlaced wire and woven baking belts, with sprockets and drums. They are for process
conveying in heat up to about 1,150 °C, in cold, in wet and hygienic food steps, and for heavy,
sharp or abrasive loads in recycling, waste and water filtration. It sells to food producers, to
oven, freezer and line builders, and to industry, mainly in the Netherlands and Germany.

A competitor makes metal conveyor belts of these kinds for these uses, or has them made and sells
them under its own name. A maker anywhere in the world counts if it sells into Europe. Solid steel
process belts count too, as Berndorf and IPCO on the list do.

These are not competitors.

- Makers of rubber, textile or steel-cord belting for mining and bulk.
- Makers of fabric, PVC, PU or PTFE belts.
- Makers of plastic modular belts, unless they also make metal process belts.
- Machine and system builders that buy their belts, such as oven, furnace and freezer builders.
- Dealers, traders, installers and service firms, unless they make or brand metal belts.
- Makers of fence, sieve, filter or architectural mesh.
- Unrelated businesses, directories and publishers.
- Names that do not exist, from bad AI answers. These have `nonexistent: true`.

## Method

The queue was copied by the owner from Settings, Competitor review, To review. The copy holds 381
companies, though the page said 383. For each one the home page was fetched with curl, with a
20 second timeout and one or a few requests per company, and product pages were read where the
home page was not clear. Where there was no website, the website did not resolve, or the name was
ambiguous, web search and directory listings (wlw.de, Europages, IndiaMART, Made-in-China) were
used. A verdict was never taken from the name alone. Many websites the answers gave do not exist;
that is recorded as evidence.

Each entry has these fields.

- `verdict` is `competitor`, `not` or `unclear`. Unclear is used only where the evidence ran out,
  such as a site under maintenance.
- `confidence` is high, medium or low.
- `makes` says what the company really makes, `reason` why that does or does not compete, and
  `evidence` gives the pages with a short quote.
- `website` is the home page to add a competitor with. It is often not a website the answers gave.
- `duplicateOf` names a company already on the competitor list that the entry is the same as.
  These are left for a person to merge, since adding them would list the company twice.

The 9 added and 79 dismissed candidates were not in the copy, so the map does not cover them. The
script checks them against the map where it can and lists the rest for a person.

Researched on 27 October 2026.

## Running the script

It needs Node 24 or later, which runs TypeScript as it is, and no packages. It reads two variables.

- `FE_BASE_URL` is the app's address.
- `FE_COOKIE` is the value of an administrator's `fe_session` cookie, copied from the browser
  after signing in. It is sent only to `FE_BASE_URL` and never printed.

First a dry run, which only reports.

```sh
FE_BASE_URL=https://digital-strategy.jorisjonkers.dev FE_COOKIE='<fe_session value>' \
  mise exec -- node scripts/competitor-map/apply.ts
```

Then the same with `--apply` to act.

```sh
FE_BASE_URL=https://digital-strategy.jorisjonkers.dev FE_COOKIE='<fe_session value>' \
  mise exec -- node scripts/competitor-map/apply.ts --apply
```

What it does is this.

- It reads the waiting, added and dismissed candidates and the company list.
- It matches each candidate to an entry by website host first, then by its folded names (no case,
  accents, punctuation or legal form). A match to entries with different verdicts is no match.
- A waiting competitor is added with the entry's name and website. A waiting non-competitor is
  dismissed, in bulk. Unclear ones, duplicates, competitors without a website or with a website
  already on the list, and unmatched ones are left waiting and listed.
- Decided candidates the map disagrees with are listed. With `--fix-decided` it undoes such a
  decision and then adds or dismisses as the map says. Duplicates are never changed.
- It goes on after a single failure, logs each action, and exits with 1 if any failed.

Running it twice changes nothing the second time. `--map <file>` reads another map.

## Tests

```sh
mise exec -- node --test scripts/competitor-map/apply.test.ts
```

The tests run the script against a small fake of the admin API, and check that the map file is well
formed.
