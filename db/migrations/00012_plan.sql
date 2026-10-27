-- +goose Up
-- A migration waits at most 5s for a lock and runs at most 60s, so it fails rather than stalls.
SET lock_timeout = '5s';
SET statement_timeout = '60s';

-- The plan's improvements an administrator ticked off by hand (internal/insight): the ones a scan
-- cannot check. Each tick names its catalogue version, who ticked it and when; undoing it deletes
-- the row.
CREATE TABLE IF NOT EXISTS plan_ticks (
    improvement_id text PRIMARY KEY CHECK (improvement_id ~ '^[a-z0-9]+(-[a-z0-9]+)*$' AND char_length(improvement_id) <= 60),
    catalogue_version bigint NOT NULL CHECK (catalogue_version >= 1),
    ticked_by text NOT NULL CHECK (char_length(ticked_by) BETWEEN 1 AND 200),
    ticked_at timestamptz NOT NULL
);

-- The plan's targets an administrator set; a target without a row keeps its default.
CREATE TABLE IF NOT EXISTS plan_targets (
    target text PRIMARY KEY CHECK (char_length(target) BETWEEN 1 AND 60),
    value double precision NOT NULL CHECK (value > 0),
    set_by text NOT NULL CHECK (char_length(set_by) BETWEEN 1 AND 200),
    set_at timestamptz NOT NULL
);
