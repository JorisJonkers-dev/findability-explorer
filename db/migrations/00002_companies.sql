-- +goose Up
-- A migration waits at most 5s for a lock and runs at most 60s, so it fails rather than stalls.
SET lock_timeout = '5s';
SET statement_timeout = '60s';

-- The companies users and scans added (internal/companies). The ones the desk research studied
-- are embedded in the binary, not stored; ids and hosts are unique across both, which the
-- application checks against the embedded ones.
CREATE TABLE IF NOT EXISTS companies (
    id text PRIMARY KEY CHECK (id ~ '^[a-z0-9]+(-[a-z0-9]+)*$' AND char_length(id) <= 80),
    name text NOT NULL CHECK (char_length(name) BETWEEN 1 AND 200),
    website_url text NOT NULL CHECK (char_length(website_url) BETWEEN 8 AND 500),
    host text NOT NULL UNIQUE CHECK (char_length(host) BETWEEN 1 AND 253),
    added_by text NOT NULL CHECK (char_length(added_by) BETWEEN 1 AND 200),
    added_at timestamptz NOT NULL
);

-- The company a scan belongs to: a researched company's id or a row above, so no foreign key.
-- Scans made before companies existed have none.
ALTER TABLE analyses ADD COLUMN IF NOT EXISTS competitor_id text
    CHECK (competitor_id ~ '^[a-z0-9]+(-[a-z0-9]+)*$');

-- A company's scans, newest first. The table is small and written only by this service, so a
-- plain index inside the migration's transaction blocks nobody for long.
-- How the terminology map classifies each keyword: the tier and family of its closest term, that
-- term, and whether the keyword is one of the map's own spellings. Keywords stored before this
-- migration have none and read as broad.
ALTER TABLE analysis_keywords ADD COLUMN IF NOT EXISTS tier text
    CHECK (tier IN ('spec', 'product', 'application', 'problem', 'broad'));
ALTER TABLE analysis_keywords ADD COLUMN IF NOT EXISTS family text CHECK (family IN (
    'eyelink', 'open_spiral', 'closed_spiral', 'tri_flex_turn_curve', 'other', 'all', 'spec_all'
));
ALTER TABLE analysis_keywords ADD COLUMN IF NOT EXISTS closest_term text CHECK (char_length(closest_term) <= 200);
ALTER TABLE analysis_keywords ADD COLUMN IF NOT EXISTS in_map boolean;

-- squawk-ignore require-concurrent-index-creation
CREATE INDEX IF NOT EXISTS analyses_competitor_created_at_idx ON analyses (competitor_id, created_at DESC);
