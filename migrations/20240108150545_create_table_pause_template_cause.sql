-- +goose Up
-- +goose StatementBegin
CREATE UNIQUE INDEX cc_pause_cause_domain_id_udx on call_center.cc_pause_cause USING btree (id, domain_id);

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
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE wfm.pause_template_cause;

DROP INDEX call_center.cc_pause_cause_domain_id_udx;
-- +goose StatementEnd
