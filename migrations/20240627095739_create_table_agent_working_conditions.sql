-- +goose Up
-- +goose StatementBegin
CREATE TABLE wfm.agent_working_conditions
(
    id                   SERIAL PRIMARY KEY,
    domain_id            BIGINT                                                                  NOT NULL,
    updated_at           TIMESTAMP WITH TIME ZONE DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC') NOT NULL,
    updated_by           BIGINT                                                                  NOT NULL,

    agent_id             BIGINT                                                                  NOT NULL,
    working_condition_id BIGINT                                                                  NOT NULL,
    pause_template_id    BIGINT,

    UNIQUE (domain_id, agent_id),
    FOREIGN KEY (domain_id) REFERENCES directory.wbt_domain (dc) ON DELETE CASCADE,
    FOREIGN KEY (domain_id, updated_by) REFERENCES directory.wbt_user (dc, id) ON DELETE SET NULL (updated_by),
    FOREIGN KEY (domain_id, agent_id) REFERENCES call_center.cc_agent (domain_id, id) ON DELETE CASCADE,
    FOREIGN KEY (domain_id, working_condition_id) REFERENCES wfm.working_condition (domain_id, id) ON DELETE CASCADE,
    FOREIGN KEY (domain_id, pause_template_id) REFERENCES wfm.pause_template (domain_id, id) ON DELETE SET NULL (pause_template_id)
);

CREATE TRIGGER tg_populate_updated_at_column
    BEFORE UPDATE
    ON wfm.agent_working_conditions
    FOR EACH ROW
EXECUTE PROCEDURE wfm.tg_populate_updated_at_column();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER tg_populate_updated_at_column On wfm.agent_working_conditions;

DROP TABLE wfm.agent_working_conditions;
-- +goose StatementEnd
