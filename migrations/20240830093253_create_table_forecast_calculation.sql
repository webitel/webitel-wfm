-- +goose Up
-- +goose StatementBegin
CREATE TABLE wfm.forecast_calculation
(
    id          SERIAL PRIMARY KEY,
    domain_id   BIGINT                                                                  NOT NULL,
    created_at  TIMESTAMP WITH TIME ZONE DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC') NOT NULL,
    created_by  BIGINT                                                                  NOT NULL,
    updated_at  TIMESTAMP WITH TIME ZONE DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC') NOT NULL,
    updated_by  BIGINT                                                                  NOT NULL,

    name        TEXT                                                                    NOT NULL,
    description TEXT,
    query       TEXT                                                                    NOT NULL,

    UNIQUE (domain_id, id),
    FOREIGN KEY (domain_id) REFERENCES directory.wbt_domain (dc) ON DELETE CASCADE,
    FOREIGN KEY (domain_id, created_by) REFERENCES directory.wbt_user (dc, id) ON DELETE SET NULL,
    FOREIGN KEY (domain_id, updated_by) REFERENCES directory.wbt_user (dc, id) ON DELETE SET NULL,

    CHECK ( char_length(name) <= 250 )
);

CREATE TRIGGER tg_populate_updated_at_column
    BEFORE UPDATE
    ON wfm.forecast_calculation
    FOR EACH ROW
EXECUTE PROCEDURE wfm.tg_populate_updated_at_column();

CREATE TABLE wfm.forecast_calculation_acl
(
    id      SERIAL PRIMARY KEY,
    dc      BIGINT             NOT NULL,
    grantor BIGINT,
    object  INTEGER            NOT NULL,
    subject BIGINT             NOT NULL,
    access  SMALLINT DEFAULT 0 NOT NULL,

    UNIQUE (object, subject) INCLUDE (access),
    UNIQUE (subject, object) INCLUDE (access),
    FOREIGN KEY (dc) REFERENCES directory.wbt_domain ON DELETE CASCADE,
    FOREIGN KEY (grantor) REFERENCES directory.wbt_auth ON DELETE SET NULL,
    FOREIGN KEY (object) REFERENCES wfm.forecast_calculation ON UPDATE CASCADE ON DELETE CASCADE,
    FOREIGN KEY (grantor, dc) REFERENCES directory.wbt_auth (id, dc) ON UPDATE CASCADE ON DELETE CASCADE,
    FOREIGN KEY (object, dc) REFERENCES wfm.forecast_calculation (id, domain_id) ON DELETE CASCADE,
    FOREIGN KEY (subject, dc) REFERENCES directory.wbt_auth (id, dc) ON DELETE CASCADE
);

CREATE INDEX forecast_calculation_acl_grantor_idx ON wfm.forecast_calculation_acl (grantor);

CREATE TRIGGER tg_forecast_calculation_set_rbac_acl
    AFTER INSERT
    ON wfm.forecast_calculation
    FOR EACH ROW
EXECUTE PROCEDURE wfm.tg_obj_default_rbac('forecast_calculation_acl');

CREATE VIEW wfm.forecast_calculation_v AS
SELECT t.id                                    AS id
     , t.domain_id                             AS domain_id
     , t.created_at                            AS created_at
     , call_center.cc_get_lookup(c.id, c.name) AS created_by
     , t.updated_at                            AS updated_at
     , call_center.cc_get_lookup(u.id, u.name) AS updated_by
     , t.name                                  AS name
     , t.description                           AS description
     , t.query                                 AS query
FROM wfm.forecast_calculation t
         LEFT JOIN directory.wbt_user c ON t.created_by = c.id
         LEFT JOIN directory.wbt_user u ON t.updated_by = u.id;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP VIEW wfm.forecast_calculation_v;

DROP TRIGGER tg_forecast_calculation_set_rbac_acl ON wfm.forecast_calculation;

DROP INDEX wfm.forecast_calculation_acl_grantor_idx;

DROP TABLE wfm.forecast_calculation_acl;

DROP TRIGGER tg_populate_updated_at_column ON wfm.forecast_calculation;

DROP TABLE wfm.forecast_calculation;
-- +goose StatementEnd
