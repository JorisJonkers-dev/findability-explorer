# Findability Explorer

A research tool that shows how a product is named and found online: on competitors' websites, in
the search terms buyers use, and in the answers of AI assistants. It was built for Tribelt B.V., a
Dutch maker of metal conveyor belts, and studies that market in Dutch, German and English.

It is one Go binary with an embedded Vue app and a Postgres database. Every visitor signs in, and
administrators start the scans and change the data.

## What it does

| Page | What you see there |
|---|---|
| **Search terms** (`/terms`) | The terminology map: the ways to search for a metal conveyor belt, which competitors use each term, and the term Tribelt uses for the same thing. Filters by language, kind of search, belt family and company live in the URL, so a view can be shared |
| **Competitors** (`/competitors`) | The competitors and group companies, their names for each belt family, and their scan history. Administrators add, edit, archive and re-scan them |
| **Scans** (`/scan/:id`) | A crawl of up to 100 pages of a company's site. It shows the most prominent search terms, the terms from the map the site uses and, when a language model is set up, the company's products matched to Tribelt's belt families |
| **Company history** (`/competitors/:id/history`) | A company's terms from its latest scan, its numbered scans, and what changed between any two of them |
| **AI readiness** (`/ai-search/readiness`) | What each site tells search engines and AI assistants: `robots.txt` rules per AI crawler, sitemaps, `llms.txt`, structured data, language versions and tech stack, each held against the report's advice |
| **Prompt test** (`/ai-search/prompt-test`) | A fixed set of buyer questions asked of the cheapest model of each large provider through OpenRouter, with and without web search. It shows how often each company is named (AI share of voice), how often its site is cited, and what the answers get wrong about Tribelt |
| **Settings** (`/settings/…`, administrators) | Language-model keys and their spend, the token and dollar budgets, the Prompt test's schedule and fact sheet, the review of new terms and of companies the answers name |

The crawler identifies itself (`FindabilityExplorer/1.0`, with a contact address), respects
`robots.txt`, waits 500 ms between requests and only reads public addresses. A site that turns it
away is marked as blocked; the crawler never poses as a browser.

## Run it on your machine

