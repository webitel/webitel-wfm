package storage

import (
	"context"
	"encoding/json"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"google.golang.org/grpc/codes"

	"github.com/webitel/webitel-go-kit/pkg/errors"

	"github.com/webitel/webitel-wfm/infra/storage/dbsql"
	"github.com/webitel/webitel-wfm/internal/model"
)

func planningJob(domain, schedule int64) *model.Job {
	return &model.Job{
		Kind:              model.JobKindPlanning,
		DomainID:          new(domain),
		WorkingScheduleID: new(schedule),
		CreatedBy:         new(int64(1)),
		CreatedIP:         new("10.0.0.1"),
	}
}

func dailyJob(at time.Time) *model.Job {
	return &model.Job{Kind: model.JobKindDailyAggregate, ScheduledFor: new(at)}
}

func enqueue(t *testing.T, jobs *Job, job *model.Job) *model.Job {
	t.Helper()

	out, err := jobs.Enqueue(context.Background(), job)
	if err != nil || out == nil {
		t.Fatalf("enqueue: %v, %v", out, err)
	}

	return out
}

func claim(t *testing.T, jobs *Job, owner string, lease time.Duration) *model.Job {
	t.Helper()

	job, err := jobs.Claim(context.Background(), owner, lease)
	if err != nil || job == nil {
		t.Fatalf("claim by %s: %v, %v", owner, job, err)
	}

	return job
}

func jobByID(t *testing.T, conn *pgx.Conn, id int64) *model.Job {
	t.Helper()

	var job model.Job
	if err := conn.QueryRow(context.Background(), `SELECT state, locked_by, locked_until, attempts, finished_at, error, result FROM wfm.job WHERE id = $1`, id).
		Scan(&job.State, &job.LockedBy, &job.LockedUntil, &job.Attempts, &job.FinishedAt, &job.Error, &job.Result); err != nil {
		t.Fatal(err)
	}

	return &job
}

func TestEnqueueJobOnPostgres(t *testing.T) {
	db, conn := newPostgres(t, shiftSeed)
	ctx := context.Background()
	jobs := NewJob(db)

	job := enqueue(t, jobs, planningJob(1, 1))
	if job.ID == 0 || job.State != model.JobStateQueued || *job.WorkingScheduleID != 1 || *job.DomainID != 1 ||
		*job.CreatedBy != 1 || *job.CreatedIP != "10.0.0.1" || job.Attempts != 0 || job.LockedBy != nil || job.CreatedAt.IsZero() {
		t.Fatalf("enqueued %+v", job)
	}

	if _, err := jobs.Enqueue(ctx, planningJob(1, 1)); !errors.Is(err, ErrJobActive) || errors.Code(err) != codes.FailedPrecondition {
		t.Fatalf("a second queued job: %v, want %v", err, ErrJobActive)
	}

	enqueue(t, jobs, planningJob(1, 3))
	claim(t, jobs, "a", time.Minute)

	if _, err := jobs.Enqueue(ctx, planningJob(1, 1)); !errors.Is(err, ErrJobActive) {
		t.Fatalf("a job beside a running one: %v, want %v", err, ErrJobActive)
	}

	if err := jobs.Complete(ctx, "a", job.ID, nil); err != nil {
		t.Fatal(err)
	}

	enqueue(t, jobs, planningJob(1, 1))

	if _, err := jobs.Enqueue(ctx, planningJob(1, 5)); err == nil {
		t.Fatal("enqueued a job for another domain's schedule")
	}

	night := time.Date(2026, 11, 17, 3, 0, 0, 0, time.UTC)
	enqueue(t, jobs, dailyJob(night))

	if out, err := jobs.Enqueue(ctx, dailyJob(night)); out != nil || err != nil {
		t.Fatalf("the same run again: %v, %v; want nothing", out, err)
	}

	enqueue(t, jobs, dailyJob(night.Add(24*time.Hour)))

	var n int
	if err := conn.QueryRow(ctx, "SELECT count(*) FROM wfm.job").Scan(&n); err != nil || n != 5 {
		t.Fatalf("%d jobs, %v; want 5", n, err)
	}
}

