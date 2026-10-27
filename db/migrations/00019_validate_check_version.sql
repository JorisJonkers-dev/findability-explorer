-- +goose Up
-- A migration waits at most 5s for a lock and runs at most 60s, so it fails rather than stalls.
SET lock_timeout = '5s';
SET statement_timeout = '60s';

-- Validates the check 00018 added NOT VALID; every stored check is version 1.
ALTER TABLE prompt_candidate_checks VALIDATE CONSTRAINT prompt_candidate_checks_check_version_check;
