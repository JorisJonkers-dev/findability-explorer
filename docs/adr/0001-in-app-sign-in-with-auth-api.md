# The app signs people in with auth-api itself, instead of the edge's forward-auth

The Findability Explorer is private: every page and every API call needs an auth-api account, and
the admin settings (the language-model keys, ADR-0002) need auth-api's `ROLE_ADMIN`. It was built
to sit behind the platform's forward-auth, which put the caller's subject in `X-User-Id` and nothing
else. That gave the app no roles to decide who is an admin, no name to show, and no sign-out of its
own; and deploy-kit's sign-in chapter puts a forward-auth check in front of a host only for a
`gated` Client, while an app that speaks OIDC is a `native` one.

So the app runs the OpenID Connect authorization code flow against auth-api itself, the way the
tribelt mirror's `/stats` portal does (tribelt ADR-0006), and keeps local accounts and server-side
sessions in its own Postgres:

- `GET /auth/login?return=<path>` starts the flow with PKCE, state and nonce, sealed in a
  short-lived `fe_oidc` cookie; only a path on this site is accepted as the return.
- `GET /auth/callback` verifies the ID token, upserts the Account (issuer + subject), opens a
  Session that holds the sealed refresh token, and sets `fe_session` (HttpOnly, SameSite=Lax,
  Secure unless the redirect URL is plain http) holding only the sealed session id.
- `POST /auth/logout` deletes the session and revokes its refresh token. It does not end the
  auth-api session, so the person stays signed in to every other app; it lands on
  `/auth/signed-out`, not the app, which would sign them straight back in.

auth-api stays the source of the roles. A request re-reads them by trading the stored refresh
token when they are older than 15 seconds, and always right before an admin change, under a row
lock so concurrent requests never spend one rotated token twice. While auth-api cannot be
reached, reading goes on with the last known roles for up to five minutes; admin changes are
refused (503).

Any auth-api account may view the app and start scans; only `ROLE_ADMIN` may use
`/api/v1/admin/*` (403 otherwise). The web app's pages redirect to sign in; every other request
without a session (assets, the API) gets 401. Unsafe requests to `/api/` and `/auth/` must come
from this origin (`Sec-Fetch-Site`, else `Origin` against `Host`), the defence against
cross-site request forgery for a cookie session.

## Consequences

- The edge adds no forward-auth: the exposure is `audience: anonymous` with deploy-kit 0.30.0,
  which does not implement the `client` block yet. When it does, it becomes `authenticated` with a
  `native` Client (redirect `/auth/callback`, protected path `/api/v1/me`).
- auth-api needs a client registration for `findability-explorer`, and Vault two secrets:
  `OIDC_CLIENT_SECRET` and `SESSION_KEY`. Without them the binary refuses to start, unless
  `DEV_USER` (and `DEV_ROLES`) stand in for a signed-in account on a local run.
- Rotating `SESSION_KEY` signs everyone out (their refresh tokens no longer open) and makes the
  stored API keys unreadable (ADR-0002).
- The per-user hourly limit on analyses counts the auth-api subject, as it did the forward-auth
  header.
