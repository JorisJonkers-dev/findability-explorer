-- +goose Up
-- A migration waits at most 5s for a lock and runs at most 60s, so it fails rather than stalls.
SET lock_timeout = '5s';
SET statement_timeout = '60s';

-- Every Claude step keeps its model and its instructions (the system message) as numbered versions
-- (the spec's section 8). A step's newest version is the one its next job uses; until the first
-- save there is no row, and the step uses the instructions it came with on Claude Haiku 5.5. The
-- first save stores that original as version 1 before the new one, so the history keeps it.
CREATE TABLE IF NOT EXISTS claude_step_versions (
    step text NOT NULL CHECK (char_length(step) BETWEEN 1 AND 50),
    number integer NOT NULL CHECK (number >= 1),
    model text NOT NULL CHECK (char_length(model) BETWEEN 1 AND 100),
    instructions text NOT NULL CHECK (char_length(instructions) BETWEEN 1 AND 20000),
    note text NOT NULL DEFAULT '' CHECK (char_length(note) <= 300),
    created_by text NOT NULL CHECK (char_length(created_by) BETWEEN 1 AND 200),
    created_at timestamptz NOT NULL,
    PRIMARY KEY (step, number)
);

