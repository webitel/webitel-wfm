-- +goose Up
-- +goose StatementBegin
DROP VIEW wfm.agent_working_schedule_v;

DROP VIEW wfm.agent_working_schedule_pause_v;

-- An agent is in a schedule once: a duplicate made the shift insert's agent lookup return two rows.
CREATE UNIQUE INDEX working_schedule_agent_schedule_agent_udx
    ON wfm.working_schedule_agent (working_schedule_id, agent_id);

-- No shifts are carried over, as none exist yet: on a table that has some, this migration fails and changes nothing.
ALTER TABLE wfm.agent_working_schedule
    DROP CONSTRAINT agent_working_schedule_working_schedule_agent_id_schedule_a_key,
    DROP COLUMN start_min,
    DROP COLUMN end_min,
    ALTER COLUMN schedule_at DROP DEFAULT,
    ADD COLUMN start_at       TIMESTAMP WITH TIME ZONE NOT NULL,
    ADD COLUMN end_at         TIMESTAMP WITH TIME ZONE NOT NULL,
    ADD COLUMN ver            INT                      NOT NULL DEFAULT 1,
    ADD COLUMN source         INT2                     NOT NULL DEFAULT 1,
    ADD COLUMN planner_run_id BIGINT,
    -- The shift is [start_at, end_at): the next one may start at end_at.
    ADD CONSTRAINT agent_working_schedule_duration_check
        CHECK ( start_at < end_at AND end_at - start_at <= INTERVAL '24 hours' ),
    -- 1 manual, 2 planner, as in wfm.ShiftSource.
    ADD CONSTRAINT agent_working_schedule_source_check CHECK ( source IN (1, 2) );

CREATE INDEX agent_working_schedule_agent_date_idx
    ON wfm.agent_working_schedule (working_schedule_agent_id, schedule_at);

-- The working date is the day the shift starts on in the time zone of its schedule's calendar.
CREATE FUNCTION wfm.tg_agent_working_schedule_working_date()
    RETURNS TRIGGER AS
$$
BEGIN
    SELECT (NEW.start_at AT TIME ZONE tz.sys_name)::DATE
    INTO NEW.schedule_at
    FROM wfm.working_schedule_agent wsa
             INNER JOIN wfm.working_schedule ws ON ws.id = wsa.working_schedule_id
             INNER JOIN flow.calendar c ON c.id = ws.calendar_id
             INNER JOIN flow.calendar_timezones tz ON tz.id = c.timezone_id
    WHERE wsa.id = NEW.working_schedule_agent_id;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER tg_working_date
    BEFORE INSERT OR UPDATE OF start_at, working_schedule_agent_id
    ON wfm.agent_working_schedule
    FOR EACH ROW
EXECUTE PROCEDURE wfm.tg_agent_working_schedule_working_date();

ALTER TABLE wfm.agent_working_schedule_pause
    DROP COLUMN start_min,
    DROP COLUMN end_min,
    ADD COLUMN start_at TIMESTAMP WITH TIME ZONE NOT NULL,
    ADD COLUMN end_at   TIMESTAMP WITH TIME ZONE NOT NULL,
    ADD CONSTRAINT agent_working_schedule_pause_duration_check CHECK ( start_at < end_at );

-- A pause lies inside its shift. The argument names the column that holds the shift id.
CREATE FUNCTION wfm.tg_agent_working_schedule_pause_inside()
    RETURNS TRIGGER AS
$$
BEGIN
    IF EXISTS (SELECT 1
               FROM wfm.agent_working_schedule s
                        INNER JOIN wfm.agent_working_schedule_pause p ON p.agent_working_schedule_id = s.id
               WHERE s.id = (to_jsonb(NEW) ->> TG_ARGV[0])::BIGINT
                 AND (p.start_at < s.start_at OR p.end_at > s.end_at)) THEN
        RAISE check_violation USING
            MESSAGE = 'a pause lies outside its shift',
            CONSTRAINT = 'agent_working_schedule_pause_inside';
    END IF;

    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- Checked at commit, so a shift and its pauses can be moved in one transaction in any order.
CREATE CONSTRAINT TRIGGER agent_working_schedule_pause_inside
    AFTER INSERT OR UPDATE OF start_at, end_at, agent_working_schedule_id
    ON wfm.agent_working_schedule_pause
    DEFERRABLE INITIALLY DEFERRED
    FOR EACH ROW
EXECUTE PROCEDURE wfm.tg_agent_working_schedule_pause_inside('agent_working_schedule_id');

CREATE CONSTRAINT TRIGGER agent_working_schedule_pause_inside
    AFTER UPDATE OF start_at, end_at
    ON wfm.agent_working_schedule
    DEFERRABLE INITIALLY DEFERRED
    FOR EACH ROW
