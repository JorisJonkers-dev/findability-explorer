-- name: ListCompanies :many
SELECT id, name, website_url, host, added_by, added_at
FROM companies
ORDER BY added_at, id;

-- name: InsertCompany :exec
INSERT INTO companies (id, name, website_url, host, added_by, added_at)
VALUES ($1, $2, $3, $4, $5, $6);
