-- +goose Up
-- +goose StatementBegin
CREATE OR REPLACE VIEW wfm.working_schedule_v AS
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
     , t.ver                                     AS ver
     , ac.total_agents                           AS total_agents
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
    ) sg ON true
         LEFT JOIN LATERAL (
    SELECT count(*) AS total_agents
    FROM wfm.working_schedule_agent wa
             INNER JOIN call_center.cc_agent a on wa.agent_id = a.id
             INNER JOIN directory.wbt_user au ON a.user_id = au.id
    WHERE wa.working_schedule_id = t.id
    ) ac ON true;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP VIEW wfm.working_schedule_v;

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
     , t.ver                                     AS ver
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
-- +goose StatementEnd
