-- +goose Up
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION delete_empty_scrolljar()
RETURNS TRIGGER AS $$
DECLARE
  affected_jar_id TEXT;
  still_exists BOOLEAN;
BEGIN
  affected_jar_id := OLD.jar_id;

  SELECT EXISTS (
    SELECT 1 FROM scroll WHERE jar_id = affected_jar_id LIMIT 1
  ) INTO still_exists;

  IF NOT still_exists THEN
    DELETE FROM scrolljar WHERE id = affected_jar_id;
  END IF;

  RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER delete_empty_scrolljar_trigger
AFTER DELETE ON scroll
FOR EACH ROW
EXECUTE FUNCTION delete_empty_scrolljar();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS delete_empty_scrolljar_trigger ON scroll;
DROP FUNCTION IF EXISTS delete_empty_scrolljar();
-- +goose StatementEnd
