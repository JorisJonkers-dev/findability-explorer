-- +goose Up
-- A migration waits at most 5s for a lock and runs at most 60s, so it fails rather than stalls.
SET lock_timeout = '5s';
SET statement_timeout = '60s';

-- An administrator publishes new versions of the Prompt set, and a version may ask in a scope of its
-- own (a language with or without a region) and with any buyer intent. The answers' checks named
-- the six imported scopes, Dutch, German and English, and three kinds; they are swapped for checks
-- of the form a scope's id, a language code, and a kind take. Every answer stored so far passes the
-- new checks, which are added NOT VALID, so new rows are checked at once without a scan of the
-- answers under the lock; a later migration validates them, in a transaction of its own.
-- squawk-ignore ban-drop-constraint -- the constraint is re-added by the next statement, and this migration is already applied
ALTER TABLE prompt_answers DROP CONSTRAINT IF EXISTS prompt_answers_scope_check;
ALTER TABLE prompt_answers ADD CONSTRAINT prompt_answers_scope_check
CHECK (scope ~ '^[a-z][a-z0-9_-]{0,39}$') NOT VALID;

-- squawk-ignore ban-drop-constraint -- the constraint is re-added by the next statement, and this migration is already applied
ALTER TABLE prompt_answers DROP CONSTRAINT IF EXISTS prompt_answers_language_check;
ALTER TABLE prompt_answers ADD CONSTRAINT prompt_answers_language_check
CHECK (language ~ '^[a-z]{2}$') NOT VALID;

-- squawk-ignore ban-drop-constraint -- the constraint is re-added by the next statement, and this migration is already applied
ALTER TABLE prompt_answers DROP CONSTRAINT IF EXISTS prompt_answers_kind_check;
ALTER TABLE prompt_answers ADD CONSTRAINT prompt_answers_kind_check
CHECK (kind IN ('supplier', 'application', 'problem', 'spec', 'broad')) NOT VALID;

