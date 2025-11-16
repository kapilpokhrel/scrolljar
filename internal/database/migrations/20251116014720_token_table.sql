-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS token (
    token_hash text PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES user(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    scope text NOT NULL,
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS token;
-- +goose StatementEnd
