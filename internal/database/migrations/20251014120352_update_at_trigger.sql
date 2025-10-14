-- +goose Up
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION set_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER set_scrolljar_timestamp_updated_at
BEFORE UPDATE ON scrolljar 
FOR EACH ROW
EXECUTE PROCEDURE set_timestamp();

CREATE TRIGGER set_scroll_timestamp_updated_at
BEFORE UPDATE ON scroll
FOR EACH ROW
EXECUTE PROCEDURE set_timestamp();


CREATE OR REPLACE FUNCTION ping_parent()
RETURNS TRIGGER AS $$
DECLARE
  parent_table text := TG_ARGV[0];
  fk_column text := TG_ARGV[1];
  affected_id text;
  query text;
BEGIN
  IF (TG_OP = 'DELETE') THEN
    EXECUTE format('SELECT ($1).%I', fk_column) INTO affected_id USING OLD;
  ELSE
    EXECUTE format('SELECT ($1).%I', fk_column) INTO affected_id USING NEW;
  END IF;

  -- "Fake" update on parent to trigger its timestamp update trigger
  query := format('UPDATE %I SET id = id WHERE id = $1', parent_table);
  EXECUTE query USING affected_id;

  RETURN NULL;
END;
$$ LANGUAGE plpgsql;


CREATE TRIGGER ping_scrolljar_on_scrollchange_trigger
AFTER INSERT OR UPDATE OR DELETE ON scroll 
FOR EACH ROW
EXECUTE FUNCTION ping_parent('scrolljar', 'jar_id');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS set_scrolljar_timestamp_updated_at ON scrolljar;
DROP TRIGGER IF EXISTS set_scroll_timestamp_updated_at ON scroll;
DROP TRIGGER IF EXISTS ping_scrolljar_on_scrollchange_trigger ON scroll;

DROP FUNCTION IF EXISTS set_timestamp();
DROP FUNCTION IF EXISTS ping_parent();
-- +goose StatementEnd
