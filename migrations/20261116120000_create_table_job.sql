-- +goose Up
-- +goose StatementBegin
-- Background jobs (ADR-004), run by the leader. Finished rows stay: who planned which schedule and when (EP_04).
CREATE TABLE wfm.job
(
    id                  BIGSERIAL PRIMARY KEY,
    kind                TEXT                     NOT NULL,
    state               TEXT                     NOT NULL DEFAULT 'queued',
    domain_id           BIGINT,
    working_schedule_id BIGINT,
    created_by          BIGINT,
    created_ip          TEXT,
    created_at          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    scheduled_for       TIMESTAMP WITH TIME ZONE,
    locked_by           TEXT,
    locked_until        TIMESTAMP WITH TIME ZONE,
    attempts            INT                      NOT NULL DEFAULT 0,
    started_at          TIMESTAMP WITH TIME ZONE,
    finished_at         TIMESTAMP WITH TIME ZONE,
    error               TEXT,
    error_code          TEXT,
    result              JSONB,

    FOREIGN KEY (domain_id) REFERENCES directory.wbt_domain (dc) ON DELETE CASCADE,
    FOREIGN KEY (domain_id, working_schedule_id) REFERENCES wfm.working_schedule (domain_id, id) ON DELETE CASCADE,
    FOREIGN KEY (domain_id, created_by) REFERENCES directory.wbt_user (dc, id) ON DELETE SET NULL (created_by),

    CONSTRAINT job_kind_check CHECK ( kind IN ('planning', 'daily_aggregate') ),
    CONSTRAINT job_state_check CHECK ( state IN ('queued', 'running', 'done', 'failed') ),
    -- A NULL domain_id would leave the schedule's foreign key unchecked.
    CONSTRAINT job_planning_schedule_check CHECK ( kind <> 'planning' OR (domain_id IS NOT NULL AND working_schedule_id IS NOT NULL) ),
    CONSTRAINT job_daily_aggregate_scheduled_check CHECK ( kind <> 'daily_aggregate' OR scheduled_for IS NOT NULL ),
    CONSTRAINT job_lease_check CHECK ( (state = 'running') = (locked_by IS NOT NULL) AND (locked_by IS NULL) = (locked_until IS NULL) ),
    CONSTRAINT job_failed_error_check CHECK ( state <> 'failed' OR error IS NOT NULL )
);

CREATE UNIQUE INDEX job_schedule_active_udx ON wfm.job (kind, working_schedule_id) WHERE state IN ('queued', 'running');

CREATE UNIQUE INDEX job_scheduled_for_udx ON wfm.job (kind, scheduled_for) WHERE scheduled_for IS NOT NULL;

CREATE INDEX job_active_idx ON wfm.job (state, id) WHERE state IN ('queued', 'running');

CREATE INDEX job_schedule_idx ON wfm.job (working_schedule_id, id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE wfm.job;
-- +goose StatementEnd
