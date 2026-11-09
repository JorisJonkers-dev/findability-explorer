-- +goose Up
-- A migration waits at most 5s for a lock and runs at most 60s, so it fails rather than stalls.
SET lock_timeout = '5s';
SET statement_timeout = '60s';

-- A scan keeps up to 1,000 keywords and every map term it finds, no longer 200 of each, so no
-- phrase a site uses is cut off before review (internal/analysis/domain.MaxKeywords).
-- 00017 validates the new checks.
ALTER TABLE analysis_keywords DROP CONSTRAINT IF EXISTS analysis_keywords_position_check;
ALTER TABLE analysis_keywords ADD CONSTRAINT analysis_keywords_position_check
CHECK (position BETWEEN 0 AND 999) NOT VALID;

ALTER TABLE analysis_known_terms DROP CONSTRAINT IF EXISTS analysis_known_terms_position_check;
ALTER TABLE analysis_known_terms ADD CONSTRAINT analysis_known_terms_position_check
CHECK (position BETWEEN 0 AND 9999) NOT VALID;
