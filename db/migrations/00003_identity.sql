-- +goose Up
-- A migration waits at most 5s for a lock and runs at most 60s, so it fails rather than stalls.
SET lock_timeout = '5s';
SET statement_timeout = '60s';

-- Local accounts and server-side sessions (internal/identity, docs/adr/0001). An account is made
-- on its first auth-api sign-in and keyed by issuer and subject; auth-api stays the only way to
-- sign in and the source of the roles. A session holds the sealed refresh token the service trades
-- for the account's current roles, so a role granted or withdrawn in auth-api applies within
-- seconds. The browser only holds the sealed session id.
CREATE TABLE IF NOT EXISTS accounts (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    issuer text NOT NULL CHECK (char_length(issuer) BETWEEN 1 AND 500),
    subject text NOT NULL CHECK (char_length(subject) BETWEEN 1 AND 200),
    name text NOT NULL DEFAULT '' CHECK (char_length(name) <= 200),
    email text NOT NULL DEFAULT '' CHECK (char_length(email) <= 320),
    roles text[] NOT NULL DEFAULT '{}',
    created_at timestamptz NOT NULL DEFAULT now(),
    last_login_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (issuer, subject)
);

CREATE TABLE IF NOT EXISTS sessions (
    id text PRIMARY KEY CHECK (char_length(id) BETWEEN 32 AND 64),
    account_id bigint NOT NULL REFERENCES accounts (id) ON DELETE CASCADE,
    refresh_token bytea NOT NULL,
    key_id text NOT NULL CHECK (char_length(key_id) <= 32),
    user_agent text NOT NULL DEFAULT '' CHECK (char_length(user_agent) <= 300),
    created_at timestamptz NOT NULL DEFAULT now(),
    last_seen_at timestamptz NOT NULL DEFAULT now(),
    roles_checked_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL
);

-- A plain CREATE INDEX on a table made in the same transaction blocks nobody.
-- squawk-ignore require-concurrent-index-creation
CREATE INDEX IF NOT EXISTS sessions_account_idx ON sessions (account_id);
-- squawk-ignore require-concurrent-index-creation
CREATE INDEX IF NOT EXISTS sessions_expires_at_idx ON sessions (expires_at);
