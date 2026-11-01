-- +goose Up
-- A migration waits at most 5s for a lock and runs at most 60s, so it fails rather than stalls.
SET lock_timeout = '5s';
SET statement_timeout = '60s';

-- The term review runs by itself every ten minutes until an administrator switches it off (the
-- spec's decision 21); Review now still runs it once. One row, written the first time anyone
-- switches it: without a row the review is on. Who switched it last, and when, is kept.
CREATE TABLE IF NOT EXISTS term_review_switch (
    singleton boolean PRIMARY KEY DEFAULT true CHECK (singleton),
    review_on boolean NOT NULL,
    updated_by text NOT NULL CHECK (char_length(updated_by) BETWEEN 1 AND 200),
    updated_at timestamptz NOT NULL
);
