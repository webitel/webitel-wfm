-- +goose Up
-- +goose StatementBegin
CREATE TABLE wfm.shift_template_time
(
    id                SERIAL PRIMARY KEY,
    domain_id         BIGINT                                                                  NOT NULL,
    created_at        TIMESTAMP WITH TIME ZONE DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC') NOT NULL,
    created_by        BIGINT                                                                  NOT NULL,
    updated_at        TIMESTAMP WITH TIME ZONE DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC') NOT NULL,
    updated_by        BIGINT                                                                  NOT NULL,

    shift_template_id BIGINT                                                                  NOT NULL,
    start_min         INT2                                                                    NOT NULL,
    end_min           INT2                                                                    NOT NULL,

    UNIQUE (domain_id, id, shift_template_id),
    FOREIGN KEY (domain_id) REFERENCES directory.wbt_domain (dc) ON DELETE CASCADE,
    FOREIGN KEY (domain_id, created_by) REFERENCES directory.wbt_user (dc, id) ON DELETE SET NULL,
    FOREIGN KEY (domain_id, updated_by) REFERENCES directory.wbt_user (dc, id) ON DELETE SET NULL,
    FOREIGN KEY (domain_id, shift_template_id) REFERENCES wfm.shift_template (domain_id, id) ON DELETE CASCADE,

    CHECK ( start_min between 0 and 1440 ),
    CHECK ( end_min between 0 and 1440 ),
    CHECK ( start_min < end_min )
);

CREATE TRIGGER tg_populate_updated_at_column
    BEFORE UPDATE
    ON wfm.shift_template_time
    FOR EACH ROW
EXECUTE PROCEDURE wfm.tg_populate_updated_at_column();

CREATE VIEW wfm.shift_template_time_v AS
SELECT d.id                                    AS id
     , d.domain_id                             AS domain_id
     , d.created_at                            AS created_at
     , call_center.cc_get_lookup(c.id, c.name) AS created_by
     , d.updated_at                            AS updated_at
     , call_center.cc_get_lookup(u.id, u.name) AS updated_by
     , d.start_min                             AS start_min
     , d.end_min                               AS end_min
FROM wfm.shift_template_time d
         LEFT JOIN directory.wbt_user c ON d.created_by = c.id
         LEFT JOIN directory.wbt_user u ON d.updated_by = u.id;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP VIEW wfm.shift_template_time_v;

DROP TRIGGER tg_populate_updated_at_column ON wfm.shift_template_time;

DROP TABLE wfm.shift_template_time;
-- +goose StatementEnd
