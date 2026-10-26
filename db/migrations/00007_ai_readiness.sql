-- +goose Up
-- A migration waits at most 5s for a lock and runs at most 60s, so it fails rather than stalls.
SET lock_timeout = '5s';
SET statement_timeout = '60s';

-- A scan's AI readiness (internal/analysis/domain.Readiness): what the site's robots.txt,
-- sitemaps, llms.txt, llms-full.txt and home page tell search engines and AI assistants, read
-- before the page crawl, so a site that blocks the crawl still has one. Sitemaps are kept only as
-- their summary. Scans made before this have no row and read as not checked.
CREATE TABLE IF NOT EXISTS analysis_readiness (
    analysis_id uuid PRIMARY KEY REFERENCES analyses (id) ON DELETE CASCADE,
    robots text NOT NULL CHECK (robots IN ('found', 'missing', 'unreachable')),
    sitemap_declared boolean NOT NULL,
    sitemap_files bigint NOT NULL CHECK (sitemap_files BETWEEN 0 AND 100),
    sitemap_addresses bigint NOT NULL CHECK (sitemap_addresses >= 0),
    sitemap_languages text [] NOT NULL CHECK (cardinality(sitemap_languages) <= 3),
    llms_present boolean NOT NULL,
    llms_bytes bigint NOT NULL CHECK (llms_bytes >= 0),
    llms_links bigint NOT NULL CHECK (llms_links >= 0),
    llms_full_present boolean NOT NULL,
    markdown boolean NOT NULL,
    home_read boolean NOT NULL,
    structured_data text [] NOT NULL CHECK (cardinality(structured_data) <= 50),
    languages text [] NOT NULL CHECK (cardinality(languages) <= 50),
    cms text NOT NULL CHECK (char_length(cms) <= 60),
    seo_plugin text NOT NULL CHECK (char_length(seo_plugin) <= 60),
    bot_protection text NOT NULL CHECK (char_length(bot_protection) <= 60)
);

-- Each search and AI crawler's access, in the order the domain lists them; none when robots.txt
-- could not be read.
CREATE TABLE IF NOT EXISTS analysis_bot_rules (
    analysis_id uuid NOT NULL REFERENCES analysis_readiness (analysis_id) ON DELETE CASCADE,
    position bigint NOT NULL CHECK (position BETWEEN 0 AND 19),
    bot text NOT NULL CHECK (char_length(bot) BETWEEN 1 AND 40),
    access text NOT NULL CHECK (access IN ('allowed', 'partly', 'blocked')),
    named boolean NOT NULL,
    PRIMARY KEY (analysis_id, position)
);

-- The raw robots.txt, llms.txt and llms-full.txt the site served, for administrators to check
-- the classification against; each capped at 100 KB, empty when the site served none.
CREATE TABLE IF NOT EXISTS analysis_readiness_files (
    analysis_id uuid PRIMARY KEY REFERENCES analysis_readiness (analysis_id) ON DELETE CASCADE,
    robots_txt text NOT NULL CHECK (octet_length(robots_txt) <= 100000),
    robots_truncated boolean NOT NULL,
    llms_txt text NOT NULL CHECK (octet_length(llms_txt) <= 100000),
    llms_truncated boolean NOT NULL,
    llms_full_txt text NOT NULL CHECK (octet_length(llms_full_txt) <= 100000),
    llms_full_truncated boolean NOT NULL
);
