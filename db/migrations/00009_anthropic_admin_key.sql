-- +goose Up
-- A migration waits at most 5s for a lock and runs at most 60s, so it fails rather than stalls.
SET lock_timeout = '5s';
SET statement_timeout = '60s';

-- An Anthropic Admin key (anthropic_admin) is stored sealed like the other keys; it only reads the
-- organisation's usage and cost reports (internal/llm). Each check is swapped in place; 00010
-- validates them in a transaction of its own.
ALTER TABLE llm_credentials DROP CONSTRAINT IF EXISTS llm_credentials_provider_check;
ALTER TABLE llm_credentials ADD CONSTRAINT llm_credentials_provider_check
CHECK (provider IN ('anthropic', 'openrouter', 'mistral', 'anthropic_admin')) NOT VALID;
ALTER TABLE llm_events DROP CONSTRAINT IF EXISTS llm_events_provider_check;
ALTER TABLE llm_events ADD CONSTRAINT llm_events_provider_check
CHECK (provider IN ('anthropic', 'openrouter', 'mistral', 'anthropic_admin')) NOT VALID;
