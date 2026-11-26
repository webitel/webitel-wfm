package storage

import (
	"context"
	"encoding/json"
	"time"

	"github.com/webitel/webitel-go-kit/pkg/errors"

	"github.com/webitel/webitel-wfm/infra/storage/dbsql"
	"github.com/webitel/webitel-wfm/internal/model"
)

var (
	ErrJobActive    = errors.FailedPrecondition("the schedule already has a job of this kind queued or running", errors.WithID("storage.job.active"))
	ErrJobLeaseLost = errors.Aborted("the job is not running under this owner", errors.WithID("storage.job.lease_lost"))
)

// ADR-004: a job interrupted this many times fails instead of going back to the queue.
const jobAttempts = 2

const jobColumns = `id, kind, state, domain_id, working_schedule_id, created_by, created_ip, created_at, scheduled_for,
    locked_by, locked_until, attempts, started_at, finished_at, error, error_code, result`

// Either unique index leaves no row: an active job of the schedule, or a periodic run already queued.
const enqueueJobSQL = `
INSERT INTO wfm.job (kind, domain_id, working_schedule_id, created_by, created_ip, scheduled_for)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT DO NOTHING
RETURNING ` + jobColumns

const claimJobSQL = `
UPDATE wfm.job
SET state        = 'running',
    locked_by    = $1,
    locked_until = now() + $2 * INTERVAL '1 millisecond',
    attempts     = attempts + 1,
    started_at   = now()
WHERE id = (SELECT id
            FROM wfm.job
            WHERE state = 'queued'
              AND kind = ANY ($3)
            ORDER BY id
            LIMIT 1 FOR UPDATE SKIP LOCKED)
RETURNING ` + jobColumns

const releaseJobsSQL = `
UPDATE wfm.job
SET state        = CASE WHEN attempts < $1 THEN 'queued' ELSE 'failed' END,
    error        = CASE WHEN attempts < $1 THEN NULL ELSE 'interrupted ' || attempts || ' times' END,
    finished_at  = CASE WHEN attempts < $1 THEN NULL ELSE now() END,
    locked_by    = NULL,
    locked_until = NULL
WHERE state = 'running'
  AND `

const completeJobSQL = `
UPDATE wfm.job
SET state = 'done', result = $3, finished_at = now(), locked_by = NULL, locked_until = NULL
WHERE id = $1 AND state = 'running' AND locked_by = $2
RETURNING id`

const failJobSQL = `
UPDATE wfm.job
SET state = 'failed', error_code = $3, error = $4, finished_at = now(), locked_by = NULL, locked_until = NULL
WHERE id = $1 AND state = 'running' AND locked_by = $2
RETURNING id`

const heldJobSQL = `SELECT EXISTS (SELECT 1 FROM wfm.job WHERE id = $1 AND state = 'running' AND locked_by = $2)`

const lastScheduleJobSQL = `
SELECT ` + jobColumns + `
FROM wfm.job
WHERE domain_id = $1 AND working_schedule_id = $2
ORDER BY id DESC
LIMIT 1`

type JobManager interface {
	Enqueue(ctx context.Context, job *model.Job) (*model.Job, error)
	Claim(ctx context.Context, owner string, lease time.Duration, kinds []model.JobKind) (*model.Job, error)
	Requeue(ctx context.Context, owner string) ([]int64, error)
	RecoverExpired(ctx context.Context) ([]int64, error)
	Complete(ctx context.Context, owner string, id int64, result json.RawMessage) error
	Fail(ctx context.Context, owner string, id int64, code, reason string) error
	Held(ctx context.Context, owner string, id int64) (bool, error)
	LastOfSchedule(ctx context.Context, domainID, scheduleID int64) (*model.Job, error)
}

type Job struct {
	db dbsql.Store
}

func NewJob(db dbsql.Store) *Job {
	return &Job{db: db}
}

// Enqueue returns nil when that periodic run is already queued.
func (j *Job) Enqueue(ctx context.Context, job *model.Job) (*model.Job, error) {
	out, err := j.get(ctx, enqueueJobSQL, job.Kind, job.DomainID, job.WorkingScheduleID, job.CreatedBy, job.CreatedIP, job.ScheduledFor)
	if out == nil && err == nil && job.WorkingScheduleID != nil {
		return nil, errors.Wrap(ErrJobActive, errors.WithValue("working_schedule_id", *job.WorkingScheduleID))
	}

	return out, err
}

// Claim runs the oldest queued job of kinds under owner's lease; nil when there is none.
// A job of another kind waits for a leader that has its handler.
func (j *Job) Claim(ctx context.Context, owner string, lease time.Duration, kinds []model.JobKind) (*model.Job, error) {
	return j.get(ctx, claimJobSQL, owner, lease.Milliseconds(), kinds)
}

// Requeue releases every job owner runs: on losing the leadership, and at the start of a term for jobs of its last one.
func (j *Job) Requeue(ctx context.Context, owner string) ([]int64, error) {
	return j.release(ctx, "locked_by = $2", owner)
}

func (j *Job) RecoverExpired(ctx context.Context) ([]int64, error) {
	return j.release(ctx, "locked_until < now()")
}

func (j *Job) Complete(ctx context.Context, owner string, id int64, result json.RawMessage) error {
	return j.finish(ctx, completeJobSQL, id, owner, result)
}

// Fail records code, the id of the error, for the front and reason for people.
func (j *Job) Fail(ctx context.Context, owner string, id int64, code, reason string) error {
	return j.finish(ctx, failJobSQL, id, owner, code, reason)
}

// Held tells whether owner still runs the job: a canceled job or a lost lease is no longer its.
func (j *Job) Held(ctx context.Context, owner string, id int64) (bool, error) {
	var held bool
	if err := j.db.Primary().Get(ctx, &held, heldJobSQL, id, owner); err != nil {
		return false, err
	}

	return held, nil
}

func (j *Job) LastOfSchedule(ctx context.Context, domainID, scheduleID int64) (*model.Job, error) {
	return j.get(ctx, lastScheduleJobSQL, domainID, scheduleID)
}

func (j *Job) get(ctx context.Context, sql string, args ...any) (*model.Job, error) {
	var job model.Job
	if err := j.db.Primary().Get(ctx, &job, sql, args...); err != nil {
		if errors.Is(err, dbsql.ErrNoRows) {
			return nil, nil //nolint:nilnil // no job is an answer here: an empty queue, a run already queued
		}

		return nil, err
	}

	return &job, nil
}

func (j *Job) release(ctx context.Context, where string, args ...any) ([]int64, error) {
	var ids []int64
	if err := j.db.Primary().Select(ctx, &ids, releaseJobsSQL+where+" RETURNING id", append([]any{jobAttempts}, args...)...); err != nil {
		return nil, err
	}

	return ids, nil
}

func (j *Job) finish(ctx context.Context, sql string, id int64, owner string, values ...any) error {
	var done int64

	err := j.db.Primary().Get(ctx, &done, sql, append([]any{id, owner}, values...)...)
	if errors.Is(err, dbsql.ErrNoRows) {
		return errors.Wrap(ErrJobLeaseLost, errors.WithValue("id", id), errors.WithValue("owner", owner))
	}

	return err
}
