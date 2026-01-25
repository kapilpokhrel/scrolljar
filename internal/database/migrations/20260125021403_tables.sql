-- +goose Up
-- +goose StatementBegin
CREATE EXTENSION IF NOT EXISTS citext;

CREATE TABLE IF NOT EXISTS user_account (
    id BIGSERIAL PRIMARY KEY,
    username TEXT NOT NULL,
    email citext UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    activated BOOL NOT NULL DEFAULT False,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS scrolljar (
    id CHAR(8) PRIMARY KEY,
    name TEXT,
    user_id BIGINT REFERENCES user_account(id) ON DELETE SET NULL,
    access SMALLINT NOT NULL DEFAULT 0,
    password_hash TEXT,
    tags TEXT[],
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS scroll (
    id CHAR(8) PRIMARY KEY,
    jar_id CHAR(8) NOT NULL REFERENCES scrolljar(id) ON DELETE CASCADE,
    title TEXT,
    format TEXT,
    uploaded BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS token (
    token_hash bytea PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES user_account(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    scope text NOT NULL,
    CONSTRAINT unique_user_scope_token UNIQUE (user_id, scope)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS token;
DROP TABLE IF EXISTS scroll;
DROP TABLE IF EXISTS scrolljar;
DROP TABLE IF EXISTS user_account;
-- +goose StatementEnd
