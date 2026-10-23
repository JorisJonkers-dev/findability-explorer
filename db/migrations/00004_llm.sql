-- +goose Up
-- A migration waits at most 5s for a lock and runs at most 60s, so it fails rather than stalls.
SET lock_timeout = '5s';
SET statement_timeout = '60s';

-- Language-model API keys an admin added (internal/llm, docs/adr/0002). The key is sealed with
-- AES-256-GCM under a key derived from SESSION_KEY; key_id names that key, and the row's id and
-- provider are bound in as associated data, so a sealed key cannot move to another row. The
-- fingerprint is the first 12 hex characters of the key's SHA-256. The key Vault delivers is not
-- stored; it is the credential `vault-anthropic`.
CREATE TABLE IF NOT EXISTS llm_credentials (
    id uuid PRIMARY KEY,
    provider text NOT NULL CHECK (provider IN ('anthropic', 'openrouter', 'mistral')),
    label text NOT NULL CHECK (char_length(label) BETWEEN 1 AND 100),
    secret bytea NOT NULL,
    key_id text NOT NULL CHECK (char_length(key_id) <= 32),
    fingerprint text NOT NULL CHECK (fingerprint ~ '^[0-9a-f]{12}$'),
    created_by text NOT NULL CHECK (char_length(created_by) BETWEEN 1 AND 200),
    created_at timestamptz NOT NULL
);

-- The credential and model product matching uses: one row at most. No row means nobody chose
-- (the Vault key with the default model, when there is one); mode off means an admin turned
-- matching off.
CREATE TABLE IF NOT EXISTS llm_active (
    singleton boolean PRIMARY KEY DEFAULT true CHECK (singleton),
    mode text NOT NULL CHECK (mode IN ('chosen', 'off')),
    credential_id text CHECK (char_length(credential_id) <= 36),
    model text CHECK (char_length(model) BETWEEN 1 AND 200),
    updated_by text NOT NULL CHECK (char_length(updated_by) BETWEEN 1 AND 200),
    updated_at timestamptz NOT NULL,
    CHECK ((mode = 'chosen') = (credential_id IS NOT NULL AND model IS NOT NULL))
);

-- Every change to and test of the settings, appended and never updated.
CREATE TABLE IF NOT EXISTS llm_events (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    at timestamptz NOT NULL,
    actor text NOT NULL CHECK (char_length(actor) BETWEEN 1 AND 200),
    action text NOT NULL CHECK (action IN (
        'credential_added', 'credential_deleted', 'credential_tested', 'active_set', 'active_cleared'
    )),
    credential_id text CHECK (char_length(credential_id) <= 36),
    provider text CHECK (provider IN ('anthropic', 'openrouter', 'mistral')),
    model text CHECK (char_length(model) <= 200),
    ok boolean,
    message text NOT NULL CHECK (char_length(message) <= 500)
);

-- squawk-ignore require-concurrent-index-creation
CREATE INDEX IF NOT EXISTS llm_events_credential_idx ON llm_events (credential_id, id DESC)
    WHERE action IN ('credential_added', 'credential_tested');

-- The provider and model that matched an analysis's products, when one was asked.
ALTER TABLE analyses ADD COLUMN IF NOT EXISTS matching_model text CHECK (char_length(matching_model) <= 300);
