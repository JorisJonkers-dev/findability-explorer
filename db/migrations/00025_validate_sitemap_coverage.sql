-- +goose Up
-- A migration waits at most 5s for a lock and runs at most 60s, so it fails rather than stalls.
SET lock_timeout = '5s';
SET statement_timeout = '60s';

-- Validates the check 00024 added NOT VALID; every readiness stored so far has neither count.
ALTER TABLE analysis_readiness VALIDATE CONSTRAINT analysis_readiness_sitemap_coverage_check;
