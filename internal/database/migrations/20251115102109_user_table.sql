-- +goose Up
-- +goose StatementBegin
CREATE EXTENSION IF NOT EXISTS citext;

CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    username TEXT NOT NULL,
    email citext UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    activated BOOL NOT NULL DEFAULT False,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);

ALTER TABLE scrolljar 
ADD CONSTRAINT scrolljar_user_id_fkey
FOREIGN KEY (user_id) REFERENCES users(id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS users;

ALTER TABLE scrolljar 
DROP CONSTRAINT scrolljar_user_id_fkey;
-- +goose StatementEnd
