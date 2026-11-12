-- +goose Up
-- A migration waits at most 5s for a lock and runs at most 60s, so it fails rather than stalls.
SET lock_timeout = '5s';
SET statement_timeout = '60s';

-- An administrator stops a scan by hand (the spec's decision 5): a waiting one is recorded at once
-- as failed with the reason "Stopped by hand. The pages read so far are kept.", a running one is
-- asked to stop here, and the instance running it, which may be another, sees the request between
-- pages and records the same. Null for a scan nobody stopped. Adding a nullable column without a
-- default rewrites no row.
ALTER TABLE analyses ADD COLUMN IF NOT EXISTS stop_requested_at timestamptz;
