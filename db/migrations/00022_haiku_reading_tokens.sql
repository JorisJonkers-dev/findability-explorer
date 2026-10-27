-- +goose Up
-- A migration waits at most 5s for a lock and runs at most 60s, so it fails rather than stalls.
SET lock_timeout = '5s';
SET statement_timeout = '60s';

-- The tokens Claude Haiku on an Anthropic key used reading an answer (internal/prompttest,
-- ADR-0007). They count towards the month's language-model tokens, as the term review's and the
-- competitor review's do, so reading stops using Haiku once the month's budget is spent. The
-- extractions stored so far count none. 00023 validates the check.
ALTER TABLE prompt_extractions ADD COLUMN IF NOT EXISTS haiku_tokens bigint NOT NULL DEFAULT 0;
ALTER TABLE prompt_extractions DROP CONSTRAINT IF EXISTS prompt_extractions_haiku_tokens_check;
ALTER TABLE prompt_extractions ADD CONSTRAINT prompt_extractions_haiku_tokens_check
CHECK (haiku_tokens >= 0) NOT VALID;
