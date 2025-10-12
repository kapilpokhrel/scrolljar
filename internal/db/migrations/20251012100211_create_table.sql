-- +goose Up
-- +goose StatementBegin
CREATE TABLE scrolljar (
    id BIGSERIAL PRIMARY KEY,
    short_id CHAR(8) UNIQUE NOT NULL,
    slug VARCHAR(64),
    name TEXT,
    access smallint NOT NULL DEFAULT 0,
    tags TEXT[],
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE scroll (
    id BIGSERIAL PRIMARY KEY,
    jar_id BIGINT NOT NULL REFERENCES scrolljar(id) ON DELETE CASCADE,
    title TEXT,
    format TEXT,
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS scrolljar;
DROP TABLE IF EXISTS scroll;
-- +goose StatementEnd