You need [mise](https://mise.jdx.dev) and Docker. mise installs the exact Go, Node, pnpm, Task and
linter versions from `mise.toml`.

```bash
git clone https://github.com/JorisJonkers-dev/findability-explorer.git
cd findability-explorer
mise install
task dev
```

Open <http://localhost:5173>. `task dev` starts Postgres in Docker (on port 55432), the API on
`:8080` and Vite with hot reload on `:5173`. You are signed in as `dev`, an administrator, with no
sign-in server needed. The database schema is migrated when the API starts.

Some features need a language model:

- **Product matching** in scans: add an Anthropic, OpenRouter or Mistral key under Settings, or
  start with `ANTHROPIC_API_KEY` set.
- **The Prompt test:** add an OpenRouter key under Settings › Prompt test.

Without `SESSION_KEY` a local run seals the keys you add with a random key, so they are unreadable
after a restart. Set `SESSION_KEY` to any string of 32 characters or more to keep them.

Everything else, the map, the companies and scans, works without any key.

## Everyday commands

`task` lists every target. These are the ones you use:

| Command | What it does |
|---|---|
| `task dev` | Run the app locally as described above |
| `task gen` | Regenerate the server, the SQL queries and the web client after changing the API contract or a query |
| `task test` | Go tests (with Postgres in Docker through testcontainers) and web unit tests, each with its coverage gate |
| `task check` | Everything CI checks on a pull request: linters, generated-code drift, tests, build and a secret scan |
| `task e2e` | Build the binary and run the Playwright suite against it, with axe accessibility checks, at desktop and phone width |
| `task fix` | Apply the formatters and the linters' autofixes |

Run `task check` and `task e2e` before you open a pull request; CI runs the same two.

## How it is built

```
openapi/v1/openapi.yaml ──ogen──▶ internal/platform/oas/        (the Go server interface)
                         ──hey-api──▶ web/src/infrastructure/api/ (the typed web client)
db/queries/*.sql ──sqlc──▶ internal/platform/pg/queries/
```

**The API contract comes first.** Every endpoint is described in `openapi/v1/openapi.yaml`, and the
Go server interface and the web client are generated from it. The three generated directories
above are never edited by hand: change the contract or the query, then run `task gen`.

**The Go code is split into contexts**, each with the same three layers:

- `domain/`: the rules, in plain Go with no I/O.
- `app/`: the use cases, which reach the outside world only through interfaces they declare.
- `adapters/`: Postgres, HTTP handlers, the crawler and the language-model clients.

A linter (depguard) keeps `domain` and `app` free of adapter imports, and both are held to 100% test
coverage.

| Context | What it owns |
|---|---|
| `internal/terminology/` | The terminology map, its seed data, and the review of new terms |
| `internal/companies/` | The researched companies, the ones administrators add, and their edits |
| `internal/analysis/` | Scans: robots and sitemap rules, the crawl, term extraction, product matching, AI readiness, company profiles, comparisons, re-scans |
| `internal/prompttest/` | The Prompt test: the prompt set, model choice, runs, schedule, budget, reading the answers, competitor candidates |
| `internal/llm/` | Language-model keys (sealed in Postgres), the chosen model, budgets, usage reports, and the Anthropic, OpenRouter and Mistral clients |
| `internal/identity/` | Signing in, sessions and roles |
| `internal/shared/` | Small shared pieces: text folding, the three focus languages, sealing secrets, site addresses |
| `internal/server/`, `cmd/findability-explorer/` | Wiring: configuration, the HTTP server, and the start-up of every context |

**The web app** in `web/src/` follows the same idea:

- `features/`: one folder per page or area, each with its own components, composables and pure
  `domain/` helpers.
- `shared/`: small helpers and UI pieces used across pages.
- `infrastructure/api/`: the generated client.

On a large screen the app lays out as a fixed 3840×2160 stage, scaled to the window, so it can be
presented. On a phone it stacks into one column.

The architecture rules behind all this are in [`docs/blueprints/`](docs/blueprints/README.md).
The decisions that shaped the app are in [`docs/adr/`](docs/adr/).

## Making a change

**A new or changed endpoint:**

1. Describe it in `openapi/v1/openapi.yaml`.
2. Run `task gen`. The build now fails until the generated handler interface is implemented.
3. Add the rule to the context's `domain/`, the use case to `app/`, and the handler under
   `adapters/web/`.
4. Call it from the web app through the generated client.

**A database change:** add the next numbered file to `db/migrations/` (goose, applied at start-up).
Put new queries in `db/queries/`, then run `task gen`. squawk lints every migration for locking
hazards. To add a check constraint to a filled table, add it `NOT VALID` in one migration and
validate it in the next, as 00005 and 00006 do.

**The research data:** the terms and companies the app starts with are in
`internal/terminology/adapters/seed/data/` (`terms.json`, `competitors.json`). The binary refuses
to start on a value the domain does not know, so a typo shows up straight away. Terms accepted in
the app's term review live in Postgres and are added on top of the seed.

**Tests:**

- **What to test:** test behaviour through the highest seam you can: a use case with fakes for its
  ports, an adapter against a real Postgres or a local HTTP server, a page through
  `web/src/testing/mountApp`.
- **No calls to paid services:** nothing in the test suite calls a language-model provider or a
  real website.
- **New pages:** add an end-to-end spec under `web/tests/e2e/`. The spec checks the page with axe
  and at phone width.

**Commits** follow [Conventional Commits](https://www.conventionalcommits.org) (`feat:`, `fix:`,
`docs:` …). Pull requests are squash-merged, and [release-please](https://github.com/googleapis/release-please)
turns the commits into the changelog and the next version. Don't edit `CHANGELOG.md` by hand.

## Configuration

| Variable | What it is |
|---|---|
| `DATABASE_URL` | Postgres connection URL (required), or `PGHOST`, `PGPORT`, `PGDATABASE`, `PGUSER`, `PGPASSWORD`, `PGSSLMODE` |
| `ADDR` | Where to listen (default `:8080`) |
| `DEV_USER`, `DEV_ROLES` | **Local only.** Every request runs as this user with these comma-separated roles (`ROLE_ADMIN` for an administrator), with no sign-in. Never set them in a deployment |
| `OIDC_ISSUER`, `OIDC_CLIENT_ID`, `OIDC_CLIENT_SECRET`, `OIDC_REDIRECT_URL` | The OpenID Connect client the app signs people in with (required unless `DEV_USER` is set) |
| `SESSION_KEY` | 32 characters or more; seals the session cookie, refresh tokens and stored API keys (required unless `DEV_USER` is set) |
| `ANTHROPIC_API_KEY` | Optional. Shown as a read-only key in Settings. While an Anthropic key works, product matching and the reading of Prompt test answers use it with `claude-haiku-5-5` (ADR-0007) |
| `LLM_MONTHLY_TOKENS` | Product matching's token budget per calendar month, all providers together (default 10,000,000) |
| `CRAWLER_CONTACT` | The contact address in the crawler's User-Agent |

Signing in is written for [auth-api](https://github.com/JorisJonkers-dev/auth-api), the estate's
OpenID Connect server: any account may read, and `ROLE_ADMIN` may change things
([ADR-0001](docs/adr/0001-in-app-sign-in-with-auth-api.md)).

## Building and deploying

`docker build .` produces one small image (distroless, non-root) that serves the API and the web
app on port 8080. It needs a Postgres database and the configuration above.

The hosted instance at `digital-strategy.jorisjonkers.dev` is deployed by the estate's
[deploy-kit](https://github.com/JorisJonkers-dev/deploy-kit). A merged release PR builds the image,
and Flux rolls it out as a canary. The deployment's description and its configuration are in
`deploy/`.

## Licence and contributing

The source is published for viewing under the
[Joris Jonkers Proprietary Source-Available License](LICENSE). It does not grant the right to use,
copy or deploy it. Contributions are welcome by invitation; see [CONTRIBUTING.md](CONTRIBUTING.md),
and report security issues as described in [SECURITY.md](SECURITY.md).
