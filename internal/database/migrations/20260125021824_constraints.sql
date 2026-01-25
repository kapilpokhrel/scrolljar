-- +goose Up
-- +goose StatementBegin
ALTER TABLE user_account ADD CONSTRAINT user_creation_not_in_future CHECK(created_at <= now());
ALTER TABLE user_account ADD CONSTRAINT user_modified_not_in_future CHECK(updated_at <= now());

ALTER TABLE scrolljar ADD CONSTRAINT jar_creation_not_in_future CHECK(created_at <= now());
ALTER TABLE scrolljar ADD CONSTRAINT jar_modified_not_in_future CHECK(updated_at <= now());
ALTER TABLE scrolljar ADD CONSTRAINT min_expiry_check CHECK(
    expires_at IS NULL OR expires_at >= created_at + INTERVAL '5 minutes'
);
ALTER TABLE scrolljar ADD CONSTRAINT password_not_null CHECK(
    access = 0 OR password_hash IS NOT NULL
);


ALTER TABLE scroll ADD CONSTRAINT scroll_creation_not_in_future CHECK(created_at <= now());
ALTER TABLE scroll ADD CONSTRAINT scroll_modified_not_in_future CHECK(updated_at <= now());
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE user_account DROP CONSTRAINT IF EXISTS user_creation_not_in_future;
ALTER TABLE user_account DROP CONSTRAINT IF EXISTS user_modified_not_in_future;

ALTER TABLE scrolljar DROP CONSTRAINT IF EXISTS jar_creation_not_in_future;
ALTER TABLE scrolljar DROP CONSTRAINT IF EXISTS jar_modified_not_in_future;
ALTER TABLE scrolljar DROP CONSTRAINT IF EXISTS min_expiry_check;
ALTER TABLE scrolljar DROP CONSTRAINT IF EXISTS password_not_null;

ALTER TABLE scroll DROP CONSTRAINT IF EXISTS scroll_creation_not_in_future;
ALTER TABLE scroll DROP CONSTRAINT IF EXISTS scroll_modified_not_in_future;
-- +goose StatementEnd