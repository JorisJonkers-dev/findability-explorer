-- +goose Up
-- A migration waits at most 5s for a lock and runs at most 60s, so it fails rather than stalls.
SET lock_timeout = '5s';
SET statement_timeout = '60s';

-- One analysis of one website (internal/analysis/domain). Its results live in child tables, one
-- row per result, ordered by position; no JSONB (docs/blueprints/go-api.md §8.1).
CREATE TABLE IF NOT EXISTS analyses (
    id uuid PRIMARY KEY,
    url text NOT NULL CHECK (char_length(url) BETWEEN 8 AND 500),
    max_pages bigint NOT NULL CHECK (max_pages BETWEEN 1 AND 30),
    status text NOT NULL CHECK (status IN ('queued', 'running', 'done', 'failed')),
    failure text CHECK (char_length(failure) <= 500),
    pages_crawled bigint NOT NULL DEFAULT 0 CHECK (pages_crawled BETWEEN 0 AND 30),
    matching text NOT NULL DEFAULT 'pending'
        CHECK (matching IN ('pending', 'disabled', 'skipped', 'done', 'failed')),
    llm_input_tokens bigint NOT NULL DEFAULT 0 CHECK (llm_input_tokens >= 0),
    llm_output_tokens bigint NOT NULL DEFAULT 0 CHECK (llm_output_tokens >= 0),
    created_by text NOT NULL CHECK (char_length(created_by) BETWEEN 1 AND 200),
    created_at timestamptz NOT NULL,
    finished_at timestamptz,
    CHECK ((status IN ('done', 'failed')) = (finished_at IS NOT NULL)),
    CHECK ((status = 'failed') = (failure IS NOT NULL))
);

-- Newest first, and the hourly limit per user. A plain CREATE INDEX on a table made in the same
-- transaction blocks nobody.
CREATE INDEX IF NOT EXISTS analyses_created_at_idx ON analyses (created_at DESC);
CREATE INDEX IF NOT EXISTS analyses_created_by_created_at_idx ON analyses (created_by, created_at);
-- Recovery at startup reads the analyses that never finished.
CREATE INDEX IF NOT EXISTS analyses_unfinished_idx ON analyses (created_at) WHERE status IN ('queued', 'running');

CREATE TABLE IF NOT EXISTS analysis_pages (
    analysis_id uuid NOT NULL REFERENCES analyses (id) ON DELETE CASCADE,
    position bigint NOT NULL CHECK (position BETWEEN 0 AND 29),
    url text NOT NULL CHECK (char_length(url) <= 2000),
    title text NOT NULL CHECK (char_length(title) <= 500),
    description text NOT NULL CHECK (char_length(description) <= 1000),
    language text NOT NULL CHECK (char_length(language) <= 16),
    word_count bigint NOT NULL CHECK (word_count >= 0),
    PRIMARY KEY (analysis_id, position)
);

CREATE TABLE IF NOT EXISTS analysis_keywords (
    analysis_id uuid NOT NULL REFERENCES analyses (id) ON DELETE CASCADE,
    position bigint NOT NULL CHECK (position BETWEEN 0 AND 199),
    phrase text NOT NULL CHECK (char_length(phrase) BETWEEN 1 AND 200),
    language text NOT NULL CHECK (char_length(language) <= 16),
    score double precision NOT NULL CHECK (score >= 0),
    pages bigint NOT NULL CHECK (pages BETWEEN 1 AND 30),
    PRIMARY KEY (analysis_id, position)
);

CREATE TABLE IF NOT EXISTS analysis_known_terms (
    analysis_id uuid NOT NULL REFERENCES analyses (id) ON DELETE CASCADE,
    position bigint NOT NULL CHECK (position BETWEEN 0 AND 199),
    term text NOT NULL CHECK (char_length(term) BETWEEN 1 AND 200),
    language text NOT NULL
        CHECK (language IN ('de', 'en', 'en-US', 'nl', 'fr', 'pl', 'it', 'es', 'sv', 'da')),
    family text NOT NULL CHECK (family IN (
        'eyelink', 'open_spiral', 'closed_spiral', 'tri_flex_turn_curve', 'other', 'all', 'spec_all'
    )),
    tier text NOT NULL CHECK (tier IN ('spec', 'product', 'application', 'problem', 'broad')),
    occurrences bigint NOT NULL CHECK (occurrences >= 1),
    used_by_tribelt boolean NOT NULL,
    PRIMARY KEY (analysis_id, position)
);

CREATE TABLE IF NOT EXISTS analysis_equivalents (
    analysis_id uuid NOT NULL REFERENCES analyses (id) ON DELETE CASCADE,
    position bigint NOT NULL CHECK (position BETWEEN 0 AND 49),
    product text NOT NULL CHECK (char_length(product) BETWEEN 1 AND 300),
    family text NOT NULL
        CHECK (family IN ('eyelink', 'open_spiral', 'closed_spiral', 'tri_flex_turn_curve', 'other')),
    reason text NOT NULL CHECK (char_length(reason) <= 1000),
    PRIMARY KEY (analysis_id, position)
);

CREATE TABLE IF NOT EXISTS analysis_equivalent_terms (
    analysis_id uuid NOT NULL,
    equivalent_position bigint NOT NULL,
    position bigint NOT NULL CHECK (position BETWEEN 0 AND 29),
    term text NOT NULL CHECK (char_length(term) BETWEEN 1 AND 200),
    language text NOT NULL CHECK (char_length(language) <= 16),
    PRIMARY KEY (analysis_id, equivalent_position, position),
    FOREIGN KEY (analysis_id, equivalent_position)
        REFERENCES analysis_equivalents (analysis_id, position) ON DELETE CASCADE
);
