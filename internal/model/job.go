package model

import (
	"encoding/json"
	"time"
)

type JobKind string

const (
	JobKindPlanning       JobKind = "planning"
	JobKindDailyAggregate JobKind = "daily_aggregate"
)

type JobState string

const (
	JobStateQueued  JobState = "queued"
	JobStateRunning JobState = "running"
	JobStateDone    JobState = "done"
	JobStateFailed  JobState = "failed"
)

type Job struct {
	ID                int64           `db:"id"`
	Kind              JobKind         `db:"kind"`
	State             JobState        `db:"state"`
	DomainID          *int64          `db:"domain_id"`
	WorkingScheduleID *int64          `db:"working_schedule_id"`
	CreatedBy         *int64          `db:"created_by"`
	CreatedIP         *string         `db:"created_ip"`
	CreatedAt         time.Time       `db:"created_at"`
	ScheduledFor      *time.Time      `db:"scheduled_for"`
	LockedBy          *string         `db:"locked_by"`
	LockedUntil       *time.Time      `db:"locked_until"`
	Attempts          int             `db:"attempts"`
	StartedAt         *time.Time      `db:"started_at"`
	FinishedAt        *time.Time      `db:"finished_at"`
	Error             *string         `db:"error"`
	Result            json.RawMessage `db:"result"`
}
