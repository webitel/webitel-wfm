-- +goose Up
-- +goose StatementBegin
CREATE SCHEMA wfm;

CREATE OR REPLACE FUNCTION wfm.tg_populate_updated_at_column()
    RETURNS TRIGGER AS
$$
BEGIN
    NEW.updated_at = now()::timestamp AT TIME ZONE 'UTC';
    RETURN NEW;
END;
$$ language 'plpgsql';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP FUNCTION wfm.tg_populate_updated_at_column;

DROP SCHEMA wfm CASCADE;
-- +goose StatementEnd
