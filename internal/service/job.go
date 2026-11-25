package service

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"
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
}

func NewJobExecutor(jobs storage.JobManager, handlers []JobHandler, cfg *config.Config, log *slog.Logger) *JobExecutor {
	e := &JobExecutor{
		jobs:     jobs,
		handlers: make(map[model.JobKind]JobHandler, len(handlers)),
		owner:    cfg.Service.NodeID,
		log:      log.With("component", "job_executor"),
		poll:     time.Second,
		timeout:  cfg.JobTimeout,
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

	poll := time.NewTicker(e.poll)

	defer poll.Stop()

	for {
		if _, err := e.jobs.RecoverExpired(ctx); err != nil {
			return err
		}

		// The lease outlives the timeout, so another leader never takes a job still running here.
		job, err := e.jobs.Claim(ctx, e.owner, e.timeout+time.Minute)
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

	go func() { done <- e.handle(run, job) }()

	poll := time.NewTicker(e.poll)
	defer poll.Stop()

	var (
		err  error
		lost bool
	)

	for waiting := true; waiting; {
		select {
		case err = <-done:
			waiting = false
		case <-poll.C:
			tick()

			if held, herr := e.jobs.Held(ctx, e.owner, job.ID); herr == nil && !held {
				lost = true

				cancel()
			}
		}
	}

	switch {
	case ctx.Err() != nil:
		// The term is over: back to the queue now rather than when the lease runs out.
		requeue, stop := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer stop()

		_, rerr := e.jobs.Requeue(requeue, e.owner)
		log.Warn("job interrupted, back in the queue", "duration", time.Since(start), "err", err, "requeue_err", rerr)
	case lost:
		log.Warn("job canceled or its lease lost, its result is not written", "duration", time.Since(start), "err", err)
	case err == nil:
		log.Info("job done", "duration", time.Since(start))
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

func (e *JobExecutor) handle(ctx context.Context, job *model.Job) (err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("panic: %v\n%s", p, debug.Stack())
		}
	}()

	return e.handlers[job.Kind].Run(ctx, job)
}
