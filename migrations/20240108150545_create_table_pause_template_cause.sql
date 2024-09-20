-- +goose Up
-- +goose StatementBegin
CREATE TABLE wfm.pause_template_cause
(
    id                SERIAL PRIMARY KEY,
    domain_id         BIGINT NOT NULL,

    pause_template_id BIGINT NOT NULL,
    pause_cause_id    BIGINT,
    duration          BIGINT NOT NULL,

    UNIQUE (domain_id, id, pause_template_id),
    FOREIGN KEY (domain_id) REFERENCES directory.wbt_domain (dc) ON DELETE CASCADE,
    FOREIGN KEY (domain_id, pause_template_id) REFERENCES wfm.pause_template (domain_id, id) ON DELETE CASCADE,
    FOREIGN KEY (domain_id, pause_cause_id) REFERENCES call_center.cc_pause_cause (domain_id, id) ON DELETE CASCADE,

    CHECK ( duration <= 1440 )
);

CREATE TRIGGER tg_populate_updated_at_column
    BEFORE UPDATE
    ON wfm.pause_template_cause
    FOR EACH ROW
EXECUTE PROCEDURE wfm.tg_populate_updated_at_column();

CREATE UNIQUE INDEX IF NOT EXISTS cc_pause_cause_domain_id_udx on call_center.cc_pause_cause USING btree (id, domain_id);

CREATE VIEW wfm.pause_template_cause_v AS
SELECT d.id                                      AS id
     , d.domain_id                               AS domain_id
     , d.pause_template_id                       AS pause_template_id
     , d.duration                                AS duration
     , call_center.cc_get_lookup(pc.id, pc.name) AS cause
FROM wfm.pause_template_cause d
         LEFT JOIN call_center.cc_pause_cause pc ON d.pause_cause_id = pc.id;

CREATE VIEW wfm.pause_template_v AS
SELECT t.id                                                                       AS id
     , t.domain_id                                                                AS domain_id
     , t.created_at                                                               AS created_at
     , call_center.cc_get_lookup(c.id, c.name)                                    AS created_by
     , t.updated_at                                                               AS updated_at
     , call_center.cc_get_lookup(u.id, u.name)                                    AS updated_by
     , t.name                                                                     AS name
     , t.description                                                              AS description
     , jsonb_agg(json_build_object('duration', ptc.duration, 'cause', ptc.cause)) AS causes
FROM wfm.pause_template t
         LEFT JOIN directory.wbt_user c ON t.created_by = c.id
         LEFT JOIN directory.wbt_user u ON t.updated_by = u.id
         LEFT JOIN wfm.pause_template_cause_v ptc ON t.id = ptc.pause_template_id
GROUP BY t.id, t.domain_id, t.created_at, c.id, c.name, t.updated_at, u.id, u.name, t.name, t.description;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP VIEW wfm.pause_template_v;

DROP VIEW wfm.pause_template_cause_v;

DROP INDEX call_center.cc_pause_cause_domain_id_udx;

DROP TRIGGER tg_populate_updated_at_column ON wfm.pause_template_cause;

DROP TABLE wfm.pause_template_cause;
-- +goose StatementEnd
