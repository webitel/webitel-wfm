package service

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"runtime/debug"
	"slices"
	"time"

	"github.com/webitel/webitel-go-kit/pkg/errors"

	"github.com/webitel/webitel-wfm/config"
	"github.com/webitel/webitel-wfm/internal/model"
	"github.com/webitel/webitel-wfm/internal/storage"
)

// JobHandlerGroup is the fx group the executor takes its handlers from.
const JobHandlerGroup = `group:"job_handlers"`

var (
	ErrJobTimeout = errors.DeadlineExceeded("the job ran out of time", errors.WithID("service.job.timeout"))
	ErrJobFailed  = errors.Internal("the job failed", errors.WithID("service.job.failed"))
)

// JobHandler runs the jobs of one kind. It completes the job itself, as its owner (job.LockedBy),
// in the transaction that writes the result. An error fails the job with the error's id as its code.
type JobHandler interface {
	Kind() model.JobKind
	Run(ctx context.Context, job *model.Job) error
}

// JobExecutor runs queued jobs on the leader, one at a time.
type JobExecutor struct {
	jobs     storage.JobManager
	handlers map[model.JobKind]JobHandler
	owner    string
	log      *slog.Logger

	poll    time.Duration
	timeout time.Duration
	// How long a handler may run past its cancellation; the lease outlives the timeout by it.
	grace time.Duration
}

func NewJobExecutor(jobs storage.JobManager, handlers []JobHandler, cfg *config.Config, log *slog.Logger) *JobExecutor {
	e := &JobExecutor{
		jobs:     jobs,
		handlers: make(map[model.JobKind]JobHandler, len(handlers)),
		owner:    cfg.Service.NodeID,
		log:      log.With("component", "job_executor"),
		poll:     time.Second,
		timeout:  cfg.JobTimeout,
		grace:    time.Minute,
	}

	for _, h := range handlers {
		e.handlers[h.Kind()] = h
	}

	return e
}

func (e *JobExecutor) Name() string { return "job_executor" }

// Run serves one leadership term: the jobs this instance left running go back to the queue first.
func (e *JobExecutor) Run(ctx context.Context, tick func()) error {
	if _, err := e.jobs.Requeue(ctx, e.owner); err != nil {
		return err
	}

	kinds := slices.Collect(maps.Keys(e.handlers))
	poll := time.NewTicker(e.poll)

	defer poll.Stop()

	for {
		if _, err := e.jobs.RecoverExpired(ctx); err != nil {
			return err
		}

		job, err := e.jobs.Claim(ctx, e.owner, e.timeout+e.grace, kinds)
		if err != nil {
			return err
		}

		if job != nil {
			e.execute(ctx, job, tick)
		}

		select {
		case <-ctx.Done():
			return nil
		case <-poll.C:
			tick()
		}
	}
}

// execute runs the job's handler under the timeout and cancels it once the job is no longer this
// instance's: canceled by the user or its lease lost.
func (e *JobExecutor) execute(ctx context.Context, job *model.Job, tick func()) {
	start := time.Now()

	log := e.log.With("job_id", job.ID, "kind", job.Kind)
	if job.WorkingScheduleID != nil {
		log = log.With("working_schedule_id", *job.WorkingScheduleID)
	}

	run, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	done := make(chan error, 1)

	go func() { done <- e.handle(run, log, job) }()

	poll := time.NewTicker(e.poll)
	defer poll.Stop()

	var (
		err      error
		lost     bool
		canceled time.Time
	)

	for waiting := true; waiting; {
		select {
		case err = <-done:
			waiting = false
		case <-poll.C:
			if run.Err() != nil && canceled.IsZero() {
				canceled = time.Now()
			}

			// A handler still running a grace past its cancellation stops the ticks, so the leader goes unhealthy.
			if canceled.IsZero() || time.Since(canceled) < e.grace {
				tick()
			}

			held, herr := e.jobs.Held(ctx, e.owner, job.ID)

			switch {
			case herr != nil && ctx.Err() == nil:
				log.Warn("could not check the job is still this instance's", "err", herr)
			case herr == nil && !held:
				lost = true

				cancel()
			}
		}
	}

	// The handler completes the job itself, so a nil error means done even if the term or the lease ended meanwhile.
	switch {
	case err == nil:
		log.Info("job done", "duration", time.Since(start))
	case ctx.Err() != nil:
		// The term is over: back to the queue now rather than when the lease runs out.
		requeue, stop := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer stop()

		_, rerr := e.jobs.Requeue(requeue, e.owner)
		log.Warn("job interrupted, back in the queue", "duration", time.Since(start), "err", err, "requeue_err", rerr)
	case lost:
		log.Warn("job canceled or its lease lost, its result is not written", "duration", time.Since(start), "err", err)
	default:
		code := errors.ID(err)
		if errors.Is(run.Err(), context.DeadlineExceeded) {
			code = errors.ID(ErrJobTimeout)
		} else if code == "" {
			code = errors.ID(ErrJobFailed)
		}

		ferr := e.jobs.Fail(ctx, e.owner, job.ID, code, err.Error())
		log.Error("job failed", "duration", time.Since(start), "code", code, "err", err, "fail_err", ferr)
	}
}

func (e *JobExecutor) handle(ctx context.Context, log *slog.Logger, job *model.Job) (err error) {
	defer func() {
		if p := recover(); p != nil {
			log.Error("job handler panicked", "panic", p, "stack", string(debug.Stack()))
			err = fmt.Errorf("panic: %v", p)
		}
	}()

	return e.handlers[job.Kind].Run(ctx, job)
}
