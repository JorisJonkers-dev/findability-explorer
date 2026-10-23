-- name: AddLlmCredential :exec
INSERT INTO llm_credentials (id, provider, label, secret, key_id, fingerprint, created_by, created_at)
VALUES (@id, @provider, @label, @secret, @key_id, @fingerprint, @created_by, @created_at);

-- name: ListLlmCredentials :many
SELECT id, provider, label, key_id, fingerprint, created_by, created_at
FROM llm_credentials
ORDER BY created_at, id;

-- name: GetLlmCredential :one
SELECT id, provider, label, secret, key_id, fingerprint, created_by, created_at
FROM llm_credentials
WHERE id = @id;

-- name: DeleteLlmCredential :execrows
DELETE FROM llm_credentials
WHERE id = @id;

-- name: GetLlmActive :one
SELECT mode, coalesce(credential_id, '')::text AS credential_id, coalesce(model, '')::text AS model
FROM llm_active
WHERE singleton;

-- name: SetLlmActive :exec
INSERT INTO llm_active (singleton, mode, credential_id, model, updated_by, updated_at)
VALUES (true, @mode, sqlc.narg(credential_id), sqlc.narg(model), @updated_by, @updated_at)
ON CONFLICT (singleton) DO UPDATE
SET mode = excluded.mode, credential_id = excluded.credential_id, model = excluded.model,
    updated_by = excluded.updated_by, updated_at = excluded.updated_at;

-- name: ResetLlmActiveFor :execrows
DELETE FROM llm_active
WHERE credential_id = @credential_id;

-- name: AddLlmEvent :exec
INSERT INTO llm_events (at, actor, action, credential_id, provider, model, ok, message)
VALUES (@at, @actor, @action, sqlc.narg(credential_id), sqlc.narg(provider), sqlc.narg(model), sqlc.narg(ok), @message);

-- name: ListLlmEvents :many
SELECT at, actor, action, credential_id, provider, model, ok, message
FROM llm_events
ORDER BY id DESC
LIMIT @lim;

-- name: LatestLlmTests :many
SELECT DISTINCT ON (credential_id) credential_id::text AS credential_id, at, ok::boolean AS ok, message
FROM llm_events
WHERE action IN ('credential_added', 'credential_tested') AND credential_id IS NOT NULL AND ok IS NOT NULL
ORDER BY credential_id, id DESC;
