-- +goose Up
-- A migration waits at most 5s for a lock and runs at most 60s, so it fails rather than stalls.
SET lock_timeout = '5s';
SET statement_timeout = '60s';

-- The provider checks 00009 widened, checked against every existing row.
ALTER TABLE llm_credentials VALIDATE CONSTRAINT llm_credentials_provider_check;
ALTER TABLE llm_events VALIDATE CONSTRAINT llm_events_provider_check;
