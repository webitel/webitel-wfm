-- +goose Up
-- +goose StatementBegin
CREATE TABLE wfm.working_schedule
(
    id                     SERIAL PRIMARY KEY,
    domain_id              BIGINT                                                                  NOT NULL,
    created_at             TIMESTAMP WITH TIME ZONE DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC') NOT NULL,
    created_by             BIGINT                                                                  NOT NULL,
    updated_at             TIMESTAMP WITH TIME ZONE DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC') NOT NULL,
    updated_by             BIGINT                                                                  NOT NULL,

    name                   TEXT                                                                    NOT NULL,
    state                  INT2                                                                    NOT NULL DEFAULT 1,
    team_id                BIGINT                                                                  NOT NULL,
    calendar_id            BIGINT                                                                  NOT NULL,

    start_date_at          DATE                     DEFAULT (CURRENT_DATE AT TIME ZONE 'UTC')      NOT NULL,
    end_date_at            DATE                     DEFAULT (CURRENT_DATE AT TIME ZONE 'UTC')      NOT NULL,
    start_time_at          INT2                     DEFAULT 0,
    end_time_at            INT2                     DEFAULT 1440,

    block_outside_activity BOOLEAN                  DEFAULT false,

    UNIQUE (domain_id, id),
    FOREIGN KEY (domain_id) REFERENCES directory.wbt_domain (dc) ON DELETE CASCADE,
    FOREIGN KEY (domain_id, created_by) REFERENCES directory.wbt_user (dc, id) ON DELETE SET NULL (created_by),
    FOREIGN KEY (domain_id, updated_by) REFERENCES directory.wbt_user (dc, id) ON DELETE SET NULL (updated_by),
    FOREIGN KEY (domain_id, team_id) REFERENCES call_center.cc_team (domain_id, id) ON DELETE SET NULL (team_id),
    FOREIGN KEY (domain_id, calendar_id) REFERENCES flow.calendar (domain_id, id) ON DELETE SET NULL (calendar_id),

    CHECK ( char_length(name) <= 250 ),
    CHECK ( start_date_at < end_date_at ),
    CHECK ( start_time_at BETWEEN 0 AND 1440 ),
    CHECK ( end_time_at BETWEEN 0 AND 1440 ),
    CHECK ( start_time_at < end_time_at )
);

CREATE TRIGGER tg_populate_updated_at_column
    BEFORE UPDATE
    ON wfm.working_schedule
    FOR EACH ROW
EXECUTE PROCEDURE wfm.tg_populate_updated_at_column();

CREATE TABLE wfm.working_schedule_extra_skill
(
    id                  SERIAL PRIMARY KEY,
    domain_id           BIGINT NOT NULL,
    working_schedule_id BIGINT,
    skill_id            BIGINT,

    UNIQUE (domain_id, id),
    FOREIGN KEY (domain_id) REFERENCES directory.wbt_domain (dc) ON DELETE CASCADE,
    FOREIGN KEY (domain_id, working_schedule_id) REFERENCES wfm.working_schedule (domain_id, id) ON DELETE CASCADE,
    FOREIGN KEY (domain_id, skill_id) REFERENCES call_center.cc_skill (domain_id, id) ON DELETE CASCADE
);

CREATE TABLE wfm.working_schedule_agent
(
    id                  SERIAL PRIMARY KEY,
    domain_id           BIGINT NOT NULL,
    working_schedule_id BIGINT,
    agent_id            BIGINT,

    UNIQUE (domain_id, id),
    FOREIGN KEY (domain_id) REFERENCES directory.wbt_domain (dc) ON DELETE CASCADE,
    FOREIGN KEY (domain_id, working_schedule_id) REFERENCES wfm.working_schedule (domain_id, id) ON DELETE CASCADE,
    FOREIGN KEY (domain_id, agent_id) REFERENCES call_center.cc_agent (domain_id, id) ON DELETE CASCADE
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE wfm.working_schedule_agent;

DROP TABLE wfm.working_schedule_extra_skill;

DROP TRIGGER tg_populate_updated_at_column ON wfm.working_schedule;

DROP TABLE wfm.working_schedule;
-- +goose StatementEnd
