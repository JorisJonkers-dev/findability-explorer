-- +goose Up
-- A migration waits at most 5s for a lock and runs at most 60s, so it fails rather than stalls.
SET lock_timeout = '5s';
SET statement_timeout = '60s';

-- Each reading of a response records the model that read it (the spec's section 8): the reading
-- step's model can change, and a change marks a break on the AI responses charts. A reading made
-- before this keeps an empty model; what read it then is not known. Adding a column with a
-- constant default rewrites no row.
ALTER TABLE prompt_extractions ADD COLUMN IF NOT EXISTS model text NOT NULL DEFAULT '';
