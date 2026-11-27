-- +goose Up
-- +goose StatementBegin
-- Days off may take the whole year, and the pause is checked against the workday in whole minutes.
ALTER TABLE wfm.working_condition
    DROP CONSTRAINT working_condition_check,
    DROP CONSTRAINT working_condition_check1,
    ADD CONSTRAINT working_condition_working_hours_check CHECK ( workday_hours * 60 >= pause_duration ),
    ADD CONSTRAINT working_condition_non_working_days_check
        CHECK ( coalesce(vacation, 0) + coalesce(sick_leaves, 0) + coalesce(days_off, 0) <= 365 );
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE wfm.working_condition
    DROP CONSTRAINT working_condition_working_hours_check,
    DROP CONSTRAINT working_condition_non_working_days_check,
    ADD CONSTRAINT working_condition_check CHECK ( (workday_hours >= (pause_duration / 60)) ),
    ADD CONSTRAINT working_condition_check1
        CHECK ( coalesce(vacation, 0) + coalesce(sick_leaves, 0) + coalesce(days_off, 0) < 365 );
-- +goose StatementEnd
