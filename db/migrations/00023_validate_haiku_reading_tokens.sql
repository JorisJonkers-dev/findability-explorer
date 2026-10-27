-- +goose Up
-- A migration waits at most 5s for a lock and runs at most 60s, so it fails rather than stalls.
SET lock_timeout = '5s';
SET statement_timeout = '60s';

-- Validates the check 00022 added NOT VALID; every stored extraction counts 0.
ALTER TABLE prompt_extractions VALIDATE CONSTRAINT prompt_extractions_haiku_tokens_check;
