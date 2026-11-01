-- +goose Up
-- A migration waits at most 5s for a lock and runs at most 60s, so it fails rather than stalls.
SET lock_timeout = '5s';
SET statement_timeout = '60s';

-- Search terms an administrator added to the list by hand (internal/terminology/domain.HandTerm):
-- the term, its language, belt family and buyer intent, and who added it when. Scans look for
-- them from the next scan on.
CREATE TABLE IF NOT EXISTS hand_terms (
    term text NOT NULL CHECK (char_length(term) BETWEEN 1 AND 200),
    language text NOT NULL CHECK (language IN ('de', 'en', 'en-US', 'nl', 'fr', 'pl', 'it', 'es', 'sv', 'da')),
    family text NOT NULL CHECK (family IN (
        'eyelink', 'open_spiral', 'closed_spiral', 'tri_flex_turn_curve', 'other', 'all', 'spec_all'
    )),
    tier text NOT NULL CHECK (tier IN ('spec', 'product', 'application', 'problem', 'broad')),
    added_by text NOT NULL CHECK (char_length(added_by) BETWEEN 1 AND 200),
    added_at timestamptz NOT NULL,
    PRIMARY KEY (term, language)
);

-- Search terms an administrator took off the list (domain.Removal), whatever their origin: no
-- longer counted or looked for until put back, which deletes the row.
CREATE TABLE IF NOT EXISTS removed_terms (
    term text NOT NULL CHECK (char_length(term) BETWEEN 1 AND 200),
    language text NOT NULL CHECK (language IN ('de', 'en', 'en-US', 'nl', 'fr', 'pl', 'it', 'es', 'sv', 'da')),
    removed_by text NOT NULL CHECK (char_length(removed_by) BETWEEN 1 AND 200),
    removed_at timestamptz NOT NULL,
    PRIMARY KEY (term, language)
);
