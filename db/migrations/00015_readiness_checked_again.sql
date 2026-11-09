-- +goose Up
-- A migration waits at most 5s for a lock and runs at most 60s, so it fails rather than stalls.
SET lock_timeout = '5s';
SET statement_timeout = '60s';

-- When an administrator checked a scan's AI readiness again, without crawling its pages or asking
-- the language model; null when the scan itself read it.
ALTER TABLE analysis_readiness ADD COLUMN IF NOT EXISTS checked_at timestamptz;
