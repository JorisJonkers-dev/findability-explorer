-- +goose Up
-- A migration waits at most 5s for a lock and runs at most 60s, so it fails rather than stalls.
SET lock_timeout = '5s';
SET statement_timeout = '60s';

-- The language model's verdicts on the review queue's candidates (internal/terminology, ADR-0005):
-- accept or reject, whether it was sure, why, and the term an accepted one becomes. A sure verdict
-- is also a decision in term_decisions, by the language model; an unsure one leaves the candidate
-- waiting for a person. The model is asked about a candidate once, so a verdict is never replaced.
CREATE TABLE IF NOT EXISTS term_verdicts (
    candidate_id text PRIMARY KEY CHECK (candidate_id ~ '^[0-9a-f]{16}$'),
    phrase text NOT NULL CHECK (char_length(phrase) BETWEEN 1 AND 200),
    accept boolean NOT NULL,
    sure boolean NOT NULL,
    reason text NOT NULL CHECK (char_length(reason) <= 500),
    term text NOT NULL CHECK (char_length(term) <= 200),
    language text NOT NULL CHECK (char_length(language) <= 16),
    family text NOT NULL CHECK (char_length(family) <= 30),
    tier text NOT NULL CHECK (char_length(tier) <= 30),
    model text NOT NULL CHECK (char_length(model) BETWEEN 1 AND 200),
    reviewed_at timestamptz NOT NULL
);

-- What each request to the language model for verdicts consumed, a failed one included.
CREATE TABLE IF NOT EXISTS term_review_usage (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    input_tokens bigint NOT NULL CHECK (input_tokens >= 0),
    output_tokens bigint NOT NULL CHECK (output_tokens >= 0),
    used_at timestamptz NOT NULL
);

CREATE INDEX IF NOT EXISTS term_review_usage_used_at ON term_review_usage (used_at);
