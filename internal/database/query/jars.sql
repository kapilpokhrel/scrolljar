-- name: GetJar :one
SELECT id, name, user_id, access, password_hash, tags, expires_at, created_at, updated_at
FROM scrolljar
WHERE id = $1 AND (expires_at IS NULL OR expires_at > now());

-- name: GetJarOwnerID :one
SELECT user_id FROM scrolljar
WHERE id = $1 AND (expires_at IS NULL OR expires_at > now());

-- name: GetJarsByUser :many
SELECT id, name, user_id, access, password_hash, tags, expires_at, created_at, updated_at
FROM scrolljar
WHERE user_id = $1 AND (expires_at IS NULL OR expires_at > now());

-- name: InsertJar :one
INSERT INTO scrolljar (id, user_id, name, access, password_hash, tags, expires_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: DeleteJar :exec
DELETE FROM scrolljar WHERE id = $1;

-- name: DeleteExpiredJars :exec
DELETE FROM scrolljar WHERE expires_at <= now();
