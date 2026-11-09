-- +goose Up
-- A migration waits at most 5s for a lock and runs at most 60s, so it fails rather than stalls.
SET lock_timeout = '5s';
SET statement_timeout = '60s';

-- Validates the position checks 00016 added NOT VALID; every row already meets the narrower old ones.
ALTER TABLE analysis_keywords VALIDATE CONSTRAINT analysis_keywords_position_check;
ALTER TABLE analysis_known_terms VALIDATE CONSTRAINT analysis_known_terms_position_check;
