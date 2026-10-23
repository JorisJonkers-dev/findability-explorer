-- name: UpsertAccount :one
INSERT INTO accounts (issuer, subject, name, email, roles, last_login_at)
VALUES (@issuer, @subject, @name, @email, @roles::text[], @at)
ON CONFLICT (issuer, subject) DO UPDATE
SET name = excluded.name, email = excluded.email, roles = excluded.roles, last_login_at = excluded.last_login_at
RETURNING id;

-- name: UpdateAccountIdentity :exec
UPDATE accounts
SET name = @name, email = @email, roles = @roles::text[]
WHERE id = @id;

-- name: CreateSession :exec
INSERT INTO sessions (id, account_id, refresh_token, key_id, user_agent, created_at, last_seen_at, roles_checked_at, expires_at)
VALUES (@id, @account_id, @refresh_token, @key_id, @user_agent, @at, @at, @at, @expires_at);

-- name: GetSession :one
SELECT s.id, s.user_agent, s.created_at, s.last_seen_at, s.roles_checked_at, s.expires_at,
    a.id AS account_id, a.issuer, a.subject, a.name, a.email, a.roles
FROM sessions AS s
JOIN accounts AS a ON a.id = s.account_id
WHERE s.id = @id AND s.expires_at > @now;

-- name: LockSession :one
SELECT id, account_id, refresh_token, key_id, roles_checked_at
FROM sessions
WHERE id = @id AND expires_at > @now
FOR UPDATE;

-- name: SaveSessionRefresh :exec
UPDATE sessions
SET refresh_token = @refresh_token, key_id = @key_id, roles_checked_at = @at, last_seen_at = @at
WHERE id = @id;

-- name: TouchSession :exec
UPDATE sessions
SET last_seen_at = @at
WHERE id = @id AND last_seen_at < @stale_before;

-- name: DeleteSession :one
DELETE FROM sessions
WHERE id = @id
RETURNING refresh_token, key_id;

-- name: DeleteExpiredSessions :exec
DELETE FROM sessions
WHERE expires_at <= @now;
