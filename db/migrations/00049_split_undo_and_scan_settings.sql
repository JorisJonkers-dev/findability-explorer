-- +goose Up
-- A migration waits at most 5s for a lock and runs at most 60s, so it fails rather than stalls.
SET lock_timeout = '5s';
SET statement_timeout = '60s';

-- A name split off a candidate remembers the candidate it was merged into, so Undo the split can
-- merge it back. Its check is added NOT VALID and validated by 00050 in a transaction of its own.
ALTER TABLE prompt_candidate_merges ADD COLUMN IF NOT EXISTS split_from text;
ALTER TABLE prompt_candidate_merges DROP CONSTRAINT IF EXISTS prompt_candidate_merges_split_from_check;
ALTER TABLE prompt_candidate_merges ADD CONSTRAINT prompt_candidate_merges_split_from_check
CHECK (split_from IS NULL OR split_from ~ '^[0-9a-f]{16}$') NOT VALID;

-- The scan settings an administrator keeps: how many pages a scan reads unless another number is
-- set. One row, written the first time anyone saves; without a row a scan reads 60 pages.
CREATE TABLE IF NOT EXISTS scan_settings (
    singleton boolean PRIMARY KEY DEFAULT true CHECK (singleton),
    default_pages integer NOT NULL CHECK (default_pages BETWEEN 1 AND 100),
    updated_by text NOT NULL CHECK (char_length(updated_by) BETWEEN 1 AND 200),
    updated_at timestamptz NOT NULL
);
