-- +goose Up
-- +goose StatementBegin
CREATE TABLE wfm.working_condition
(
    id                 SERIAL PRIMARY KEY,
    domain_id          BIGINT                                                                  NOT NULL,
    created_at         TIMESTAMP WITH TIME ZONE DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC') NOT NULL,
    created_by         BIGINT                                                                  NOT NULL,
    updated_at         TIMESTAMP WITH TIME ZONE DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC') NOT NULL,
    updated_by         BIGINT                                                                  NOT NULL,

    name               TEXT                                                                    NOT NULL,
    description        TEXT,
    workday_hours      INT2,
    workdays_per_month INT2,
    vacation           INT2,
    sick_leaves        INT2,
    days_off           INT2,
    pause_duration     INT2,

    pause_template_id  BIGINT                                                                  NOT NULL,
    shift_template_id  BIGINT,

    UNIQUE (domain_id, id),
    FOREIGN KEY (domain_id) REFERENCES directory.wbt_domain (dc) ON DELETE CASCADE,
    FOREIGN KEY (domain_id, created_by) REFERENCES directory.wbt_user (dc, id) ON DELETE SET NULL (created_by),
    FOREIGN KEY (domain_id, updated_by) REFERENCES directory.wbt_user (dc, id) ON DELETE SET NULL (updated_by),
    FOREIGN KEY (domain_id, pause_template_id) REFERENCES wfm.pause_template (domain_id, id) ON DELETE SET NULL (pause_template_id),
    FOREIGN KEY (domain_id, shift_template_id) REFERENCES wfm.shift_template (domain_id, id) ON DELETE SET NULL (shift_template_id),

    CHECK ( char_length(name) <= 250 ),
    CHECK ( workday_hours isnull or workday_hours between 0 and 24 ),
    CHECK ( workdays_per_month isnull or workdays_per_month between 0 and 31 ),
    CHECK ( vacation isnull or vacation between 0 and 365 ),
    CHECK ( sick_leaves isnull or sick_leaves between 0 and 365 ),
    CHECK ( days_off isnull or days_off between 0 and 365 ),
    CHECK ( pause_duration isnull or pause_duration between 0 and 1440 ),
    CHECK ( (workday_hours >= (pause_duration / 60)) ),
    CHECK ( coalesce(vacation, 0) + coalesce(sick_leaves, 0) + coalesce(days_off, 0) < 365 )
);

CREATE TRIGGER tg_populate_updated_at_column
    BEFORE UPDATE
    ON wfm.working_condition
    FOR EACH ROW
EXECUTE PROCEDURE wfm.tg_populate_updated_at_column();

CREATE VIEW wfm.working_condition_v AS
SELECT t.id                                        AS id
     , t.domain_id                                 AS domain_id
     , t.created_at                                AS created_at
     , call_center.cc_get_lookup(c.id, c.name)     AS created_by
     , t.updated_at                                AS updated_at
     , call_center.cc_get_lookup(u.id, u.name)     AS updated_by
     , t.name                                      AS name
     , t.description                               AS description
     , t.workday_hours                             AS workday_hours
     , t.workdays_per_month                        AS workdays_per_month
     , t.vacation                                  AS vacation
     , t.sick_leaves                               AS sick_leaves
     , t.days_off                                  AS days_off
     , t.pause_duration                            AS pause_duration
     , call_center.cc_get_lookup(st.id, st.name)   AS shift_template
     , call_center.cc_get_lookup(svc.id, svc.name) AS pause_template
FROM wfm.working_condition t
         LEFT JOIN directory.wbt_user c ON t.created_by = c.id
         LEFT JOIN directory.wbt_user u ON t.updated_by = u.id
         LEFT JOIN wfm.shift_template st ON t.shift_template_id = st.id
         LEFT JOIN wfm.pause_template svc ON t.pause_template_id = svc.id;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP VIEW wfm.working_condition_v;

DROP TRIGGER tg_populate_updated_at_column ON wfm.working_condition;

DROP TABLE wfm.working_condition;
-- +goose StatementEnd
