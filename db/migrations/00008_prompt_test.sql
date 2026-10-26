-- +goose Up
-- A migration waits at most 5s for a lock and runs at most 60s, so it fails rather than stalls.
SET lock_timeout = '5s';
SET statement_timeout = '60s';

-- The Prompt test (internal/prompttest, docs/adr/0004). Its settings: one row at most; no row
-- means the defaults. Money is kept in millionths of a dollar, so sums are exact.
CREATE TABLE IF NOT EXISTS prompt_settings (
    singleton boolean PRIMARY KEY DEFAULT true CHECK (singleton),
    credential_id text CHECK (char_length(credential_id) <= 36),
    providers text [] NOT NULL CHECK (cardinality(providers) BETWEEN 1 AND 20),
    monthly_budget_micros bigint NOT NULL CHECK (monthly_budget_micros BETWEEN 1 AND 1000000000),
    cadence text NOT NULL CHECK (cadence IN ('monthly', 'weekly')),
    next_due timestamptz NOT NULL,
    fact_sheet text NOT NULL CHECK (char_length(fact_sheet) BETWEEN 1 AND 4000),
    fact_sheet_version bigint NOT NULL CHECK (fact_sheet_version >= 1),
    fact_sheet_updated_by text CHECK (char_length(fact_sheet_updated_by) <= 200),
    fact_sheet_updated_at timestamptz,
    updated_by text NOT NULL CHECK (char_length(updated_by) BETWEEN 1 AND 200),
    updated_at timestamptz NOT NULL
);

-- One Prompt run: the prompt-set and fact-sheet versions it used, and the instance that holds it
-- (owner) with when it was last heard of, so another instance takes over a run whose holder went
-- quiet. The partial unique index keeps one run running at a time, across instances.
CREATE TABLE IF NOT EXISTS prompt_runs (
    id uuid PRIMARY KEY,
    prompt_set_version bigint NOT NULL CHECK (prompt_set_version >= 1),
    fact_sheet_version bigint NOT NULL CHECK (fact_sheet_version >= 1),
    fact_sheet text NOT NULL CHECK (char_length(fact_sheet) <= 4000),
    trigger text NOT NULL CHECK (trigger IN ('manual', 'schedule')),
    started_by text NOT NULL CHECK (char_length(started_by) BETWEEN 1 AND 200),
    started_at timestamptz NOT NULL,
    finished_at timestamptz,
    status text NOT NULL CHECK (status IN ('running', 'done', 'stopped', 'failed')),
    failure text CHECK (char_length(failure) <= 300),
    tasks bigint NOT NULL CHECK (tasks >= 0),
    owner text NOT NULL CHECK (char_length(owner) BETWEEN 1 AND 64),
    heartbeat_at timestamptz NOT NULL,
    CHECK ((status = 'running') = (finished_at IS NULL))
);
CREATE UNIQUE INDEX IF NOT EXISTS prompt_runs_one_running_idx ON prompt_runs (status) WHERE status = 'running';
CREATE INDEX IF NOT EXISTS prompt_runs_started_at_idx ON prompt_runs (started_at DESC);

-- The model a run asks for each provider, and the paid fallback of a free one.
CREATE TABLE IF NOT EXISTS prompt_run_models (
    run_id uuid NOT NULL REFERENCES prompt_runs (id) ON DELETE CASCADE,
    position bigint NOT NULL CHECK (position BETWEEN 0 AND 19),
    provider text NOT NULL CHECK (char_length(provider) BETWEEN 1 AND 40),
    model text NOT NULL CHECK (char_length(model) BETWEEN 1 AND 200),
    free boolean NOT NULL,
    fallback text NOT NULL CHECK (char_length(fallback) <= 200),
    PRIMARY KEY (run_id, position)
);

