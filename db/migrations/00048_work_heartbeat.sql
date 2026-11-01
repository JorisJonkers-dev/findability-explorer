-- +goose Up
-- A migration waits at most 5s for a lock and runs at most 60s, so it fails rather than stalls.
SET lock_timeout = '5s';
SET statement_timeout = '60s';

-- Work that runs in memory beats while it runs (internal/shared/work.Heartbeat), so a starting or
-- sweeping instance ends only work whose server has stopped, never work a live pod still runs during
-- a rolling deploy. Null for work begun before this: the old pod does not beat, and such work counts
-- as left behind only an hour after it began.
ALTER TABLE work_runs ADD COLUMN IF NOT EXISTS heartbeat_at timestamptz;

-- A Recommendation run keeps its running state and a stop request in the table, not in memory, so a
-- stop that reaches another instance stops it, and a run whose server stopped is ended as failed. The
-- columns are nullable without a default: a change of the catalogue only.
ALTER TABLE recommendation_runs ADD COLUMN IF NOT EXISTS heartbeat_at timestamptz;
ALTER TABLE recommendation_runs ADD COLUMN IF NOT EXISTS stop_requested_at timestamptz;
