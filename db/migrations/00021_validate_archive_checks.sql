-- +goose Up
-- A migration waits at most 5s for a lock and runs at most 60s, so it fails rather than stalls.
SET lock_timeout = '5s';
SET statement_timeout = '60s';

-- Validates the checks 00020 added NOT VALID; every row has the new columns empty.
ALTER TABLE companies VALIDATE CONSTRAINT companies_archived_by_check;
ALTER TABLE companies VALIDATE CONSTRAINT companies_archived_check;
ALTER TABLE companies VALIDATE CONSTRAINT companies_kept_by_check;
ALTER TABLE companies VALIDATE CONSTRAINT companies_kept_check;
ALTER TABLE company_overrides VALIDATE CONSTRAINT company_overrides_kept_by_check;
ALTER TABLE company_overrides VALIDATE CONSTRAINT company_overrides_kept_check;