-- One answer: one prompt of the run, one provider's model, one mode; failure when the model gave
-- none. The prompt's own text is kept, so an answer reads the same whatever later versions ask.
CREATE TABLE IF NOT EXISTS prompt_answers (
    id uuid PRIMARY KEY,
    run_id uuid NOT NULL REFERENCES prompt_runs (id) ON DELETE CASCADE,
    prompt_id text NOT NULL CHECK (char_length(prompt_id) BETWEEN 1 AND 80),
    family text NOT NULL CHECK (family IN ('eyelink', 'open_spiral', 'closed_spiral', 'tri_flex_turn_curve', 'other')),
    kind text NOT NULL CHECK (kind IN ('supplier', 'application', 'problem')),
    scope text NOT NULL CHECK (scope IN ('twente', 'netherlands', 'germany', 'dutch', 'german', 'english')),
    language text NOT NULL CHECK (language IN ('nl', 'de', 'en')),
    prompt_text text NOT NULL CHECK (char_length(prompt_text) <= 1000),
    prompt_set_version bigint NOT NULL CHECK (prompt_set_version >= 1),
    provider text NOT NULL CHECK (char_length(provider) BETWEEN 1 AND 40),
    model text NOT NULL CHECK (char_length(model) BETWEEN 1 AND 200),
    mode text NOT NULL CHECK (mode IN ('knowledge', 'web')),
    answer text NOT NULL CHECK (char_length(answer) <= 100000),
    citations text [] NOT NULL CHECK (cardinality(citations) <= 100),
    input_tokens bigint NOT NULL CHECK (input_tokens >= 0),
    output_tokens bigint NOT NULL CHECK (output_tokens >= 0),
    cost_micros bigint NOT NULL CHECK (cost_micros >= 0),
    failure text CHECK (char_length(failure) <= 300),
    answered_at timestamptz NOT NULL,
    UNIQUE (run_id, prompt_id, provider, mode)
);
CREATE INDEX IF NOT EXISTS prompt_answers_answered_at_idx ON prompt_answers (answered_at);

-- How the fixed extractor read an answer; failed when it could not, after a retry.
CREATE TABLE IF NOT EXISTS prompt_extractions (
    answer_id uuid PRIMARY KEY REFERENCES prompt_answers (id) ON DELETE CASCADE,
    failed boolean NOT NULL,
    addresses text [] NOT NULL CHECK (cardinality(addresses) <= 200),
    names_tribelt boolean NOT NULL,
    cites_tribelt boolean NOT NULL,
    correctness text NOT NULL CHECK (correctness IN ('', 'correct', 'partly', 'wrong')),
    wrong_claims text [] NOT NULL CHECK (cardinality(wrong_claims) <= 10),
    input_tokens bigint NOT NULL CHECK (input_tokens >= 0),
    output_tokens bigint NOT NULL CHECK (output_tokens >= 0),
    cost_micros bigint NOT NULL CHECK (cost_micros >= 0),
    extracted_at timestamptz NOT NULL
);
CREATE INDEX IF NOT EXISTS prompt_extractions_extracted_at_idx ON prompt_extractions (extracted_at);

-- The companies an answer names, in order, matched to the company list when they are on it.
CREATE TABLE IF NOT EXISTS prompt_answer_companies (
    answer_id uuid NOT NULL REFERENCES prompt_extractions (answer_id) ON DELETE CASCADE,
    position bigint NOT NULL CHECK (position BETWEEN 1 AND 30),
    name text NOT NULL CHECK (char_length(name) BETWEEN 1 AND 200),
    url text NOT NULL CHECK (char_length(url) <= 500),
    competitor_id text CHECK (char_length(competitor_id) <= 80),
    PRIMARY KEY (answer_id, position)
);
CREATE INDEX IF NOT EXISTS prompt_answer_companies_unmatched_idx ON prompt_answer_companies (answer_id) WHERE competitor_id IS NULL;

-- An administrator's decision on a competitor candidate, keyed by the candidate's id (made from its
-- name), so a dismissed name stays dismissed in later runs. Undoing a decision deletes its row.
CREATE TABLE IF NOT EXISTS prompt_candidate_decisions (
    id text PRIMARY KEY CHECK (id ~ '^[0-9a-f]{16}$'),
    name text NOT NULL CHECK (char_length(name) BETWEEN 1 AND 200),
    status text NOT NULL CHECK (status IN ('added', 'dismissed')),
    competitor_id text CHECK (char_length(competitor_id) <= 80),
    decided_by text NOT NULL CHECK (char_length(decided_by) BETWEEN 1 AND 200),
    decided_at timestamptz NOT NULL,
    CHECK ((status = 'added') = (competitor_id IS NOT NULL))
);
