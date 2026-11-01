-- +goose Up
-- A migration waits at most 5s for a lock and runs at most 60s, so it fails rather than stalls.
SET lock_timeout = '5s';
SET statement_timeout = '60s';

-- The Recommendation run (the spec's section 10, 11.9): each run with how it ended, the model, what
-- it added, replaced and kept, and the tokens it used on the Anthropic key; and the recommendations
-- the runs wrote. A run that ends done replaces the open ones (neither done nor edited) with its
-- own, in one transaction; one stopped, cut short or failed changes none. New tables only, so the
-- old pod serves on.
CREATE TABLE IF NOT EXISTS recommendation_runs (
    id uuid PRIMARY KEY,
    started_at timestamptz NOT NULL,
    finished_at timestamptz,
    ending text CHECK (ending IN ('done', 'doneWithFailures', 'stoppedByHand', 'stoppedByBudget', 'failed')),
    started_by text NOT NULL CHECK (char_length(started_by) BETWEEN 1 AND 200),
    model text NOT NULL CHECK (char_length(model) <= 200),
    gaps bigint NOT NULL DEFAULT 0 CHECK (gaps >= 0),
    added bigint NOT NULL DEFAULT 0 CHECK (added >= 0),
    replaced bigint NOT NULL DEFAULT 0 CHECK (replaced >= 0),
    kept bigint NOT NULL DEFAULT 0 CHECK (kept >= 0),
    input_tokens bigint NOT NULL DEFAULT 0 CHECK (input_tokens >= 0),
    output_tokens bigint NOT NULL DEFAULT 0 CHECK (output_tokens >= 0),
    CHECK ((finished_at IS NULL) = (ending IS NULL))
);
CREATE INDEX IF NOT EXISTS recommendation_runs_started_at_idx ON recommendation_runs (started_at DESC);

CREATE TABLE IF NOT EXISTS recommendations (
    id text PRIMARY KEY CHECK (char_length(id) BETWEEN 1 AND 60 AND id ~ '^[a-z0-9]+(-[a-z0-9]+)*$'),
    run_id uuid NOT NULL REFERENCES recommendation_runs (id),
    gap_key text NOT NULL CHECK (char_length(gap_key) BETWEEN 1 AND 600),
    kind text NOT NULL CHECK (kind IN ('term', 'page', 'listing', 'fact', 'readiness')),
    language text NOT NULL DEFAULT '' CHECK (char_length(language) <= 10),
    family text NOT NULL DEFAULT '' CHECK (char_length(family) <= 40),
    specifics text [] NOT NULL CHECK (cardinality(specifics) <= 10),
    rule_benefit smallint NOT NULL CHECK (rule_benefit BETWEEN 1 AND 5),
    rule_effort smallint NOT NULL CHECK (rule_effort BETWEEN 1 AND 5),
    title text NOT NULL CHECK (char_length(title) BETWEEN 1 AND 200),
    where_url text NOT NULL DEFAULT '' CHECK (char_length(where_url) <= 500),
    where_note text NOT NULL DEFAULT '' CHECK (char_length(where_note) <= 200),
    why text NOT NULL CHECK (char_length(why) <= 500),
    steps text [] NOT NULL CHECK (cardinality(steps) <= 6),
    done_when text NOT NULL CHECK (char_length(done_when) <= 300),
    benefit smallint NOT NULL CHECK (benefit BETWEEN 1 AND 5),
    effort smallint NOT NULL CHECK (effort BETWEEN 1 AND 5),
    score_reason text NOT NULL DEFAULT '' CHECK (char_length(score_reason) <= 300),
    created_at timestamptz NOT NULL,
    done_by text CHECK (char_length(done_by) BETWEEN 1 AND 200),
    done_at timestamptz,
    edited_at timestamptz,
    CHECK ((done_by IS NULL) = (done_at IS NULL))
);
CREATE INDEX IF NOT EXISTS recommendations_run_idx ON recommendations (run_id);
