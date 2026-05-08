-- name: UpsertToken :exec
INSERT INTO token (token_hash, user_id, expires_at, scope)
VALUES ($1, $2, $3, $4)
ON CONFLICT ON CONSTRAINT unique_user_scope_token
DO UPDATE SET
    token_hash = $1,
    expires_at = $3;

-- name: GetTokenByHash :one
SELECT user_id, scope, expires_at
FROM token
WHERE token_hash = $1 AND (expires_at IS NULL OR expires_at > now());

-- name: DeleteTokenByHash :exec
DELETE FROM token WHERE token_hash = $1;

-- name: DeleteUserTokens :exec
DELETE FROM token WHERE user_id = $1;

-- name: DeleteExpiredTokens :exec
DELETE FROM token WHERE expires_at <= now();
