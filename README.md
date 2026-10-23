# Findability Explorer

The course-material repository links this repository as a submodule at `findability-explorer/`.

The interactive companion to the Digital Strategy group report on Tribelt B.V. (University of
Twente, 2026-1A), served privately at `digital-strategy.jorisjonkers.dev`. It shows how metal
conveyor belts are named across companies and languages, and analyses any website's keywords.

- **Search terms** (`/terms`): search the terminology map (146 terms, 10 languages) from the
  group's desk research of 22 October 2026, plus every term an administrator accepted from the
  review queue, with filters by language, intent tier, belt family and company. Every filter lives
  in the URL, so a view can be shared.
- **Competitors** (`/competitors`): the 25 competitors and 3 group companies, with their own name
  for each Tribelt belt family, and every company an administrator added.
- **Scans** (`/scan/:id`): crawl up to 100 pages of a company's website (default 60) and report its
  most prominent search terms, the terms from the map it uses, and, when a language model is
  active, its products matched to Tribelt's families. The crawl starts from the home page, reads
  the site's sitemaps (the ones `robots.txt` lists, or `/sitemap.xml`, sitemap indexes and gzip
  included) and the language versions each page declares (`hreflang`), and fetches the pages whose
  address and link text use the belt vocabulary first: product, specification and application
  pages before news, contact, legal and careers. It stays polite (robots.txt with RFC 9309
  wildcards, 500 ms between requests, public addresses only) and records which belt family each
  page is about when that is clear. This ports and extends
  [RNgo-Sch/digital-strategy-webscraper](https://github.com/RNgo-Sch/digital-strategy-webscraper).
- **Search terms and history** (`/competitors/:id/history`): a company's search terms from its
  latest finished scan, by Tribelt belt family (eyelink, open spiral, closed spiral, Tri-Flex,
  other) with broad and specification terms apart, per language, with the scan each term first
  appeared in; the numbered scan history; and a comparison of any two finished scans (terms added,
  removed, and moved by at least a quarter of their score and 5 points, or 2 pages).
- **Term review** (`/settings/terms`, administrators): every phrase a finished scan found outside
  the map, across companies, with its evidence and the classifier's suggested family and tier. An
  administrator accepts one (with the text, language, family, tier and companies the map should
  show), rejects it, or undoes either ([ADR-0003](docs/adr/0003-scans-are-versions-and-the-map-grows-through-review.md)).

The whole app is private. Everyone signs in with their auth-api account
(`auth.jorisjonkers.dev`); any account may read the map, the companies, every scan and its history.
Only auth-api's `ROLE_ADMIN` may start a scan or a re-scan, add, edit, archive or remove a company,
review terms, and open the admin settings
([ADR-0001](docs/adr/0001-in-app-sign-in-with-auth-api.md)).

- **Companies** (administrators): an added company can be renamed, moved to another website (a host
  another company has is refused) or deleted with every scan of it. A researched company is the
  desk research's record: an administrator's change to it is stored beside it, and removing it
  archives it (it leaves the list and the re-scans; its scans stay; restoring it brings it back).
  Nothing is removed while a scan of it is queued or running.
- **Re-scans** (`POST /api/v1/admin/rescans`, administrators, on demand only): one scan per named
  company, or of every company that is not archived, has a website and was scanned before. A
  company with a scan still queued or running is skipped. Three scans run at a time, each on
  another company's site; re-scans do not count against the hourly limit on single scans.

- **Admin settings** (`/api/v1/admin/llm`): API keys for Anthropic, OpenRouter and Mistral, more
  than one each, sealed in Postgres and never shown again; which key and model product matching
  uses; the monthly token budget; and a log of every change and test
  ([ADR-0002](docs/adr/0002-sealed-llm-credentials-in-postgres.md)).

It is built from the estate's [template-go-vue](https://github.com/JorisJonkers-dev/template-go-vue):
a contract-first Go API (`openapi/v1/openapi.yaml`, ogen, sqlc, goose) and a Vue SPA embedded in
one binary. The architecture rules are in [`docs/blueprints/`](docs/blueprints/README.md).

| Path | What it is |
|------|------------|
| `internal/terminology/` | The terminology map: search with folding of case, accents and ß; the embedded seed data; the review queue and the terms accepted from it |
| `internal/analysis/` | Website analyses: target, robots and sitemap rules, the link-scoring frontier, keyword and known-term extraction, the crawler, the matcher, company profiles and scan comparisons, re-scans, Postgres persistence |
| `internal/companies/` | The researched companies, the ones administrators added, and administrators' changes to both |
| `internal/identity/` | Signing in with auth-api: local accounts, server-side sessions, roles re-read from auth-api, the gate in front of the web app |
| `internal/llm/` | The language-model settings: sealed API keys, the active credential and model, the event log, and the Anthropic, OpenRouter and Mistral clients |
| `internal/shared/fold/` | The one text normalisation both contexts and the web client use |
| `internal/shared/seal/` | AES-256-GCM sealing under keys derived from `SESSION_KEY`, for stored secrets and cookies |
| `web/src/features/` | The poster and its section views, the company history, the scan panel, the term review and the settings |

## Run it

```bash
mise install    # the pinned toolchain
task check      # lint, generated-code drift, Go and web tests with coverage, build, secret scan
task e2e        # Playwright with axe against the built binary and the compose Postgres
task dev        # Postgres, the API on :8080, Vite on :5173
```

Environment:

| Variable | What it is |
|---|---|
| `DATABASE_URL` | Postgres (required); or `PGHOST` with the other libpq variables |
| `ADDR` | Where to listen (default `:8080`) |
| `OIDC_ISSUER`, `OIDC_CLIENT_ID`, `OIDC_CLIENT_SECRET`, `OIDC_REDIRECT_URL` | The client registration at auth-api (required unless `DEV_USER` is set). Cookies are `Secure` unless the redirect URL is plain http |
| `SESSION_KEY` | At least 32 characters; seals the session cookie, the refresh tokens and the API keys admins add (required unless `DEV_USER` is set) |
| `ANTHROPIC_API_KEY` | The Vault key, shown as the read-only credential `vault-anthropic`; until an admin chooses, matching uses it with `claude-haiku-5-5` |
| `LLM_MONTHLY_TOKENS` | The language-model budget per calendar month, every provider together (default 2,000,000) |
| `ANALYSES_PER_HOUR` | Single scans one account may start per hour (default 10); re-scans do not count |
| `CRAWLER_CONTACT` | Put in the crawler's User-Agent |
| `DEV_USER`, `DEV_ROLES` | Local runs only: every request runs as `DEV_USER` with the comma-separated `DEV_ROLES` (`ROLE_ADMIN` opens the admin settings), without auth-api. Never set them in a deployment |

### Signing in

`GET /auth/login?return=/path` sends the browser to auth-api and back to `/path`;
`GET /auth/callback` finishes the sign-in and sets the `fe_session` cookie (HttpOnly,
SameSite=Lax). `POST /auth/logout` ends the session here, revokes its refresh token, and lands on
`/auth/signed-out`; the auth-api session stays. Without a session a page redirects to sign in and
everything else, the API included, answers 401; an admin operation answers 403 to an account
without `ROLE_ADMIN`. Roles are re-read from auth-api when they are older than 15 seconds and right
before every admin change. Unsafe requests to `/api/` and `/auth/` must come from the app's own
origin.

### Product matching

The admin settings list the Vault key and every key an admin added, each with its fingerprint and
latest test. Adding a key checks it with the provider first. The admin picks a credential and one of
its models (`GET …/credentials/{id}/models`), or turns matching off; each analysis uses the choice
made when it runs and records it as `matchingModel`, such as `openrouter: anthropic/claude-haiku-5-5`.

To update the research part of the terminology map, replace both files in
`internal/terminology/adapters/seed/data/` with the research JSON from the course-material
repository; the binary refuses to start on a value the domain does not know. Terms accepted in the
term review live in Postgres (`accepted_terms`) and join the seed's at startup and after every
decision; a term's `origin` says which it is.

### Migrations 00005 and 00006

00005 widens the page limits to 100 (the checks are swapped `NOT VALID`; 00006 validates them in a
transaction of its own), adds a page's `family`, numbers every scan per company (`scan_number`,
backfilled in the order the scans were started, unique per company), marks re-scans (`rescan`),
and adds `company_overrides`, `term_decisions` and `accepted_terms`. Nothing is dropped; existing
scans keep their results, read as scan 1, 2, … of their company, and their phrases outside the map
fill the review queue at once.

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

**State: live.** Merging a release PR builds the multi-arch image to
`ghcr.io/jorisjonkers-dev/findability-explorer`, publishes the signed Intent Fragment and starts
composition (`compose: true`); Flux applies it at `digital-strategy.jorisjonkers.dev`.

- **Sign-in:** auth-api registers the client `findability-explorer` (auth-api#111). People need the
  `FINDABILITY_EXPLORER` service permission, granted in home-portal's admin page; admins always pass.
- **Secrets:** `estate/bootstrap/secrets.sh --only findability-explorer` writes the client secret to
  `secret/auth-api` and `secret/findability-explorer-system/oidc`, and optionally the Anthropic key
  (without it, matching is off until an admin adds a key in the app). `SESSION_KEY` is a deploy-kit
  generated key. The client secret becomes derived once deploy-kit implements the `client` block
  (spec/v1, chapter 17); then the exposure becomes `authenticated` + `signIn: native` (ADR-0001).
- **Estate:** listed in `participants.yml` and `platform.intent.yml`'s `handover` (estate#197).

**Actions minutes are scarce in this private repo.** CI runs on pull requests that touch code and
by hand, not on pushes to main; run `task check` and `task e2e` locally before you push.
