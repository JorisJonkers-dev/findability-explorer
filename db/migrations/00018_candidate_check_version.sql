-- +goose Up
-- A migration waits at most 5s for a lock and runs at most 60s, so it fails rather than stalls.
SET lock_timeout = '5s';
SET statement_timeout = '60s';

-- Which version of the check made a competitor candidate's check (internal/prompttest, ADR-0009).
-- The checks stored so far were made without Tribelt's target market and without evidence beyond
-- the names, so they are version 1, and every waiting company is checked again with version 2.
-- 00019 validates the check.
ALTER TABLE prompt_candidate_checks ADD COLUMN IF NOT EXISTS check_version bigint NOT NULL DEFAULT 1;
ALTER TABLE prompt_candidate_checks DROP CONSTRAINT IF EXISTS prompt_candidate_checks_check_version_check;
ALTER TABLE prompt_candidate_checks ADD CONSTRAINT prompt_candidate_checks_check_version_check
CHECK (check_version >= 1) NOT VALID;

-- What each request to Claude Haiku for merging and checking the candidates consumed, a failed one
-- included; it counts towards the month's tokens.
CREATE TABLE IF NOT EXISTS prompt_review_usage (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    input_tokens bigint NOT NULL CHECK (input_tokens >= 0),
    output_tokens bigint NOT NULL CHECK (output_tokens >= 0),
    used_at timestamptz NOT NULL
);

CREATE INDEX IF NOT EXISTS prompt_review_usage_used_at ON prompt_review_usage (used_at);
