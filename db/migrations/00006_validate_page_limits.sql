-- +goose Up
-- A migration waits at most 5s for a lock and runs at most 60s, so it fails rather than stalls.
SET lock_timeout = '5s';
SET statement_timeout = '60s';

-- The page limits 00005 widened to 100, checked against every existing row.
ALTER TABLE analyses VALIDATE CONSTRAINT analyses_max_pages_check;
ALTER TABLE analyses VALIDATE CONSTRAINT analyses_pages_crawled_check;
ALTER TABLE analysis_pages VALIDATE CONSTRAINT analysis_pages_position_check;
ALTER TABLE analysis_keywords VALIDATE CONSTRAINT analysis_keywords_pages_check;
