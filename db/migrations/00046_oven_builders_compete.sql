-- +goose Up
-- A migration waits at most 5s for a lock and runs at most 60s, so it fails rather than stalls.
SET lock_timeout = '5s';
SET statement_timeout = '60s';

-- Version 3 of the competitor definition: an oven, freezer or furnace builder that makes or sells
-- its own metal belts, also as spare belts for its machines, is a competitor (ADR-0009). It is
-- version 2 with that one sentence added. The newest version is the one in use, so companies whose
-- check was made against version 2 are checked again. Nothing happens if version 3 exists already.
INSERT INTO competitor_definitions (version, text, saved_at, saved_by)
SELECT 3,
    replace(text,
        'A maker anywhere in the world that sells into Europe counts.',
        'A maker anywhere in the world that sells into Europe counts. A builder of ovens, freezers or furnaces that makes or sells its own metal belts, also as spare belts for its machines, is a competitor too, because a buyer can take its belt instead of Tribelt’s.'),
    now(), 'import'
FROM competitor_definitions
WHERE version = 2
ON CONFLICT (version) DO NOTHING;
