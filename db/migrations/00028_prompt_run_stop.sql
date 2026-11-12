-- +goose Up
-- A migration waits at most 5s for a lock and runs at most 60s, so it fails rather than stalls.
SET lock_timeout = '5s';
SET statement_timeout = '60s';

-- An administrator can stop a running Prompt run. The request is stored, not held in memory, so the
-- instance that runs it sees it at its next heartbeat, whichever instance received the request. A
-- nullable column without a default is a change of the catalogue only, so the old pod serves on.
ALTER TABLE prompt_runs ADD COLUMN IF NOT EXISTS stop_requested_at timestamptz;