func TestEnqueueJobOnceUnderRaceOnPostgres(t *testing.T) {
	db, _ := newPostgres(t, shiftSeed)
	jobs := NewJob(db)

	const callers = 8

	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		created int
		refused int
	)

	for range callers {
		wg.Go(func() {
			_, err := jobs.Enqueue(context.Background(), planningJob(1, 2))

			mu.Lock()
			defer mu.Unlock()

			switch {
			case err == nil:
				created++
			case errors.Is(err, ErrJobActive):
				refused++
			default:
				t.Errorf("enqueue: %v", err)
			}
		})
	}

	wg.Wait()

	if created != 1 || refused != callers-1 {
		t.Fatalf("created %d, refused %d; want 1 and %d", created, refused, callers-1)
	}
}

func TestClaimJobOnPostgres(t *testing.T) {
	db, _ := newPostgres(t, shiftSeed)
	ctx := context.Background()
	jobs := NewJob(db)

	if job, err := jobs.Claim(ctx, "a", time.Minute); job != nil || err != nil {
		t.Fatalf("claim from an empty queue: %v, %v", job, err)
	}

	first := enqueue(t, jobs, planningJob(1, 1))
	second := enqueue(t, jobs, dailyJob(time.Date(2026, 11, 17, 3, 0, 0, 0, time.UTC)))

	before := time.Now()
	job := claim(t, jobs, "a", 6*time.Minute)

	if job.ID != first.ID || job.State != model.JobStateRunning || *job.LockedBy != "a" || job.Attempts != 1 || job.StartedAt == nil {
		t.Fatalf("claimed %+v, want job %d running under a", job, first.ID)
	}

	if lease := job.LockedUntil.Sub(before); lease < 6*time.Minute-5*time.Second || lease > 6*time.Minute+5*time.Second {
		t.Fatalf("lease %v, want 6m", lease)
	}

	if job := claim(t, jobs, "b", time.Minute); job.ID != second.ID {
		t.Fatalf("claimed %d, want %d", job.ID, second.ID)
	}

	if job, err := jobs.Claim(ctx, "a", time.Minute); job != nil || err != nil {
		t.Fatalf("claim with every job taken: %v, %v", job, err)
	}
}

func TestClaimJobSkipsALockedRowOnPostgres(t *testing.T) {
	db, conn := newPostgres(t, shiftSeed)
	ctx := context.Background()
	jobs := NewJob(db)

	locked := enqueue(t, jobs, planningJob(1, 1))
	free := enqueue(t, jobs, planningJob(1, 3))

	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, "SELECT 1 FROM wfm.job WHERE id = $1 FOR UPDATE", locked.ID); err != nil {
		t.Fatal(err)
	}

	wait, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	job, err := jobs.Claim(wait, "a", time.Minute)
	if err != nil || job == nil || job.ID != free.ID {
		t.Fatalf("claimed %v, %v; want job %d without waiting", job, err, free.ID)
	}
}

func TestClaimJobOnceUnderRaceOnPostgres(t *testing.T) {
	db, _ := newPostgres(t, shiftSeed)
	jobs := NewJob(db)

	const total, claimers = 30, 8

	for i := range total {
		enqueue(t, jobs, dailyJob(time.Date(2026, 1, 1, 3, 0, 0, 0, time.UTC).AddDate(0, 0, i)))
	}

	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		claimed []int64
	)

	for range claimers {
		wg.Go(func() {
			for {
				job, err := jobs.Claim(context.Background(), "a", time.Minute)
				if err != nil {
					t.Errorf("claim: %v", err)

					return
				}

				if job == nil {
					return
				}

				mu.Lock()

				claimed = append(claimed, job.ID)

				mu.Unlock()
			}
		})
	}

	wg.Wait()

	slices.Sort(claimed)

	if len(claimed) != total || len(slices.Compact(claimed)) != total {
		t.Fatalf("claimed %v, want each of %d jobs once", claimed, total)
	}
}

