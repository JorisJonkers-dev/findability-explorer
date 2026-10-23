-- +goose Up
-- A migration waits at most 5s for a lock and runs at most 60s, so it fails rather than stalls.
SET lock_timeout = '5s';
SET statement_timeout = '60s';

-- Deeper scans: a crawl fetches up to 100 pages (internal/analysis/domain.MaxPages). The tables
-- are small and written only by this service; each check is swapped in place, and 00006 validates
-- them in a transaction of its own.
ALTER TABLE analyses DROP CONSTRAINT IF EXISTS analyses_max_pages_check;
ALTER TABLE analyses ADD CONSTRAINT analyses_max_pages_check CHECK (max_pages BETWEEN 1 AND 100) NOT VALID;
ALTER TABLE analyses DROP CONSTRAINT IF EXISTS analyses_pages_crawled_check;
ALTER TABLE analyses ADD CONSTRAINT analyses_pages_crawled_check CHECK (pages_crawled BETWEEN 0 AND 100) NOT VALID;
ALTER TABLE analysis_pages DROP CONSTRAINT IF EXISTS analysis_pages_position_check;
ALTER TABLE analysis_pages ADD CONSTRAINT analysis_pages_position_check CHECK (position BETWEEN 0 AND 99) NOT VALID;
ALTER TABLE analysis_keywords DROP CONSTRAINT IF EXISTS analysis_keywords_pages_check;
ALTER TABLE analysis_keywords ADD CONSTRAINT analysis_keywords_pages_check CHECK (pages BETWEEN 1 AND 100) NOT VALID;

-- The belt family a crawled page is about, when its title, headings and address make that clear.
ALTER TABLE analysis_pages ADD COLUMN IF NOT EXISTS family text CHECK (family IN (
    'eyelink', 'open_spiral', 'closed_spiral', 'tri_flex_turn_curve', 'other'
));

-- Scans are versions: each scan of a company has a number, 1 for its first, that never changes.
-- Existing scans are numbered in the order they were started.
ALTER TABLE analyses ADD COLUMN IF NOT EXISTS scan_number bigint CHECK (scan_number >= 1);
UPDATE analyses
SET scan_number = numbered.n
FROM (
    SELECT id, row_number() OVER (PARTITION BY competitor_id ORDER BY created_at, id) AS n
    FROM analyses
    WHERE competitor_id IS NOT NULL
) AS numbered
WHERE analyses.id = numbered.id AND analyses.scan_number IS NULL;
-- squawk-ignore require-concurrent-index-creation
CREATE UNIQUE INDEX IF NOT EXISTS analyses_competitor_scan_number_idx ON analyses (competitor_id, scan_number);

-- A scan queued by an administrator's re-scan of many companies, which the hourly limit on single
-- scans leaves out.
ALTER TABLE analyses ADD COLUMN IF NOT EXISTS rescan boolean NOT NULL DEFAULT false;

-- An administrator's changes to a researched company, which the binary embeds and never changes:
-- another name or website, or archived. Added companies are changed in their own row.
CREATE TABLE IF NOT EXISTS company_overrides (
    id text PRIMARY KEY CHECK (id ~ '^[a-z0-9]+(-[a-z0-9]+)*$' AND char_length(id) <= 80),
    name text CHECK (char_length(name) BETWEEN 1 AND 200),
    website_url text CHECK (char_length(website_url) BETWEEN 8 AND 500),
    host text UNIQUE CHECK (char_length(host) BETWEEN 1 AND 253),
    archived_by text CHECK (char_length(archived_by) BETWEEN 1 AND 200),
    archived_at timestamptz,
    updated_by text NOT NULL CHECK (char_length(updated_by) BETWEEN 1 AND 200),
    updated_at timestamptz NOT NULL,
    CHECK ((website_url IS NULL) = (host IS NULL)),
    CHECK ((archived_at IS NULL) = (archived_by IS NULL))
);

-- The review queue (internal/terminology, docs/adr/0003): an administrator's decision on a phrase
-- scans found outside the terminology map, keyed by the phrase's id, and the term an accepted one
-- became. Undoing a decision deletes its row, and the term with it.
CREATE TABLE IF NOT EXISTS term_decisions (
    candidate_id text PRIMARY KEY CHECK (candidate_id ~ '^[0-9a-f]{16}$'),
    phrase text NOT NULL CHECK (char_length(phrase) BETWEEN 1 AND 200),
    status text NOT NULL CHECK (status IN ('accepted', 'rejected')),
    decided_by text NOT NULL CHECK (char_length(decided_by) BETWEEN 1 AND 200),
    decided_at timestamptz NOT NULL
);

CREATE TABLE IF NOT EXISTS accepted_terms (
    candidate_id text PRIMARY KEY REFERENCES term_decisions (candidate_id) ON DELETE CASCADE,
    term text NOT NULL CHECK (char_length(term) BETWEEN 1 AND 200),
    language text NOT NULL
        CHECK (language IN ('de', 'en', 'en-US', 'nl', 'fr', 'pl', 'it', 'es', 'sv', 'da')),
    family text NOT NULL CHECK (family IN (
        'eyelink', 'open_spiral', 'closed_spiral', 'tri_flex_turn_curve', 'other', 'all', 'spec_all'
    )),
    tier text NOT NULL CHECK (tier IN ('spec', 'product', 'application', 'problem', 'broad')),
    used_by text [] NOT NULL CHECK (cardinality(used_by) <= 20),
    source text NOT NULL CHECK (char_length(source) <= 500)
);
