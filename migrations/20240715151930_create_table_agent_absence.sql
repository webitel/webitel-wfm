-- +goose Up
-- +goose StatementBegin
CREATE TABLE wfm.agent_absence
(
    id              SERIAL PRIMARY KEY,
    domain_id       BIGINT                                                                  NOT NULL,
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC') NOT NULL,
    created_by      BIGINT                                                                  NOT NULL,
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC') NOT NULL,
    updated_by      BIGINT                                                                  NOT NULL,

    absent_at       DATE                     DEFAULT (CURRENT_DATE AT TIME ZONE 'UTC')      NOT NULL,
    agent_id        BIGINT                                                                  NOT NULL,
    absence_type_id BIGINT                                                                  NOT NULL,

    UNIQUE (domain_id, id),
    UNIQUE (absent_at, agent_id, absence_type_id),
    FOREIGN KEY (domain_id) REFERENCES directory.wbt_domain (dc) ON DELETE CASCADE,
    FOREIGN KEY (domain_id, created_by) REFERENCES directory.wbt_user (dc, id) ON DELETE SET NULL (created_by),
    FOREIGN KEY (domain_id, updated_by) REFERENCES directory.wbt_user (dc, id) ON DELETE SET NULL (updated_by),
    FOREIGN KEY (domain_id, agent_id) REFERENCES call_center.cc_agent (domain_id, id) ON DELETE CASCADE
);

CREATE TRIGGER tg_populate_updated_at_column
    BEFORE UPDATE
    ON wfm.agent_absence
    FOR EACH ROW
EXECUTE PROCEDURE wfm.tg_populate_updated_at_column();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER tg_populate_updated_at_column ON wfm.agent_absence;

DROP TABLE wfm.agent_absence;
-- +goose StatementEnd
