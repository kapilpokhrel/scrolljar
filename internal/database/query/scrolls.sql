-- name: InsertScroll :one
INSERT INTO scroll (id, jar_id, title, format)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetScroll :one
SELECT s.id, s.jar_id, s.title, s.format, s.uploaded, s.created_at, s.updated_at
FROM scroll s
JOIN scrolljar j ON j.id = s.jar_id
WHERE s.id = $1 AND (j.expires_at IS NULL OR j.expires_at > now());

-- name: GetScrollsByJar :many
SELECT s.id, s.jar_id, s.title, s.format, s.uploaded, s.created_at, s.updated_at
FROM scroll s
JOIN scrolljar j ON j.id = s.jar_id
WHERE s.jar_id = $1 AND s.uploaded = TRUE AND (j.expires_at IS NULL OR j.expires_at > now());

-- name: UpdateScroll :one
UPDATE scroll
SET title = $1, format = $2
WHERE id = $3 AND updated_at = $4
RETURNING updated_at;

-- name: SetScrollUploaded :one
UPDATE scroll
SET uploaded = TRUE
WHERE id = $1 AND updated_at = $2
RETURNING updated_at;

-- name: DeleteScroll :exec
DELETE FROM scroll WHERE id = $1;

-- name: GetExistingScrollIDs :many
SELECT id FROM scroll WHERE id = ANY($1::TEXT[]);
