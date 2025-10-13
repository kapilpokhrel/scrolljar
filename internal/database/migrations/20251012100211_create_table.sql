-- +goose Up
-- +goose StatementBegin
CREATE TABLE scrolljar (
    id BIGSERIAL PRIMARY KEY,
    slug CHAR(8) UNIQUE NOT NULL,
    name TEXT,
    access smallint NOT NULL DEFAULT 0,
    tags TEXT[],
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE scroll (
    id smallint NOT NULL,
    jar_id BIGINT NOT NULL REFERENCES scrolljar(id) ON DELETE CASCADE,
    title TEXT,
    format TEXT,
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now(),
    PRIMARY KEY(id, jar_id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS scroll;
DROP TABLE IF EXISTS scrolljar;
-- +goose StatementEnd
