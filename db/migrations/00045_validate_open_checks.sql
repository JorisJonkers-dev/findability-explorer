-- +goose Up
-- A migration waits at most 5s for a lock and runs at most 60s, so it fails rather than stalls.
SET lock_timeout = '5s';
SET statement_timeout = '60s';

-- Validates the checks 00033 and 00041 added NOT VALID, in a transaction of their own: the
-- events' action check that allows budget_set, and the answers' scope, language, and kind checks.
-- Every stored row passes them; the old checks they replaced were narrower.
ALTER TABLE llm_events VALIDATE CONSTRAINT llm_events_action_check;
ALTER TABLE prompt_answers VALIDATE CONSTRAINT prompt_answers_scope_check;
ALTER TABLE prompt_answers VALIDATE CONSTRAINT prompt_answers_language_check;
ALTER TABLE prompt_answers VALIDATE CONSTRAINT prompt_answers_kind_check;
