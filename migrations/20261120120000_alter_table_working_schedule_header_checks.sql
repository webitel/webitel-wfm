-- +goose Up
-- +goose StatementBegin
-- A schedule may last one day, and its time scale may run past midnight.
ALTER TABLE wfm.working_schedule
    DROP CONSTRAINT working_schedule_check,
    DROP CONSTRAINT working_schedule_check1,
    ADD CONSTRAINT working_schedule_check CHECK ( start_date_at <= end_date_at );
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE wfm.working_schedule
    DROP CONSTRAINT working_schedule_check,
    ADD CONSTRAINT working_schedule_check CHECK ( start_date_at < end_date_at ),
    ADD CONSTRAINT working_schedule_check1 CHECK ( start_time_at < end_time_at );
-- +goose StatementEnd
