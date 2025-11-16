-- +goose Up
-- +goose StatementBegin
CREATE EXTENSION IF NOT EXISTS citext;

CREATE TABLE IF NOT EXISTS user (
    id BIGSERIAL PRIMARY KEY,
    username TEXT NOT NULL,
    email citext UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    activated BOOL NOT NULL DEFAULT False,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS user;
-- +goose StatementEnd
