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
  from the map it uses, and, when a language model is active, its products matched to Tribelt's
  families. This ports and extends
  [RNgo-Sch/digital-strategy-webscraper](https://github.com/RNgo-Sch/digital-strategy-webscraper).

The whole app is private. Everyone signs in with their auth-api account
(`auth.jorisjonkers.dev`); any account may view and scan, and only auth-api's `ROLE_ADMIN` may open
the admin settings ([ADR-0001](docs/adr/0001-in-app-sign-in-with-auth-api.md)).

- **Admin settings** (`/api/v1/admin/llm`): API keys for Anthropic, OpenRouter and Mistral, more
  than one each, sealed in Postgres and never shown again; which key and model product matching
  uses; the monthly token budget; and a log of every change and test
  ([ADR-0002](docs/adr/0002-sealed-llm-credentials-in-postgres.md)).

It is built from the estate's [template-go-vue](https://github.com/JorisJonkers-dev/template-go-vue):
a contract-first Go API (`openapi/v1/openapi.yaml`, ogen, sqlc, goose) and a Vue SPA embedded in
one binary. The architecture rules are in [`docs/blueprints/`](docs/blueprints/README.md).

| Path | What it is |
|------|------------|
| `internal/terminology/` | The terminology map: search with folding of case, accents and ß; the embedded seed data |
| `internal/analysis/` | Website analyses: target and robots rules, keyword and known-term extraction, the crawler, the Claude matcher, Postgres persistence |
| `internal/identity/` | Signing in with auth-api: local accounts, server-side sessions, roles re-read from auth-api, the gate in front of the web app |
| `internal/llm/` | The language-model settings: sealed API keys, the active credential and model, the event log, and the Anthropic, OpenRouter and Mistral clients |
| `internal/shared/fold/` | The one text normalisation both contexts and the web client use |
| `internal/shared/seal/` | AES-256-GCM sealing under keys derived from `SESSION_KEY`, for stored secrets and cookies |
| `web/src/features/` | `terminology` and `analyses` pages |

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
| `ANALYSES_PER_HOUR` | Analyses one account may start per hour (default 10) |
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
