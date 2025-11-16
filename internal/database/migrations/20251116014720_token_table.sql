-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS token (
    token_hash bytea PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    scope text NOT NULL,
    CONSTRAINT unique_user_scope_token UNIQUE (user_id, scope)
);

CREATE OR REPLACE FUNCTION delete_expired_token()
RETURNS TRIGGER AS $$
BEGIN
  DELETE FROM token WHERE expires_at IS NOT NULL AND expires_at <= NOW();
  RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER delete_expired_token_trigger
BEFORE INSERT OR UPDATE ON token 
FOR EACH STATEMENT
EXECUTE FUNCTION delete_expired_token();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS token;

DROP TRIGGER IF EXISTS delete_empty_token_trigger ON token;
DROP FUNCTION IF EXISTS delete_expired_token();
-- +goose StatementEnd
