-- +goose Up
-- A migration waits at most 5s for a lock and runs at most 60s, so it fails rather than stalls.
SET lock_timeout = '5s';
SET statement_timeout = '60s';

-- The Anthropic key's monthly budget in millionths of a dollar (internal/llm): one row at most.
-- Every job a Claude model does is priced at the model's list price from its stored token counts
-- and stops once the month's spend reaches it. No row means $15 until an admin sets one. It
-- replaces LLM_MONTHLY_TOKENS; the OpenRouter key's budget stays in prompt_settings.
CREATE TABLE IF NOT EXISTS llm_budget (
    singleton boolean PRIMARY KEY DEFAULT true CHECK (singleton),
    monthly_budget_micros bigint NOT NULL CHECK (monthly_budget_micros BETWEEN 10000 AND 1000000000),
    updated_by text NOT NULL CHECK (char_length(updated_by) BETWEEN 1 AND 200),
    updated_at timestamptz NOT NULL
);

-- A change to the budget is an event. The action check is swapped in place; the rows already
-- stored all pass it, and new rows are checked at once.
-- squawk-ignore ban-drop-constraint -- the constraint is re-added by the next statement, and this migration is already applied
ALTER TABLE llm_events DROP CONSTRAINT IF EXISTS llm_events_action_check;
ALTER TABLE llm_events ADD CONSTRAINT llm_events_action_check
CHECK (action IN (
    'credential_added', 'credential_deleted', 'credential_tested', 'active_set', 'active_cleared', 'budget_set'
)) NOT VALID;
