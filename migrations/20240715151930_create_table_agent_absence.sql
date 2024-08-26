-- +goose Up
-- +goose StatementBegin
CREATE TABLE wfm.agent_absence
(
    id              SERIAL PRIMARY KEY,
    domain_id       BIGINT                                                                  NOT NULL,
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC') NOT NULL,
    created_by      BIGINT                                                                  NOT NULL,
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC') NOT NULL,
    updated_by      BIGINT                                                                  NOT NULL,

    absent_at       DATE                     DEFAULT (CURRENT_DATE AT TIME ZONE 'UTC')      NOT NULL,
    agent_id        BIGINT                                                                  NOT NULL,
    absence_type_id BIGINT                                                                  NOT NULL,

    UNIQUE (domain_id, id),
    UNIQUE (absent_at, agent_id, absence_type_id),
    FOREIGN KEY (domain_id) REFERENCES directory.wbt_domain (dc) ON DELETE CASCADE,
    FOREIGN KEY (domain_id, created_by) REFERENCES directory.wbt_user (dc, id) ON DELETE SET NULL,
    FOREIGN KEY (domain_id, updated_by) REFERENCES directory.wbt_user (dc, id) ON DELETE SET NULL,
    FOREIGN KEY (domain_id, agent_id) REFERENCES call_center.cc_agent (domain_id, id) ON DELETE CASCADE
);

CREATE TRIGGER tg_populate_updated_at_column
    BEFORE UPDATE
    ON wfm.agent_absence
    FOR EACH ROW
EXECUTE PROCEDURE wfm.tg_populate_updated_at_column();

CREATE TABLE wfm.agent_absence_acl
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
    FOREIGN KEY (object) REFERENCES wfm.agent_absence ON UPDATE CASCADE ON DELETE CASCADE,
    FOREIGN KEY (grantor, dc) REFERENCES directory.wbt_auth (id, dc) ON UPDATE CASCADE ON DELETE CASCADE,
    FOREIGN KEY (object, dc) REFERENCES wfm.agent_absence (id, domain_id) ON DELETE CASCADE,
    FOREIGN KEY (subject, dc) REFERENCES directory.wbt_auth (id, dc) ON DELETE CASCADE
);

CREATE INDEX agent_absence_acl_grantor_idx ON wfm.agent_absence_acl (grantor);

CREATE TRIGGER tg_agent_absence_set_rbac_acl
    AFTER INSERT
    ON wfm.agent_absence
    FOR EACH ROW
EXECUTE PROCEDURE wfm.tg_obj_default_rbac('agent_absence_acl');

CREATE VIEW wfm.agent_absence_v AS
SELECT t.id                                                            AS id
     , t.domain_id                                                     AS domain_id
     , t.created_at                                                    AS created_at
     , call_center.cc_get_lookup(c.id, coalesce(c.name, c.username))   AS created_by
     , t.updated_at                                                    AS updated_at
     , call_center.cc_get_lookup(u.id, u.name)                         AS updated_by
     , t.absent_at                                                     AS absent_at
     , call_center.cc_get_lookup(a.id, coalesce(au.name, au.username)) AS agent
     , t.absence_type_id                                               AS absence_type_id
FROM wfm.agent_absence t
         LEFT JOIN directory.wbt_user c ON t.created_by = c.id
         LEFT JOIN directory.wbt_user u ON t.updated_by = u.id
         LEFT JOIN call_center.cc_agent a ON t.agent_id = a.id
         LEFT JOIN directory.wbt_user au ON a.user_id = au.id;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP VIEW wfm.agent_absence_v;

DROP TRIGGER tg_agent_absence_set_rbac_acl ON wfm.agent_absence;

DROP INDEX wfm.agent_absence_acl_grantor_idx;

DROP TABLE wfm.agent_absence_acl;

DROP TRIGGER tg_populate_updated_at_column ON wfm.agent_absence;

DROP TABLE wfm.agent_absence;
-- +goose StatementEnd