func TestFinishJobOnlyByItsOwnerOnPostgres(t *testing.T) {
	db, conn := newPostgres(t, shiftSeed)
	ctx := context.Background()
	jobs := NewJob(db)

	enqueue(t, jobs, planningJob(1, 1))
	job := claim(t, jobs, "a", time.Minute)

	if err := jobs.Complete(ctx, "b", job.ID, nil); !errors.Is(err, ErrJobLeaseLost) {
		t.Fatalf("complete by another owner: %v, want %v", err, ErrJobLeaseLost)
	}

	if err := jobs.Fail(ctx, "b", job.ID, "boom"); !errors.Is(err, ErrJobLeaseLost) {
		t.Fatalf("fail by another owner: %v, want %v", err, ErrJobLeaseLost)
	}

	if got := jobByID(t, conn, job.ID); got.State != model.JobStateRunning || *got.LockedBy != "a" {
		t.Fatalf("after other owners: %+v, want running under a", got)
	}

	result := json.RawMessage(`{"created": 2, "deleted": 1}`)
	if err := jobs.Complete(ctx, "a", job.ID, result); err != nil {
		t.Fatal(err)
	}

	got := jobByID(t, conn, job.ID)
	if got.State != model.JobStateDone || got.LockedBy != nil || got.LockedUntil != nil || got.FinishedAt == nil || string(got.Result) != string(result) {
		t.Fatalf("completed %+v, result %s", got, got.Result)
	}

	if err := jobs.Complete(ctx, "a", job.ID, result); !errors.Is(err, ErrJobLeaseLost) {
		t.Fatalf("complete twice: %v, want %v", err, ErrJobLeaseLost)
	}

	if err := jobs.Fail(ctx, "a", 999, "boom"); !errors.Is(err, ErrJobLeaseLost) {
		t.Fatalf("fail an unknown job: %v, want %v", err, ErrJobLeaseLost)
	}

	enqueue(t, jobs, planningJob(1, 1))
	failed := claim(t, jobs, "a", time.Minute)

	if err := jobs.Fail(ctx, "a", failed.ID, "boom"); err != nil {
		t.Fatal(err)
	}

	if got := jobByID(t, conn, failed.ID); got.State != model.JobStateFailed || *got.Error != "boom" || got.LockedBy != nil || got.FinishedAt == nil {
		t.Fatalf("failed %+v", got)
	}
}

func TestRequeueJobOnPostgres(t *testing.T) {
	db, conn := newPostgres(t, shiftSeed)
	ctx := context.Background()
	jobs := NewJob(db)

	enqueue(t, jobs, planningJob(1, 1))
	enqueue(t, jobs, planningJob(1, 3))
	mine := claim(t, jobs, "a", time.Minute)
	other := claim(t, jobs, "b", time.Minute)

	ids, err := jobs.Requeue(ctx, "a")
	if err != nil || !slices.Equal(ids, []int64{mine.ID}) {
		t.Fatalf("requeued %v, %v; want [%d]", ids, err, mine.ID)
	}

	if got := jobByID(t, conn, mine.ID); got.State != model.JobStateQueued || got.LockedBy != nil || got.LockedUntil != nil || got.Attempts != 1 {
		t.Fatalf("requeued %+v", got)
	}

	if got := jobByID(t, conn, other.ID); got.State != model.JobStateRunning || *got.LockedBy != "b" {
		t.Fatalf("another owner's job %+v", got)
	}

	if err := jobs.Complete(ctx, "a", mine.ID, nil); !errors.Is(err, ErrJobLeaseLost) {
		t.Fatalf("complete a requeued job: %v, want %v", err, ErrJobLeaseLost)
	}

	if again := claim(t, jobs, "a", time.Minute); again.ID != mine.ID || again.Attempts != 2 {
		t.Fatalf("claimed %+v, want job %d on its second attempt", again, mine.ID)
	}

	if _, err := jobs.Requeue(ctx, "a"); err != nil {
		t.Fatal(err)
	}

	got := jobByID(t, conn, mine.ID)
	if got.State != model.JobStateFailed || got.Error == nil || *got.Error != "interrupted 2 times" || got.FinishedAt == nil || got.LockedBy != nil {
		t.Fatalf("interrupted twice %+v", got)
	}

	enqueue(t, jobs, planningJob(1, 1))
}

func TestRecoverExpiredJobsOnPostgres(t *testing.T) {
	db, conn := newPostgres(t, shiftSeed)
	ctx := context.Background()
	jobs := NewJob(db)

	enqueue(t, jobs, planningJob(1, 1))
	enqueue(t, jobs, planningJob(1, 3))
	expired := claim(t, jobs, "a", -time.Second)
	held := claim(t, jobs, "b", time.Hour)

	ids, err := jobs.RecoverExpired(ctx)
	if err != nil || !slices.Equal(ids, []int64{expired.ID}) {
		t.Fatalf("recovered %v, %v; want [%d]", ids, err, expired.ID)
	}

	if got := jobByID(t, conn, expired.ID); got.State != model.JobStateQueued || got.LockedBy != nil {
		t.Fatalf("recovered %+v", got)
	}

	if got := jobByID(t, conn, held.ID); got.State != model.JobStateRunning || *got.LockedBy != "b" {
		t.Fatalf("a held lease %+v", got)
	}

	claim(t, jobs, "c", -time.Second)

	if _, err := jobs.RecoverExpired(ctx); err != nil {
		t.Fatal(err)
	}

	if got := jobByID(t, conn, expired.ID); got.State != model.JobStateFailed || *got.Error != "interrupted 2 times" {
		t.Fatalf("expired twice %+v", got)
	}
}

