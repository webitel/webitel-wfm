-- +goose Up
-- +goose StatementBegin
DROP VIEW wfm.agent_working_schedule_v;

-- A calendar cell: an agent's working date with its shifts, absence and lock. An agent
-- with nothing in the schedule comes back once, with no date.
CREATE VIEW wfm.agent_working_schedule_v AS
(
SELECT ws.id                                                           AS working_schedule_id
     , ws.domain_id                                                    AS domain_id
     , call_center.cc_get_lookup(a.id, coalesce(wu.name, wu.username)) AS agent
     , d.date                                                          AS date
     , d.locked                                                        AS locked
     , d.absence                                                       AS absence
     , d.shifts                                                        AS shifts
FROM wfm.working_schedule ws
         INNER JOIN wfm.working_schedule_agent wsa ON wsa.working_schedule_id = ws.id
         INNER JOIN call_center.cc_agent a ON a.id = wsa.agent_id
         INNER JOIN directory.wbt_user wu ON wu.id = a.user_id
         LEFT JOIN LATERAL (
    SELECT x.date                                                                      AS date
         , coalesce(bool_or(x.locked), false)                                          AS locked
         -- One absence a day (Core design 1.4): the last one written wins until WFM-96 keys them by date.
         , (array_agg(x.absence ORDER BY x.written_at DESC, x.absence DESC)
            FILTER ( WHERE x.absence IS NOT NULL ))[1]                                 AS absence
         , jsonb_agg(x.shift ORDER BY x.start_at) FILTER ( WHERE x.shift IS NOT NULL ) AS shifts
    FROM (SELECT aa.absent_at         AS date
               , NULL::boolean        AS locked
               , aa.absence_type_id   AS absence
               , NULL::jsonb          AS shift
               , NULL::timestamptz    AS start_at
               , aa.updated_at        AS written_at
          FROM wfm.agent_absence aa
          WHERE aa.domain_id = ws.domain_id
            AND aa.agent_id = wsa.agent_id
            AND aa.absent_at BETWEEN ws.start_date_at AND ws.end_date_at

          UNION ALL

          -- A shift on the date in another active schedule (state 3) locks it, as in WFM-94.
          SELECT aws.schedule_at, true, NULL, NULL, NULL, NULL
          FROM wfm.agent_working_schedule aws
                   INNER JOIN wfm.working_schedule_agent wsa2 ON wsa2.id = aws.working_schedule_agent_id
                   INNER JOIN wfm.working_schedule ws2 ON ws2.id = wsa2.working_schedule_id
          WHERE wsa2.agent_id = wsa.agent_id
            AND ws2.domain_id = ws.domain_id
            AND ws2.id <> ws.id
            AND ws2.state = 3
            AND aws.schedule_at BETWEEN ws.start_date_at AND ws.end_date_at

          UNION ALL

          SELECT aws.schedule_at
               , NULL
               , NULL
               , jsonb_build_object('id', aws.id
                  , 'domain_id', aws.domain_id
                  , 'created_at', aws.created_at
                  , 'created_by', call_center.cc_get_lookup(c.id, c.name)
                  , 'updated_at', aws.updated_at
                  , 'updated_by', call_center.cc_get_lookup(u.id, u.name)
                  , 'start_at', aws.start_at
                  , 'end_at', aws.end_at
                  , 'working_date', aws.schedule_at
                  , 'ver', aws.ver
                  , 'source', aws.source
                  , 'planner_run_id', aws.planner_run_id
                  , 'pauses', p.pauses
                  , 'skills', k.skills)
               , aws.start_at
               , NULL
          FROM wfm.agent_working_schedule aws
                   LEFT JOIN directory.wbt_user c ON c.id = aws.created_by
                   LEFT JOIN directory.wbt_user u ON u.id = aws.updated_by
                   LEFT JOIN LATERAL (
              SELECT jsonb_agg(jsonb_build_object('id', pv.id
                                   , 'domain_id', pv.domain_id
                                   , 'created_at', pv.created_at
                                   , 'created_by', pv.created_by
                                   , 'updated_at', pv.updated_at
                                   , 'updated_by', pv.updated_by
                                   , 'start_at', pv.start_at
                                   , 'end_at', pv.end_at
                                   , 'cause', pv.cause) ORDER BY pv.start_at) AS pauses
              FROM wfm.agent_working_schedule_pause_v pv
              WHERE pv.agent_working_schedule_id = aws.id
              ) p ON true
                   LEFT JOIN LATERAL (
              SELECT jsonb_agg(jsonb_build_object('skill', call_center.cc_get_lookup(s.id, s.name)
                                   , 'capacity', sk.capacity
                                   , 'enabled', sk.enabled) ORDER BY s.id) AS skills
              FROM wfm.agent_working_schedule_skill sk
                       INNER JOIN call_center.cc_skill s ON s.id = sk.skill_id
              WHERE sk.agent_working_schedule_id = aws.id
              ) k ON true
          WHERE aws.working_schedule_agent_id = wsa.id) x
    GROUP BY x.date
    ) d ON true
    );
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP VIEW wfm.agent_working_schedule_v;

CREATE VIEW wfm.agent_working_schedule_v AS
(
SELECT ws.id                                                           AS working_schedule_id
     , ws.domain_id                                                    AS domain_id
     , call_center.cc_get_lookup(a.id, coalesce(wu.name, wu.username)) AS agent
     , x.date                                                          AS date
     , x.locked                                                        AS locked
     , x.absence                                                       AS absence
     , x.shift                                                         AS shift
FROM wfm.working_schedule ws
         INNER JOIN wfm.working_schedule_agent wsa ON wsa.working_schedule_id = ws.id
         INNER JOIN call_center.cc_agent a ON a.id = wsa.agent_id
         INNER JOIN directory.wbt_user wu ON wu.id = a.user_id
         LEFT JOIN LATERAL (
    SELECT null               AS locked
         , aa.absent_at       AS date
         , aa.agent_id        AS agent_id
         , aa.absence_type_id AS absence
         , null::jsonb           shift
    FROM wfm.agent_absence aa
    WHERE aa.agent_id = wsa.agent_id
      AND aa.absent_at BETWEEN ws.start_date_at AND ws.end_date_at

    UNION ALL

    select true
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

    select null
         , aws.schedule_at
         , wsa.agent_id
         , null
         , jsonb_build_object('id', aws.id
        , 'domain_id', aws.domain_id
        , 'created_at', aws.created_at
        , 'created_by', call_center.cc_get_lookup(c.id, c.name)
        , 'updated_at', aws.updated_at
        , 'updated_by', call_center.cc_get_lookup(u.id, u.name)
        , 'start_at', aws.start_at
        , 'end_at', aws.end_at
        , 'pauses', p.pauses)
    FROM wfm.agent_working_schedule aws
             INNER JOIN directory.wbt_user c ON aws.created_by = c.id
             LEFT JOIN directory.wbt_user u ON aws.updated_by = u.id
             LEFT JOIN LATERAL (
        SELECT jsonb_agg(jsonb_build_object('id', id
            , 'domain_id', domain_id
            , 'created_at', created_at
            , 'created_by', created_by
            , 'updated_at', updated_at
            , 'updated_by', updated_by
            , 'start_at', start_at
            , 'end_at', end_at
            , 'cause', cause)) pauses
        FROM wfm.agent_working_schedule_pause_v
        WHERE agent_working_schedule_id = aws.id
        ) p ON true
    WHERE aws.working_schedule_agent_id = wsa.id
    ) x ON true
    );
-- +goose StatementEnd
