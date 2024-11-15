-- +goose Up
-- +goose StatementBegin
CREATE TABLE wfm.working_schedule
(
    id                     SERIAL PRIMARY KEY,
    domain_id              BIGINT                                                                  NOT NULL,
    created_at             TIMESTAMP WITH TIME ZONE DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC') NOT NULL,
    created_by             BIGINT,
    updated_at             TIMESTAMP WITH TIME ZONE DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC') NOT NULL,
    updated_by             BIGINT,

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
    FOREIGN KEY (domain_id, team_id) REFERENCES call_center.cc_team (domain_id, id) ON DELETE RESTRICT,
    FOREIGN KEY (domain_id, calendar_id) REFERENCES flow.calendar (domain_id, id) ON DELETE RESTRICT,

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

CREATE UNIQUE INDEX cc_skill_domain_id_udx on call_center.cc_skill USING btree (id, domain_id);

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

CREATE VIEW wfm.working_schedule_v AS
SELECT t.id                                      AS id
     , t.domain_id                               AS domain_id
     , t.created_at                              AS created_at
     , call_center.cc_get_lookup(c.id, c.name)   AS created_by
     , t.updated_at                              AS updated_at
     , call_center.cc_get_lookup(u.id, u.name)   AS updated_by
     , t.name                                    AS name
     , t.state                                   AS state
     , call_center.cc_get_lookup(at.id, at.name) AS team
     , call_center.cc_get_lookup(ca.id, ca.name) AS calendar
     , t.start_date_at                           AS start_date_at
     , t.end_date_at                             AS end_date_at
     , t.start_time_at                           AS start_time_at
     , t.end_time_at                             AS end_time_at
     , t.block_outside_activity                  AS block_outside_activity
     , ag.agents                                 AS agents
     , sg.skills                                 AS extra_skills
FROM wfm.working_schedule t
         LEFT JOIN directory.wbt_user c ON t.created_by = c.id
         LEFT JOIN directory.wbt_user u ON t.updated_by = u.id
         LEFT JOIN call_center.cc_team at ON t.team_id = at.id
         LEFT JOIN flow.calendar ca ON t.calendar_id = ca.id
         LEFT JOIN LATERAL (
    SELECT jsonb_agg(call_center.cc_get_lookup(a.id, au.name)) AS agents
    FROM wfm.working_schedule_agent wa
             INNER JOIN call_center.cc_agent a on wa.agent_id = a.id
             INNER JOIN directory.wbt_user au ON a.user_id = au.id
    WHERE wa.working_schedule_id = t.id
    ) ag ON true
         LEFT JOIN LATERAL (
    SELECT jsonb_agg(call_center.cc_get_lookup(s.id, s.name)) AS skills
    FROM wfm.working_schedule_extra_skill ws
             INNER JOIN call_center.cc_skill s on ws.skill_id = s.id
    WHERE ws.working_schedule_id = t.id
    ) sg ON true;
;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP VIEW wfm.working_schedule_v;

DROP TABLE wfm.working_schedule_agent;

DROP TABLE wfm.working_schedule_extra_skill;

DROP TRIGGER tg_populate_updated_at_column ON wfm.working_schedule;

DROP TABLE wfm.working_schedule;
-- +goose StatementEnd