func TestLastJobOfScheduleOnPostgres(t *testing.T) {
	db, _ := newPostgres(t, shiftSeed)
	ctx := context.Background()
	jobs := NewJob(db)

	if job, err := jobs.LastOfSchedule(ctx, 1, 1); job != nil || err != nil {
		t.Fatalf("a schedule never planned: %v, %v", job, err)
	}

	first := enqueue(t, jobs, planningJob(1, 1))
	claim(t, jobs, "a", time.Minute)

	if err := jobs.Fail(ctx, "a", first.ID, "boom"); err != nil {
		t.Fatal(err)
	}

	last := enqueue(t, jobs, planningJob(1, 1))
	enqueue(t, jobs, planningJob(1, 3))

	job, err := jobs.LastOfSchedule(ctx, 1, 1)
	if err != nil || job == nil || job.ID != last.ID {
		t.Fatalf("last job %v, %v; want %d", job, err, last.ID)
	}

	if job, err := jobs.LastOfSchedule(ctx, 2, 1); job != nil || err != nil {
		t.Fatalf("another domain read it: %v, %v", job, err)
	}
}

func TestJobOfALostLeaseStaysWithItsNewOwnerOnPostgres(t *testing.T) {
	db, conn := newPostgres(t, shiftSeed)
	ctx := context.Background()
	jobs := NewJob(db)

	enqueue(t, jobs, planningJob(1, 1))
	job := claim(t, jobs, "old", -time.Second)

	if _, err := jobs.RecoverExpired(ctx); err != nil {
		t.Fatal(err)
	}

	claim(t, jobs, "new", time.Minute)

	if err := jobs.Complete(ctx, "old", job.ID, json.RawMessage(`{"created": 1}`)); !errors.Is(err, ErrJobLeaseLost) {
		t.Fatalf("complete by the old owner: %v, want %v", err, ErrJobLeaseLost)
	}

	if err := jobs.Fail(ctx, "old", job.ID, "boom"); !errors.Is(err, ErrJobLeaseLost) {
		t.Fatalf("fail by the old owner: %v, want %v", err, ErrJobLeaseLost)
	}

	if got := jobByID(t, conn, job.ID); got.State != model.JobStateRunning || *got.LockedBy != "new" || got.Result != nil {
		t.Fatalf("after the old owner: %+v", got)
	}
}

func TestCompleteJobInTheCallersTransactionOnPostgres(t *testing.T) {
	db, conn := newPostgres(t, shiftSeed)
	ctx := context.Background()
	jobs := NewJob(db)

	enqueue(t, jobs, planningJob(1, 1))
	job := claim(t, jobs, "a", time.Minute)
	rollBack := errors.Internal("roll back")

	err := db.InTx(ctx, func(ctx context.Context, tx dbsql.Tx) error {
		if err := NewJob(tx).Complete(ctx, "a", job.ID, nil); err != nil {
			return err
		}

		return rollBack
	})
	if !errors.Is(err, rollBack) {
		t.Fatal(err)
	}

	if got := jobByID(t, conn, job.ID); got.State != model.JobStateRunning {
		t.Fatalf("rolled back, yet the job is %s", got.State)
	}

	err = db.InTx(ctx, func(ctx context.Context, tx dbsql.Tx) error { return NewJob(tx).Complete(ctx, "b", job.ID, nil) })
	if !errors.Is(err, ErrJobLeaseLost) {
		t.Fatalf("complete by another owner in a transaction: %v, want %v", err, ErrJobLeaseLost)
	}

	if err := db.InTx(ctx, func(ctx context.Context, tx dbsql.Tx) error { return NewJob(tx).Complete(ctx, "a", job.ID, nil) }); err != nil {
		t.Fatal(err)
	}

	if got := jobByID(t, conn, job.ID); got.State != model.JobStateDone {
		t.Fatalf("committed, yet the job is %s", got.State)
	}
}
