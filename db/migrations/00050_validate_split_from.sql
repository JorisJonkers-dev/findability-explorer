-- +goose Up
-- A migration waits at most 5s for a lock and runs at most 60s, so it fails rather than stalls.
SET lock_timeout = '5s';
SET statement_timeout = '60s';

-- Validates the check 00049 added NOT VALID, in a transaction of its own: every stored row has no
-- split_from yet, so every row passes it.
ALTER TABLE prompt_candidate_merges VALIDATE CONSTRAINT prompt_candidate_merges_split_from_check;
