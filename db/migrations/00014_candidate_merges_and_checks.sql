-- +goose Up
-- A migration waits at most 5s for a lock and runs at most 60s, so it fails rather than stalls.
SET lock_timeout = '5s';
SET statement_timeout = '60s';

-- Competitor candidates that are one company (internal/prompttest, ADR-0006): a candidate merged
-- into another candidate (into_id) or into a company on the list (competitor_id), by the language
-- model or a person, or split off by a person so it is never merged again (kept_apart).
CREATE TABLE IF NOT EXISTS prompt_candidate_merges (
    candidate_id text PRIMARY KEY CHECK (candidate_id ~ '^[0-9a-f]{16}$'),
    into_id text CHECK (into_id ~ '^[0-9a-f]{16}$'),
    competitor_id text CHECK (char_length(competitor_id) BETWEEN 1 AND 80),
    kept_apart boolean NOT NULL,
    merged_by text NOT NULL CHECK (char_length(merged_by) BETWEEN 1 AND 200),
    merged_at timestamptz NOT NULL,
    CHECK (num_nonnulls(into_id, competitor_id) + kept_apart::int = 1)
);

-- The language model's check of a candidate: whether it is a competitor of Tribelt, whether it was
-- sure, and why. A suggestion only; the decision stays in prompt_candidate_decisions.
CREATE TABLE IF NOT EXISTS prompt_candidate_checks (
    candidate_id text PRIMARY KEY CHECK (candidate_id ~ '^[0-9a-f]{16}$'),
    competitor boolean NOT NULL,
    sure boolean NOT NULL,
    reason text NOT NULL CHECK (char_length(reason) <= 500),
    model text NOT NULL CHECK (char_length(model) BETWEEN 1 AND 200),
    checked_at timestamptz NOT NULL
);
