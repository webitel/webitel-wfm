-- +goose Up
-- +goose StatementBegin
CREATE TABLE wfm.agent_working_schedule
(
    id                        SERIAL PRIMARY KEY,
    domain_id                 BIGINT                                                                  NOT NULL,
    created_at                TIMESTAMP WITH TIME ZONE DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC') NOT NULL,
    created_by                BIGINT,
    updated_at                TIMESTAMP WITH TIME ZONE DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC') NOT NULL,
    updated_by                BIGINT,

    working_schedule_agent_id BIGINT                                                                  NOT NULL,
    schedule_at               DATE                     DEFAULT (CURRENT_DATE AT TIME ZONE 'UTC')      NOT NULL,
    schedule_type             INT2                                                                    NOT NULL,
    pause_cause_id            BIGINT,
    start_min                 INT2,
    end_min                   INT2,

    UNIQUE (domain_id, id),
    FOREIGN KEY (domain_id) REFERENCES directory.wbt_domain (dc) ON DELETE CASCADE,
    FOREIGN KEY (domain_id, created_by) REFERENCES directory.wbt_user (dc, id) ON DELETE SET NULL (created_by),
    FOREIGN KEY (domain_id, updated_by) REFERENCES directory.wbt_user (dc, id) ON DELETE SET NULL (updated_by),
    FOREIGN KEY (domain_id, working_schedule_agent_id) REFERENCES wfm.working_schedule_agent (domain_id, id) ON DELETE CASCADE,
    FOREIGN KEY (domain_id, pause_cause_id) REFERENCES call_center.cc_pause_cause (domain_id, id) ON DELETE SET NULL (pause_cause_id)
);

CREATE VIEW wfm.agent_working_schedule_holidays_v AS
(
SELECT ws.id          working_schedule_id
     , ws.domain_id
     , i::date     AS date
     , ca.excepted AS name
FROM wfm.working_schedule ws
         LEFT JOIN generate_series(ws.start_date_at, ws.end_date_at, '1d'::interval) i ON TRUE
         INNER JOIN LATERAL (SELECT (SELECT x.name
                                     FROM unnest(c.excepts) AS x
                                     WHERE NOT x.disabled IS TRUE
                                       AND CASE
                                               WHEN x.repeat IS TRUE THEN
                                                   to_char(i::date, 'MM-DD') =
                                                   to_char((to_timestamp(x.date / 1000) AT TIME ZONE ct.sys_name)::date,
                                                           'MM-DD')
                                               ELSE
                                                   i::date = (to_timestamp(x.date / 1000) AT TIME ZONE ct.sys_name)::date
                                         END
                                     LIMIT 1) excepted
                             FROM flow.calendar c
                                      INNER JOIN flow.calendar_timezones ct ON c.timezone_id = ct.id
                             WHERE c.id = ws.calendar_id) ca ON ca.excepted NOTNULL
    );

CREATE VIEW wfm.agent_working_schedule_v AS
(
SELECT ws.id                                                           AS working_schedule_id
     , ws.domain_id                                                    AS domain_id
     , call_center.cc_get_lookup(a.id, coalesce(wu.name, wu.username)) AS agent
     , x.date                                                          AS date
     , x.type                                                          AS type
     , x.absence                                                       AS absence
     , x.shifts                                                        AS shifts
FROM wfm.working_schedule ws
         INNER JOIN wfm.working_schedule_agent wsa ON wsa.working_schedule_id = ws.id
         INNER JOIN call_center.cc_agent a ON a.id = wsa.agent_id
         INNER JOIN directory.wbt_user wu ON wu.id = a.user_id
         LEFT JOIN LATERAL (
    SELECT 1                  AS type
         , aa.absent_at       AS date
         , aa.agent_id        AS agent_id
         , aa.absence_type_id AS absence
         , null::jsonb           shifts
    FROM wfm.agent_absence aa
    WHERE aa.agent_id = wsa.agent_id
      AND aa.absent_at BETWEEN ws.start_date_at AND ws.end_date_at

    UNION ALL

    select 2
         , aws.schedule_at
         , wsa2.agent_id
         , null
         , null::jsonb
    FROM wfm.agent_working_schedule aws
             INNER JOIN wfm.working_schedule_agent wsa2 ON wsa2.id = aws.working_schedule_agent_id
             INNER JOIN wfm.working_schedule ws2 ON ws2.id = wsa2.working_schedule_id
    WHERE wsa2.agent_id = wsa.agent_id
      AND ws2.id != ws.id
      AND aws.schedule_at BETWEEN ws.start_date_at AND ws.end_date_at

    UNION ALL

    select 3
         , aws.schedule_at
         , wsa.agent_id
         , null
         , jsonb_build_array(jsonb_build_object('id', aws.id
        , 'domain_id', aws.domain_id
        , 'created_at', aws.created_at
        , 'created_by', call_center.cc_get_lookup(c.id, c.name)
        , 'updated_at', aws.updated_at
        , 'updated_by', call_center.cc_get_lookup(u.id, u.name)
        , 'type', aws.schedule_type
        , 'start', aws.start_min
        , 'end', aws.end_min))
    FROM wfm.agent_working_schedule aws
             INNER JOIN directory.wbt_user c ON aws.created_by = c.id
             LEFT JOIN directory.wbt_user u ON aws.updated_by = u.id
    WHERE aws.working_schedule_agent_id = wsa.id
    ) x ON true
    );
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP VIEW wfm.agent_working_schedule_v;

DROP VIEW wfm.agent_working_schedule_holidays_v;

DROP TABLE wfm.agent_working_schedule;
-- +goose StatementEnd
