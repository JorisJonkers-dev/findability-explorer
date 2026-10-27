-- +goose Up
-- A migration waits at most 5s for a lock and runs at most 60s, so it fails rather than stalls.
SET lock_timeout = '5s';
SET statement_timeout = '60s';

-- An added company is archived as a researched one is (internal/companies, ADR-0010): it leaves
-- the list and the re-scans, its scans stay, and it can be restored. Who archived it is an
-- administrator or the language model.
ALTER TABLE companies ADD COLUMN IF NOT EXISTS archived_by text;
ALTER TABLE companies ADD COLUMN IF NOT EXISTS archived_at timestamptz;

-- A person who restored a company, or kept it on the list, so the language model never archives
-- it. Both tables hold it: an added company in its row, a researched one beside the research.
ALTER TABLE companies ADD COLUMN IF NOT EXISTS kept_by text;
ALTER TABLE companies ADD COLUMN IF NOT EXISTS kept_at timestamptz;
ALTER TABLE company_overrides ADD COLUMN IF NOT EXISTS kept_by text;
ALTER TABLE company_overrides ADD COLUMN IF NOT EXISTS kept_at timestamptz;

-- The checks on the new columns; 00021 validates them in a transaction of its own.
ALTER TABLE companies DROP CONSTRAINT IF EXISTS companies_archived_by_check;
ALTER TABLE companies ADD CONSTRAINT companies_archived_by_check
CHECK (char_length(archived_by) BETWEEN 1 AND 200) NOT VALID;
ALTER TABLE companies DROP CONSTRAINT IF EXISTS companies_archived_check;
ALTER TABLE companies ADD CONSTRAINT companies_archived_check
CHECK ((archived_at IS NULL) = (archived_by IS NULL)) NOT VALID;
ALTER TABLE companies DROP CONSTRAINT IF EXISTS companies_kept_by_check;
ALTER TABLE companies ADD CONSTRAINT companies_kept_by_check
CHECK (char_length(kept_by) BETWEEN 1 AND 200) NOT VALID;
ALTER TABLE companies DROP CONSTRAINT IF EXISTS companies_kept_check;
ALTER TABLE companies ADD CONSTRAINT companies_kept_check
CHECK ((kept_at IS NULL) = (kept_by IS NULL)) NOT VALID;
ALTER TABLE company_overrides DROP CONSTRAINT IF EXISTS company_overrides_kept_by_check;
ALTER TABLE company_overrides ADD CONSTRAINT company_overrides_kept_by_check
CHECK (char_length(kept_by) BETWEEN 1 AND 200) NOT VALID;
ALTER TABLE company_overrides DROP CONSTRAINT IF EXISTS company_overrides_kept_check;
ALTER TABLE company_overrides ADD CONSTRAINT company_overrides_kept_check
CHECK ((kept_at IS NULL) = (kept_by IS NULL)) NOT VALID;

-- The language model's check of a company on the competitor list (internal/prompttest, ADR-0010),
-- against Tribelt's target market as for a candidate: whether it competes, whether it was sure,
-- why, and the check version that made it. A sure "not a competitor" archives the company.
CREATE TABLE IF NOT EXISTS prompt_company_checks (
    company_id text PRIMARY KEY CHECK (company_id ~ '^[a-z0-9]+(-[a-z0-9]+)*$' AND char_length(company_id) <= 80),
    competitor boolean NOT NULL,
    sure boolean NOT NULL,
    reason text NOT NULL CHECK (char_length(reason) <= 500),
    model text NOT NULL CHECK (char_length(model) BETWEEN 1 AND 200),
    check_version bigint NOT NULL CHECK (check_version >= 1),
    checked_at timestamptz NOT NULL
);
