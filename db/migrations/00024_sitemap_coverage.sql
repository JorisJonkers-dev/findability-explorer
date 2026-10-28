-- +goose Up
-- A migration waits at most 5s for a lock and runs at most 60s, so it fails rather than stalls.
SET lock_timeout = '5s';
SET statement_timeout = '60s';

-- How much of a site its sitemaps cover (internal/analysis/domain.SitemapCoverage, ADR-0013): the
-- pages the scan read, and how many of them the sitemaps list. A scan counts them after its crawl,
-- a check again against the pages its scan read. Both are null for a readiness recorded before
-- this, or for a scan that read no page: its coverage was not measured. 00025 validates the check.
ALTER TABLE analysis_readiness ADD COLUMN IF NOT EXISTS sitemap_pages_read bigint;
ALTER TABLE analysis_readiness ADD COLUMN IF NOT EXISTS sitemap_pages_listed bigint;
ALTER TABLE analysis_readiness DROP CONSTRAINT IF EXISTS analysis_readiness_sitemap_coverage_check;
ALTER TABLE analysis_readiness ADD CONSTRAINT analysis_readiness_sitemap_coverage_check
CHECK (
    (sitemap_pages_read IS NULL AND sitemap_pages_listed IS NULL)
    OR (sitemap_pages_read > 0 AND sitemap_pages_listed BETWEEN 0 AND sitemap_pages_read)
) NOT VALID;
