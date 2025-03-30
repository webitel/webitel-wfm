-- +goose Up
-- +goose StatementBegin
CREATE TABLE wfm.shift_template
(
    id          SERIAL PRIMARY KEY,
    domain_id   BIGINT                                                                  NOT NULL,
    created_at  TIMESTAMP WITH TIME ZONE DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC') NOT NULL,
    created_by  BIGINT                                                                  NOT NULL,
    updated_at  TIMESTAMP WITH TIME ZONE DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC') NOT NULL,
    updated_by  BIGINT                                                                  NOT NULL,

    name        TEXT                                                                    NOT NULL,
    description TEXT,
    times       JSONB                                                                   NOT NULL,

    UNIQUE (domain_id, id),
    FOREIGN KEY (domain_id) REFERENCES directory.wbt_domain (dc) ON DELETE CASCADE,
    FOREIGN KEY (domain_id, created_by) REFERENCES directory.wbt_user (dc, id) ON DELETE SET NULL (created_by),
    FOREIGN KEY (domain_id, updated_by) REFERENCES directory.wbt_user (dc, id) ON DELETE SET NULL (updated_by),

    CHECK ( char_length(name) <= 250 )

    /*
        FIXME: times as JSONB array
        CHECK ( (times ->> 'start')::int between 0 and 1440 ),
        CHECK ( (times ->> 'end')::int between 0 and 1440 ),
        CHECK ( (times ->> 'start')::int < (times ->> 'end')::int )
     */
);

CREATE TRIGGER tg_populate_updated_at_column
    BEFORE UPDATE
    ON wfm.shift_template
    FOR EACH ROW
EXECUTE PROCEDURE wfm.tg_populate_updated_at_column();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER tg_populate_updated_at_column ON wfm.shift_template;

DROP TABLE wfm.shift_template;
-- +goose StatementEnd
