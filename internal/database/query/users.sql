-- name: InsertUser :one
INSERT INTO user_account (username, email, password_hash)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetUserByID :one
SELECT id, username, email, password_hash, activated, created_at, updated_at
FROM user_account
WHERE id = $1;

-- name: GetUserByEmail :one
SELECT id, username, email, password_hash, activated, created_at, updated_at
FROM user_account
WHERE email = $1;

-- name: UpdateUser :one
UPDATE user_account
SET username = $1, email = $2, activated = $3, password_hash = $4
WHERE id = $5 AND updated_at = $6
RETURNING updated_at;
