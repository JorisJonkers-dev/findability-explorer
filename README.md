# Findability Explorer

The course-material repository links this repository as a submodule at `findability-explorer/`.

The interactive companion to the Digital Strategy group report on Tribelt B.V. (University of
Twente, 2026-1A), served privately at `digital-strategy.jorisjonkers.dev`. It shows how metal
conveyor belts are named across companies and languages, and analyses any website's keywords.

- **Keywords** (`/keywords`): search the terminology map (146 terms, 10 languages) from the
  group's desk research of 22 October 2026, with filters by language, intent tier, belt family and
  company. Every filter lives in the URL, so a view can be shared.
- **Competitors** (`/competitors`): the 25 competitors and 3 group companies, with their own name
  for each Tribelt belt family.
- **Analyses** (`/analyses`): crawl up to 30 pages of a website (same site, breadth first, polite,
  robots.txt respected, public addresses only) and report its most prominent keywords, the terms
  from the map it uses, and, with an Anthropic API key, its products matched to Tribelt's families
  by Claude Haiku. This ports and extends
  [RNgo-Sch/digital-strategy-webscraper](https://github.com/RNgo-Sch/digital-strategy-webscraper).

It is built from the estate's [template-go-vue](https://github.com/JorisJonkers-dev/template-go-vue):
a contract-first Go API (`openapi/v1/openapi.yaml`, ogen, sqlc, goose) and a Vue SPA embedded in
one binary. The architecture rules are in [`docs/blueprints/`](docs/blueprints/README.md).

| Path | What it is |
|------|------------|
| `internal/terminology/` | The terminology map: search with folding of case, accents and ß; the embedded seed data |
| `internal/analysis/` | Website analyses: target and robots rules, keyword and known-term extraction, the crawler, the Claude matcher, Postgres persistence |
| `internal/shared/fold/` | The one text normalisation both contexts and the web client use |
| `web/src/features/` | `terminology` and `analyses` pages |

## Run it

```bash
mise install    # the pinned toolchain
task check      # lint, generated-code drift, Go and web tests with coverage, build, secret scan
task e2e        # Playwright with axe against the built binary and the compose Postgres
task dev        # Postgres, the API on :8080, Vite on :5173
```

Environment: `DATABASE_URL` (required), `ADDR` (default `:8080`), `ANTHROPIC_API_KEY` (enables
product matching), `LLM_MONTHLY_TOKENS` (default 2,000,000), `ANALYSES_PER_HOUR` per user
(default 10), `CRAWLER_CONTACT` (put in the crawler's User-Agent), and `DEV_USER` for local runs
only. Never set `DEV_USER` in a deployment.

To update the terminology map, replace both files in `internal/terminology/adapters/seed/data/`
with the research JSON from the course-material repository; the binary refuses to start on a value
the domain does not know.

## Deploying

The app deploys through the estate's [deploy-kit](https://github.com/JorisJonkers-dev/deploy-kit):
`deploy/findability-explorer.project.yml` is its Project Intent and
`deploy/env/findability-explorer/base.env` its Process configuration. Validate both with the
version the estate pins (`estate/toolkit.yml`):

```bash
npx @jorisjonkers-dev/deploy-kit@0.30.0 validate deploy/findability-explorer.project.yml deploy/env
```

(The package is on GitHub Packages: point `@jorisjonkers-dev` at `https://npm.pkg.github.com` with a
token that may read packages.)

**State: deployable, not live.** Merging a release PR builds the multi-arch image to
`ghcr.io/jorisjonkers-dev/findability-explorer` and publishes the signed Intent Fragment, with
`compose: false`, so nothing runs. Going live is one change in each of two places:

1. Vault: `vault kv put secret/findability-explorer-system/anthropic api-key=…` (without it the app
   runs with product matching off).
2. The `estate` repository: an entry in `participants.yml` and the project in
   `platform.intent.yml`'s `handover`; then set `compose: true` in `.github/workflows/release.yml`
   and cut a release. Composition writes `projects/findability-explorer/source.yaml` and Flux
   applies it at `digital-strategy.jorisjonkers.dev`, behind the platform's forward-auth.

**Actions minutes are scarce in this private repo.** CI runs on pull requests that touch code and
by hand, not on pushes to main; run `task check` and `task e2e` locally before you push.
