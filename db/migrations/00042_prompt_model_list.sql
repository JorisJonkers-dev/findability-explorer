-- +goose Up
-- A migration waits at most 5s for a lock and runs at most 60s, so it fails rather than stalls.
SET lock_timeout = '5s';
SET statement_timeout = '60s';

-- The Model list (the spec's decision 10): the OpenRouter models every Prompt run asks, each its
-- own column. Empty until an administrator saves one; until then runs ask the cheapest model of
-- each provider in providers, as before.
ALTER TABLE prompt_settings ADD COLUMN IF NOT EXISTS models text [] NOT NULL DEFAULT '{}';

ALTER TABLE prompt_settings DROP CONSTRAINT IF EXISTS prompt_settings_models_count;
ALTER TABLE prompt_settings ADD CONSTRAINT prompt_settings_models_count CHECK (cardinality(models) <= 20) NOT VALID;
ALTER TABLE prompt_settings VALIDATE CONSTRAINT prompt_settings_models_count;
