-- +goose Up
-- +goose StatementBegin
-- The id of the error a job failed with: the front tells a timeout from a forecast error or a bad result by it.
ALTER TABLE wfm.job
    ADD COLUMN error_code TEXT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE wfm.job
    DROP COLUMN error_code;
-- +goose StatementEnd
