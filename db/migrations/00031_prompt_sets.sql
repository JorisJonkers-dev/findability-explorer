-- +goose Up
-- A migration waits at most 5s for a lock and runs at most 60s, so it fails rather than stalls.
SET lock_timeout = '5s';
SET statement_timeout = '60s';

-- The Prompt set as data (the redesign spec, section 11.3): published versions with a note, their
-- questions, and the scopes they are asked in. New tables only, so the old pod serves on.

-- A scope: a language a question is asked in, with or without a region. Its id is what an answer
-- records as its scope.
CREATE TABLE IF NOT EXISTS prompt_scopes (
    id text PRIMARY KEY CHECK (id ~ '^[a-z][a-z0-9_-]{0,39}$'),
    language text NOT NULL CHECK (language ~ '^[a-z]{2}$'),
    region text CHECK (char_length(region) BETWEEN 1 AND 100),
    name text NOT NULL UNIQUE CHECK (char_length(name) BETWEEN 1 AND 100)
);

-- A published version of the Prompt set. A published version never changes; a change is a new one.
CREATE TABLE IF NOT EXISTS prompt_sets (
    version bigint PRIMARY KEY CHECK (version BETWEEN 1 AND 100000),
    published_at timestamptz NOT NULL,
    published_by text NOT NULL CHECK (char_length(published_by) BETWEEN 1 AND 200),
    note text NOT NULL CHECK (char_length(note) BETWEEN 1 AND 300)
);

-- A question of a version, in order: its belt family, buyer intent, scope, and text, with its
-- English translation when it is in another language and one is known. Its prompt id is what
-- answers record, so a version's answers stay linked to its questions.
CREATE TABLE IF NOT EXISTS prompt_set_questions (
    version bigint NOT NULL REFERENCES prompt_sets (version),
    position bigint NOT NULL CHECK (position BETWEEN 1 AND 1000),
    prompt_id text NOT NULL CHECK (char_length(prompt_id) BETWEEN 1 AND 80),
    family text NOT NULL CHECK (family IN ('eyelink', 'open_spiral', 'closed_spiral', 'tri_flex_turn_curve', 'other')),
    intent text NOT NULL CHECK (intent IN ('spec', 'product', 'application', 'problem', 'broad')),
    scope_id text NOT NULL REFERENCES prompt_scopes (id),
    text text NOT NULL CHECK (char_length(text) BETWEEN 1 AND 1000),
    english text CHECK (char_length(english) BETWEEN 1 AND 1000),
    PRIMARY KEY (version, prompt_id),
    UNIQUE (version, position)
);
CREATE INDEX IF NOT EXISTS prompt_set_questions_scope_idx ON prompt_set_questions (scope_id);
