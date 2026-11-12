-- +goose Up
-- A migration waits at most 5s for a lock and runs at most 60s, so it fails rather than stalls.
SET lock_timeout = '5s';
SET statement_timeout = '60s';

-- Reading a response tells claims wrong about Tribelt apart from facts it left out (the spec's
-- decision 17): wrong_claims keeps the claims the fact sheet contradicts, the only ones that
-- become fact sheet corrections, and left_out holds the facts from the fact sheet a response
-- naming Tribelt does not mention, at most ten as for wrong claims. Readings stored so far mixed
-- the two in wrong_claims and have no left-out list. A constant default rewrites no row.
ALTER TABLE prompt_extractions ADD COLUMN IF NOT EXISTS left_out text [] NOT NULL DEFAULT '{}'
CHECK (cardinality(left_out) <= 10);
