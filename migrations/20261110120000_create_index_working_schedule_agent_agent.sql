-- +goose Up
-- +goose StatementBegin
-- The shift write (WFM-94) and the calendar lock (WFM-95) look an agent up across schedules.
CREATE INDEX working_schedule_agent_agent_idx
    ON wfm.working_schedule_agent (agent_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX wfm.working_schedule_agent_agent_idx;
-- +goose StatementEnd
