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
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER tg_populate_updated_at_column ON wfm.working_condition;

DROP TABLE wfm.working_condition;
-- +goose StatementEnd
