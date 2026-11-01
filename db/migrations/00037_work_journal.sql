-- +goose Up
-- A migration waits at most 5s for a lock and runs at most 60s, so it fails rather than stalls.
SET lock_timeout = '5s';
SET statement_timeout = '60s';

-- The work journal (internal/shared/work, internal/activity): the pieces of background work no
-- table of their own records, so Activity lists them with the Prompt runs and scans (the spec's
-- section 6, Activity, and 11.1): the AI readiness check, the term review, the review of the
-- competitor candidates, and the check of the competitor list; Recommendation runs later. A row is
-- written when the work begins and finished when it ends: how (stopped by hand among them), what
-- it did (done of total, failed), and the tokens it used on the Anthropic key.
CREATE TABLE IF NOT EXISTS work_runs (
    id uuid PRIMARY KEY,
    kind text NOT NULL CHECK (kind IN (
        'promptRun', 'scan', 'rescan', 'readinessCheck', 'termReview', 'candidateReview', 'competitorCheck',
        'recommendationRun'
    )),
    started_by text NOT NULL CHECK (char_length(started_by) BETWEEN 1 AND 200),
    started_at timestamptz NOT NULL,
    finished_at timestamptz,
    ending text CHECK (ending IN ('done', 'doneWithFailures', 'stoppedByHand', 'stoppedByBudget', 'failed')),
    done bigint NOT NULL DEFAULT 0 CHECK (done >= 0),
    total bigint NOT NULL DEFAULT 0 CHECK (total >= 0),
    failed bigint NOT NULL DEFAULT 0 CHECK (failed >= 0),
    input_tokens bigint NOT NULL DEFAULT 0 CHECK (input_tokens >= 0),
    output_tokens bigint NOT NULL DEFAULT 0 CHECK (output_tokens >= 0),
    CHECK ((finished_at IS NULL) = (ending IS NULL))
);
-- The newest first, and the running ones. A plain CREATE INDEX on a table made in the same
-- transaction blocks nobody.
CREATE INDEX IF NOT EXISTS work_runs_started_at_idx ON work_runs (started_at DESC);
CREATE INDEX IF NOT EXISTS work_runs_running_idx ON work_runs (started_at) WHERE finished_at IS NULL;
