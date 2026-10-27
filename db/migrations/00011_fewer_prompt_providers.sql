-- +goose Up
-- A migration waits at most 5s for a lock and runs at most 60s, so it fails rather than stalls.
SET lock_timeout = '5s';
SET statement_timeout = '60s';

-- The Prompt test's providers drop Meta and xAI (docs/adr/0004). Settings that still hold the old
-- default list take the new one; a list an administrator changed is left as it is.
UPDATE prompt_settings
SET providers = ARRAY['openai', 'google', 'anthropic', 'mistralai', 'deepseek']
WHERE providers = ARRAY['openai', 'google', 'anthropic', 'mistralai', 'meta-llama', 'x-ai', 'deepseek'];
