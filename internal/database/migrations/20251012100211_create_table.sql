-- +goose Up
-- +goose StatementBegin
CREATE TABLE scrolljar (
    id CHAR(8) PRIMARY KEY,
    name TEXT,
    user_id BIGINT,
    access smallint NOT NULL DEFAULT 0,
    password_hash text,
    tags TEXT[],
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE scroll (
    id CHAR(8) PRIMARY KEY,
    jar_id CHAR(8) NOT NULL REFERENCES scrolljar(id) ON DELETE CASCADE,
    title TEXT,
    format TEXT,
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS scroll;
DROP TABLE IF EXISTS scrolljar;
-- +goose StatementEnd
