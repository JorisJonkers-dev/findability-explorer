-- +goose Up
-- A migration waits at most 5s for a lock and runs at most 60s, so it fails rather than stalls.
SET lock_timeout = '5s';
SET statement_timeout = '60s';

-- Scans keep more (the spec's decision 18). Every column is new, with a constant default, so no
-- row is rewritten and nothing is backfilled: scans made before this read as keeping nothing more.

-- The first sitemap a scan read (the one robots.txt lists, or /sitemap.xml), as served, capped
-- like the other files at 100 KB, with its address, for the files viewer.
ALTER TABLE analysis_readiness_files ADD COLUMN IF NOT EXISTS sitemap_url text NOT NULL DEFAULT ''
CHECK (char_length(sitemap_url) <= 2000);
ALTER TABLE analysis_readiness_files ADD COLUMN IF NOT EXISTS sitemap_txt text NOT NULL DEFAULT ''
CHECK (octet_length(sitemap_txt) <= 100000);
ALTER TABLE analysis_readiness_files ADD COLUMN IF NOT EXISTS sitemap_truncated boolean NOT NULL DEFAULT false;

-- The pages that use each known term, by their position among the scan's pages (at most ten).
ALTER TABLE analysis_known_terms ADD COLUMN IF NOT EXISTS pages bigint [] NOT NULL DEFAULT '{}'
CHECK (cardinality(pages) <= 10);

-- The visible text of each page of a scan of tribelt.nl, for the Recommendation run, capped at
-- 100 KB; empty for every other company's pages.
ALTER TABLE analysis_pages ADD COLUMN IF NOT EXISTS body text NOT NULL DEFAULT ''
CHECK (octet_length(body) <= 100000);
