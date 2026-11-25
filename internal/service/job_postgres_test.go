package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"go.uber.org/goleak"

	"github.com/webitel/webitel-go-kit/infra/pgw"
	"github.com/webitel/webitel-go-kit/pkg/errors"

	"github.com/webitel/webitel-wfm/config"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql"
	"github.com/webitel/webitel-wfm/internal/model"
	"github.com/webitel/webitel-wfm/internal/storage"
	"github.com/webitel/webitel-wfm/migrations"
	"github.com/webitel/webitel-wfm/migrations/migrationstest"
)

// The executor runs each job's handler in a goroutine; none may outlive a term.
func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func newJobs(t *testing.T) (*storage.Job, *pgx.Conn) {
	t.Helper()

	ctx := context.Background()
	dsn := migrationstest.Up(t, migrations.Embed)

	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}

	t.Cleanup(func() { _ = conn.Close(ctx) })

	pm, err := pgw.NewPoolManager(ctx, pgw.WithPrimaryConfig(pgw.PrimaryConfig{DSN: dsn, MaxConns: 8}))
	if err != nil {
		t.Fatalf("pool manager: %v", err)
	}

	db := dbsql.New(pm)

	t.Cleanup(func() { _ = db.Close() })

	return storage.NewJob(db), conn
}

type jobHandler struct {
	kind model.JobKind
	run  func(ctx context.Context, job *model.Job) error
}

func (h jobHandler) Kind() model.JobKind { return h.kind }

func (h jobHandler) Run(ctx context.Context, job *model.Job) error { return h.run(ctx, job) }

// completing finishes every job it gets as its owner, the way a handler writing a result does.
func completing(jobs storage.JobManager, before func(ctx context.Context, job *model.Job) error) jobHandler {
	return jobHandler{kind: model.JobKindDailyAggregate, run: func(ctx context.Context, job *model.Job) error {
		if err := before(ctx, job); err != nil {
			return err
		}

		return jobs.Complete(ctx, *job.LockedBy, job.ID, json.RawMessage(`{"days": 1}`))
	}}
}

func newExecutor(jobs storage.JobManager, handlers ...JobHandler) *JobExecutor {
	e := NewJobExecutor(jobs, handlers, &config.Config{Service: config.Service{NodeID: "a"}}, slog.New(slog.DiscardHandler))
	e.poll, e.timeout = 10*time.Millisecond, time.Minute

	return e
}

// start runs one term of e; the returned func ends it and waits for Run to return.
func start(t *testing.T, e *JobExecutor) (func(), *atomic.Int64) {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	ticks := new(atomic.Int64)

	go func() {
		defer close(done)

		_ = e.Run(ctx, func() { ticks.Add(1) })
	}()

	stop := sync.OnceFunc(func() {
		cancel()

		select {
		case <-done:
		case <-time.After(10 * time.Second):
			t.Error("Run did not return once the term ended")
		}
	})

	t.Cleanup(stop)

	return stop, ticks
}

var night = time.Date(2026, 11, 17, 3, 0, 0, 0, time.UTC)

func enqueueDaily(t *testing.T, jobs storage.JobManager, day int) int64 {
	t.Helper()

	job, err := jobs.Enqueue(context.Background(), &model.Job{Kind: model.JobKindDailyAggregate, ScheduledFor: new(night.AddDate(0, 0, day))})
	if err != nil || job == nil {
		t.Fatalf("enqueue: %v, %v", job, err)
	}

	return job.ID
}

type jobRow struct {
	state    model.JobState
	attempts int
	owner    *string
	code     *string
	error    *string
	result   string
}

func rowOf(t *testing.T, conn *pgx.Conn, id int64) jobRow {
	t.Helper()

	var r jobRow
	if err := conn.QueryRow(context.Background(), "SELECT state, attempts, locked_by, error_code, error, coalesce(result::text, '') FROM wfm.job WHERE id = $1", id).
		Scan(&r.state, &r.attempts, &r.owner, &r.code, &r.error, &r.result); err != nil {
		t.Fatal(err)
	}

	return r
}

func eventually(t *testing.T, what string, cond func() bool) {
	t.Helper()

	for deadline := time.Now().Add(5 * time.Second); !cond(); time.Sleep(5 * time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
	}
}

func TestJobExecutorRunsJobsOneAtATimeOnPostgres(t *testing.T) {
	jobs, conn := newJobs(t)

	var running, most atomic.Int32

	e := newExecutor(jobs)
	h := completing(jobs, func(_ context.Context, job *model.Job) error {
		if lease := job.LockedUntil.Sub(*job.StartedAt); lease <= e.timeout {
			t.Errorf("a lease of %v under a timeout of %v", lease, e.timeout)
		}

		n := running.Add(1)
		defer running.Add(-1)

		for m := most.Load(); n > m && !most.CompareAndSwap(m, n); m = most.Load() {
		}

		time.Sleep(20 * time.Millisecond)

		return nil
	})

	ids := []int64{enqueueDaily(t, jobs, 0), enqueueDaily(t, jobs, 1), enqueueDaily(t, jobs, 2)}
	e.handlers[h.Kind()] = h
	start(t, e)

	for _, id := range ids {
		eventually(t, "every job done", func() bool { return rowOf(t, conn, id).state == model.JobStateDone })

		if r := rowOf(t, conn, id); r.attempts != 1 || r.owner != nil || r.result != `{"days": 1}` {
			t.Errorf("job %d: %+v", id, r)
		}
	}

	if most.Load() != 1 {
		t.Errorf("%d jobs ran at once, want 1", most.Load())
	}
}

