-- name: ListCompanies :many
SELECT id, name, website_url, host, added_by, added_at
FROM companies
ORDER BY added_at, id;

-- name: InsertCompany :exec
INSERT INTO companies (id, name, website_url, host, added_by, added_at)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: UpdateCompany :execrows
UPDATE companies
SET name = $2, website_url = $3, host = $4
WHERE id = $1;

-- name: DeleteCompany :execrows
DELETE FROM companies
WHERE id = $1;

-- name: ListCompanyOverrides :many
SELECT id, name, website_url, host, archived_by, archived_at, updated_by, updated_at
FROM company_overrides
ORDER BY id;

-- name: UpsertCompanyOverride :exec
INSERT INTO company_overrides (id, name, website_url, host, archived_by, archived_at, updated_by, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (id) DO UPDATE
SET name = excluded.name, website_url = excluded.website_url, host = excluded.host,
    archived_by = excluded.archived_by, archived_at = excluded.archived_at,
    updated_by = excluded.updated_by, updated_at = excluded.updated_at;
