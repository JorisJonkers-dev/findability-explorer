-- +goose Up
-- A migration waits at most 5s for a lock and runs at most 60s, so it fails rather than stalls.
SET lock_timeout = '5s';
SET statement_timeout = '60s';

-- The term review and the review of the competitor candidates record the Claude model that
-- answered each request, and the work journal the model a piece of work used, as their steps may
-- now name another model than Claude Haiku 5.5; the month's Anthropic spending and Activity price
-- each at its model. A row from before has no model and is priced as Haiku, which every job asked
-- then. The Recommendation run records its model already. Nullable columns without a default or a
-- check rewrite and scan no row; the application keeps a model to a model id's length.
ALTER TABLE term_review_usage ADD COLUMN IF NOT EXISTS model text;
ALTER TABLE prompt_review_usage ADD COLUMN IF NOT EXISTS model text;
ALTER TABLE work_runs ADD COLUMN IF NOT EXISTS model text;