func TestJobExecutorFailsJobsWithACodeOnPostgres(t *testing.T) {
	jobs, conn := newJobs(t)
	forecast := errors.Internal("the forecast procedure failed", errors.WithID("storage.forecast.procedure"))

	runs := map[int64]func(ctx context.Context) error{
		enqueueDaily(t, jobs, 0): func(context.Context) error { return errors.Wrap(forecast, errors.WithValue("team", 1)) },
		enqueueDaily(t, jobs, 1): func(context.Context) error { return errors.New("boom") },
		enqueueDaily(t, jobs, 2): func(context.Context) error { panic("broken handler") },
		enqueueDaily(t, jobs, 3): func(ctx context.Context) error {
			<-ctx.Done()

			return ctx.Err()
		},
	}

	e := newExecutor(jobs, jobHandler{kind: model.JobKindDailyAggregate, run: func(ctx context.Context, job *model.Job) error { return runs[job.ID](ctx) }})
	e.timeout = 100 * time.Millisecond
	start(t, e)

	want := []string{"storage.forecast.procedure", "service.job.failed", "service.job.failed", "service.job.timeout"}

	for i := range 4 {
		id := int64(i + 1)

		eventually(t, "every job failed", func() bool { return rowOf(t, conn, id).state == model.JobStateFailed })

		if r := rowOf(t, conn, id); r.code == nil || *r.code != want[i] || r.error == nil || r.owner != nil {
			t.Errorf("job %d: %+v %v, want code %s", id, r, r.code, want[i])
		}
	}

	if r := rowOf(t, conn, 3); !strings.HasPrefix(*r.error, "panic: broken handler") {
		t.Errorf("a panic recorded as %q", *r.error)
	}
}

func TestJobExecutorCancelsAJobNoLongerItsOnPostgres(t *testing.T) {
	jobs, conn := newJobs(t)
	started, canceled := make(chan int64, 1), make(chan struct{})

	var returned atomic.Bool

	h := completing(jobs, func(ctx context.Context, job *model.Job) error {
		if job.ID != 1 {
			if !returned.Load() {
				t.Error("the next job ran beside the canceled one")
			}

			return nil
		}

		started <- job.ID

		<-ctx.Done()
		close(canceled)
		time.Sleep(50 * time.Millisecond)
		returned.Store(true)

		return ctx.Err()
	})

	var logs strings.Builder

	e := newExecutor(jobs, h)
	e.log = slog.New(slog.NewTextHandler(&logs, nil))

	first := enqueueDaily(t, jobs, 0)
	stop, ticks := start(t, e)

	<-started

	running := ticks.Load()

	eventually(t, "ticks while a job runs", func() bool { return ticks.Load() > running+3 })

	// What a Cancel does: the job is no longer running under this owner.
	if _, err := conn.Exec(context.Background(), `UPDATE wfm.job SET state = 'failed', error = 'canceled', locked_by = NULL, locked_until = NULL WHERE id = $1`, first); err != nil {
		t.Fatal(err)
	}

	select {
	case <-canceled:
	case <-time.After(5 * time.Second):
		t.Fatal("the handler of a canceled job kept running")
	}

	next := enqueueDaily(t, jobs, 1)
	eventually(t, "the next job done", func() bool { return rowOf(t, conn, next).state == model.JobStateDone })

	if r := rowOf(t, conn, first); r.state != model.JobStateFailed || *r.error != "canceled" || r.code != nil || r.attempts != 1 {
		t.Errorf("the executor wrote over a canceled job: %+v", r)
	}

	stop()

	if !strings.Contains(logs.String(), "job canceled") || strings.Contains(logs.String(), "job failed") {
		t.Errorf("a canceled job logged as\n%s", logs.String())
	}
}

func TestJobExecutorRequeuesOnTheEndOfTheTermOnPostgres(t *testing.T) {
	jobs, conn := newJobs(t)
	started := make(chan struct{}, 1)
	block := atomic.Bool{}
	block.Store(true)

	h := completing(jobs, func(ctx context.Context, _ *model.Job) error {
		if !block.Load() {
			return nil
		}

		started <- struct{}{}

		<-ctx.Done()

		return ctx.Err()
	})

	id := enqueueDaily(t, jobs, 0)
	stop, _ := start(t, newExecutor(jobs, h))

	<-started
	stop()

	if r := rowOf(t, conn, id); r.state != model.JobStateQueued || r.owner != nil || r.attempts != 1 {
		t.Fatalf("a job of an ended term: %+v, want it queued after one attempt", r)
	}

	block.Store(false)
	start(t, newExecutor(jobs, h))

	eventually(t, "the job done in the next term", func() bool { return rowOf(t, conn, id).state == model.JobStateDone })

	if r := rowOf(t, conn, id); r.attempts != 2 {
		t.Errorf("done after %d attempts, want 2", r.attempts)
	}
}

func TestJobExecutorTakesBackJobsLeftRunningOnPostgres(t *testing.T) {
	jobs, conn := newJobs(t)
	ctx := context.Background()

	// Its own job from a term that ended in a crash, and another instance's with an expired lease.
	mine, theirs := enqueueDaily(t, jobs, 0), enqueueDaily(t, jobs, 1)
	if _, err := jobs.Claim(ctx, "a", time.Hour); err != nil {
		t.Fatal(err)
	}

	if _, err := jobs.Claim(ctx, "b", -time.Second); err != nil {
		t.Fatal(err)
	}

	start(t, newExecutor(jobs, completing(jobs, func(context.Context, *model.Job) error { return nil })))

	for _, id := range []int64{mine, theirs} {
		eventually(t, "jobs left running done", func() bool { return rowOf(t, conn, id).state == model.JobStateDone })

		if r := rowOf(t, conn, id); r.attempts != 2 {
			t.Errorf("job %d done after %d attempts, want 2", id, r.attempts)
		}
	}
}
