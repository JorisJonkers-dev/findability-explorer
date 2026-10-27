-- name: ListCompanies :many
SELECT id, name, website_url, host, added_by, added_at, archived_by, archived_at, kept_by, kept_at
FROM companies
ORDER BY added_at, id;

-- name: InsertCompany :exec
INSERT INTO companies (id, name, website_url, host, added_by, added_at)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: UpdateCompany :execrows
-- An added company's name and website, whether it is archived, and whether a person kept it.
UPDATE companies
SET name = @name, website_url = @website_url, host = @host,
    archived_by = sqlc.narg(archived_by), archived_at = sqlc.narg(archived_at),
    kept_by = sqlc.narg(kept_by), kept_at = sqlc.narg(kept_at)
WHERE id = @id;

-- name: DeleteCompany :execrows
DELETE FROM companies
WHERE id = $1;

-- name: ListCompanyOverrides :many
SELECT id, name, website_url, host, archived_by, archived_at, kept_by, kept_at, updated_by, updated_at
FROM company_overrides
ORDER BY id;

-- name: UpsertCompanyOverride :exec
INSERT INTO company_overrides (id, name, website_url, host, archived_by, archived_at, kept_by, kept_at, updated_by, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
ON CONFLICT (id) DO UPDATE
SET name = excluded.name, website_url = excluded.website_url, host = excluded.host,
    archived_by = excluded.archived_by, archived_at = excluded.archived_at,
    kept_by = excluded.kept_by, kept_at = excluded.kept_at,
    updated_by = excluded.updated_by, updated_at = excluded.updated_at;