EXECUTE PROCEDURE wfm.tg_agent_working_schedule_pause_inside('id');

-- A shift switches on or off a skill the agent already has, with the agent's own capacity (WFM-114).
ALTER TABLE wfm.agent_working_schedule_skill
    ALTER COLUMN capacity DROP NOT NULL,
    ADD COLUMN enabled BOOLEAN NOT NULL DEFAULT true;

CREATE UNIQUE INDEX agent_working_schedule_skill_shift_skill_udx
    ON wfm.agent_working_schedule_skill (agent_working_schedule_id, skill_id);

-- The views change only where the columns did; the calendar cell format is WFM-95.
CREATE VIEW wfm.agent_working_schedule_pause_v AS
(
SELECT t.id                                      AS id
     , t.domain_id                               AS domain_id
     , t.created_at                              AS created_at
     , call_center.cc_get_lookup(c.id, c.name)   AS created_by
     , t.updated_at                              AS updated_at
     , call_center.cc_get_lookup(u.id, u.name)   AS updated_by
     , t.agent_working_schedule_id               AS agent_working_schedule_id
     , t.start_at                                AS start_at
     , t.end_at                                  AS end_at
     , call_center.cc_get_lookup(pc.id, pc.name) AS cause
FROM wfm.agent_working_schedule_pause t
         LEFT JOIN call_center.cc_pause_cause pc ON t.pause_cause_id = pc.id
         LEFT JOIN directory.wbt_user c ON t.created_by = c.id
         LEFT JOIN directory.wbt_user u ON t.updated_by = u.id
    );

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

-- +goose Down
-- +goose StatementBegin
DROP VIEW wfm.agent_working_schedule_v;

DROP VIEW wfm.agent_working_schedule_pause_v;

DROP INDEX wfm.agent_working_schedule_skill_shift_skill_udx;

ALTER TABLE wfm.agent_working_schedule_skill
    DROP COLUMN enabled,
    ALTER COLUMN capacity SET NOT NULL;

DROP TRIGGER agent_working_schedule_pause_inside ON wfm.agent_working_schedule;

DROP TRIGGER agent_working_schedule_pause_inside ON wfm.agent_working_schedule_pause;

DROP FUNCTION wfm.tg_agent_working_schedule_pause_inside;

ALTER TABLE wfm.agent_working_schedule_pause
    DROP COLUMN start_at,
    DROP COLUMN end_at,
    ADD COLUMN start_min INT2 NOT NULL,
    ADD COLUMN end_min   INT2 NOT NULL;

DROP TRIGGER tg_working_date ON wfm.agent_working_schedule;

DROP FUNCTION wfm.tg_agent_working_schedule_working_date;

DROP INDEX wfm.agent_working_schedule_agent_date_idx;

ALTER TABLE wfm.agent_working_schedule
    DROP COLUMN start_at,
    DROP COLUMN end_at,
    DROP COLUMN ver,
    DROP COLUMN source,
    DROP COLUMN planner_run_id,
    ALTER COLUMN schedule_at SET DEFAULT (CURRENT_DATE AT TIME ZONE 'UTC'),
    ADD COLUMN start_min INT2 NOT NULL,
    ADD COLUMN end_min   INT2 NOT NULL,
    ADD UNIQUE (working_schedule_agent_id, schedule_at);

DROP INDEX wfm.working_schedule_agent_schedule_agent_udx;

CREATE VIEW wfm.agent_working_schedule_pause_v AS
(
SELECT t.id                                      AS id
     , t.domain_id                               AS domain_id
     , t.created_at                              AS created_at
     , call_center.cc_get_lookup(c.id, c.name)   AS created_by
     , t.updated_at                              AS updated_at
     , call_center.cc_get_lookup(u.id, u.name)   AS updated_by
     , t.agent_working_schedule_id               AS agent_working_schedule_id
     , t.start_min                               AS start_min
     , t.end_min                                 AS end_min
     , call_center.cc_get_lookup(pc.id, pc.name) AS cause
FROM wfm.agent_working_schedule_pause t
         LEFT JOIN call_center.cc_pause_cause pc ON t.pause_cause_id = pc.id
         LEFT JOIN directory.wbt_user c ON t.created_by = c.id
         LEFT JOIN directory.wbt_user u ON t.updated_by = u.id
    );

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
        , 'start', aws.start_min
        , 'end', aws.end_min
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
            , 'start', start_min
            , 'end', end_min
            , 'cause', cause)) pauses
        FROM wfm.agent_working_schedule_pause_v
        WHERE agent_working_schedule_id = aws.id
        ) p ON true
    WHERE aws.working_schedule_agent_id = wsa.id
    ) x ON true
    );
-- +goose StatementEnd
