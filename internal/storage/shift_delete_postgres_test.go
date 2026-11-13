package storage

import (
	"context"
	"slices"
	"testing"
	"time"

	"google.golang.org/grpc/codes"

	"github.com/webitel/webitel-go-kit/pkg/errors"

	"github.com/webitel/webitel-wfm/infra/storage/dbsql"
	"github.com/webitel/webitel-wfm/internal/model"
)

var errRollBack = errors.New("roll the test transaction back")

func workingDate(t *testing.T, s string) time.Time {
	t.Helper()

	d, err := time.Parse(time.DateOnly, s)
	if err != nil {
		t.Fatal(err)
	}

	return d
}

func seedShiftsToDelete(t *testing.T, db dbsql.Store) map[string]int64 {
	t.Helper()

	lunch := shiftAt(t, 1, "2026-11-02 09:00+02", "2026-11-02 17:00+02", "2026-11-02 12:00+02", "2026-11-02 13:00+02")
	lunch.Skills = []*model.ShiftSkill{{Skill: model.LookupItem{ID: 1}, Enabled: true}}

	names := []string{"1 on 2nd", "1 on 3rd", "1 night of 5th", "2 on 3rd", "2 on 6th", "2 early on 8th"}
	in := []*model.Shift{
		lunch,
		shiftAt(t, 1, "2026-11-03 09:00+02", "2026-11-03 17:00+02"),
		shiftAt(t, 1, "2026-11-05 22:00+02", "2026-11-06 06:00+02"),
		shiftAt(t, 2, "2026-11-03 09:00+02", "2026-11-03 17:00+02"),
		shiftAt(t, 2, "2026-11-06 09:00+02", "2026-11-06 17:00+02"),
		shiftAt(t, 2, "2026-11-08 01:00+02", "2026-11-08 09:00+02"),
	}

	ids, err := NewAgentWorkingSchedule(db).CreateShifts(context.Background(), &model.SignedInUser{ID: 1, DomainID: 1}, 1, in)
	if err != nil {
		t.Fatalf("seed: %v %v", err, violationsOf(err))
	}

	out := make(map[string]int64, len(names))
	for i, name := range names {
		out[name] = ids[i]
	}

	return out
}

func allShifts(t *testing.T, ctx context.Context, n dbsql.Node) []int64 {
	t.Helper()

	var ids []int64
	if err := n.Select(ctx, &ids, "SELECT id FROM wfm.agent_working_schedule ORDER BY id"); err != nil {
		t.Fatal(err)
	}

	return ids
}

func rowsOf(t *testing.T, ctx context.Context, n dbsql.Node, table string) int {
	t.Helper()

	var count int
	if err := n.Get(ctx, &count, "SELECT count(*) FROM "+table); err != nil {
		t.Fatal(err)
	}

	return count
}

func idsOf(shifts map[string]int64, names ...string) []int64 {
	ids := make([]int64, 0, len(names))
	for _, name := range names {
		ids = append(ids, shifts[name])
	}

	slices.Sort(ids)

	return ids
}

func without(all, gone []int64) []int64 {
	return slices.DeleteFunc(slices.Clone(all), func(id int64) bool { return slices.Contains(gone, id) })
}

func TestDeleteShiftsOnPostgres(t *testing.T) {
	db, _ := newPostgres(t)
	ctx := context.Background()
	shifts := seedShiftsToDelete(t, db)
	before := allShifts(t, ctx, db.Primary())

	cases := []struct {
		name     string
		from, to string
		agents   []int64
		want     []string
	}{
		{
			name: "every agent over the whole schedule",
			from: "2026-10-01", to: "2026-11-30",
			want: []string{"1 on 2nd", "1 on 3rd", "1 night of 5th", "2 on 3rd", "2 on 6th", "2 early on 8th"},
		},
		{
			name: "one agent",
			from: "2026-10-01", to: "2026-11-30",
			agents: []int64{2},
			want:   []string{"2 on 3rd", "2 on 6th", "2 early on 8th"},
		},
		{
			name: "both ends are in the period",
			from: "2026-11-03", to: "2026-11-05",
			want: []string{"1 on 3rd", "1 night of 5th", "2 on 3rd"},
		},
		{
			name: "one day",
			from: "2026-11-02", to: "2026-11-02",
			want: []string{"1 on 2nd"},
		},
		{
			name: "a night shift belongs to the day it starts",
			from: "2026-11-06", to: "2026-11-06",
			want: []string{"2 on 6th"},
		},
		{
			name: "the working date is the calendar's, not UTC's",
			from: "2026-11-08", to: "2026-11-08",
			want: []string{"2 early on 8th"},
		},
		{
			name: "the UTC date of that shift is not its working date",
			from: "2026-11-07", to: "2026-11-07",
		},
		{
			name: "agents with and without shifts here",
			from: "2026-11-01", to: "2026-11-30",
			agents: []int64{1, 3, 4},
			want:   []string{"1 on 2nd", "1 on 3rd", "1 night of 5th"},
		},
		{
			name: "a period with no shifts",
			from: "2026-11-20", to: "2026-11-30",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			want := idsOf(shifts, c.want...)
			from, to := workingDate(t, c.from), workingDate(t, c.to)

			err := db.InTx(ctx, func(ctx context.Context, tx dbsql.Tx) error {
				got, err := deleteShifts(ctx, tx, 1, 1, from, to, c.agents)
				if err != nil {
					t.Fatalf("delete: %v", err)
				}

				if !slices.Equal(got, want) {
					t.Errorf("deleted %v, want %v", got, want)
				}

				if left := allShifts(t, ctx, tx.Primary()); !slices.Equal(left, without(before, want)) {
					t.Errorf("left %v, want %v", left, without(before, want))
				}

				keep := 1
				if slices.Contains(c.want, "1 on 2nd") {
					keep = 0
				}

				for _, table := range []string{"wfm.agent_working_schedule_pause", "wfm.agent_working_schedule_skill"} {
					if n := rowsOf(t, ctx, tx.Primary(), table); n != keep {
						t.Errorf("%s: %d rows, want %d", table, n, keep)
					}
				}

				return errRollBack
			})
			if !errors.Is(err, errRollBack) {
				t.Fatalf("tx: %v", err)
			}
		})
	}

	if after := allShifts(t, ctx, db.Primary()); !slices.Equal(after, before) {
		t.Errorf("the rolled back deletes left %v, want %v", after, before)
	}
}

func TestDeleteShiftsInTheCallersTransactionOnPostgres(t *testing.T) {
	db, conn := newPostgres(t)
	ctx := context.Background()
	shifts := seedShiftsToDelete(t, db)
	before := allShifts(t, ctx, db.Primary())
	from, to := workingDate(t, "2026-11-01"), workingDate(t, "2026-11-30")

	err := db.InTx(ctx, func(ctx context.Context, tx dbsql.Tx) error {
		if _, err := deleteShifts(ctx, tx, 1, 1, from, to, []int64{2}); err != nil {
			return err
		}

		return errRollBack
	})
	if !errors.Is(err, errRollBack) {
		t.Fatalf("tx: %v", err)
	}

	if after := allShifts(t, ctx, db.Primary()); !slices.Equal(after, before) {
		t.Fatalf("rolled back: left %v, want %v", after, before)
	}

	var deleted []int64

	err = db.InTx(ctx, func(ctx context.Context, tx dbsql.Tx) error {
		var err error

		deleted, err = deleteShifts(ctx, tx, 1, 1, from, to, []int64{2})

		return err
	})
	if err != nil {
		t.Fatalf("tx: %v", err)
	}

	want := idsOf(shifts, "2 on 3rd", "2 on 6th", "2 early on 8th")
	if !slices.Equal(deleted, want) {
		t.Errorf("deleted %v, want %v", deleted, want)
	}

	var left int
	if err := conn.QueryRow(ctx, "SELECT count(*) FROM wfm.agent_working_schedule WHERE id = ANY ($1)", want).Scan(&left); err != nil {
		t.Fatal(err)
	}

	if left != 0 {
		t.Errorf("%d of the committed deletes are still there", left)
	}
}

func TestDeleteShiftsRefusesOnPostgres(t *testing.T) {
	db, conn := newPostgres(t)
	ctx := context.Background()

	seedShiftsToDelete(t, db)

	if _, err := conn.Exec(ctx, `INSERT INTO wfm.agent_working_schedule (domain_id, created_by, working_schedule_agent_id, schedule_at, start_at, end_at)
VALUES (1, 1, 5, '2026-11-20', '2026-11-20 09:00+02', '2026-11-20 17:00+02')`); err != nil {
		t.Fatal(err)
	}

	before := allShifts(t, ctx, db.Primary())
	from, to := workingDate(t, "2026-10-01"), workingDate(t, "2026-11-30")

	cases := []struct {
		name             string
		domain, schedule int64
		wantErr          error
		wantCode         codes.Code
	}{
		{name: "an archived schedule", domain: 1, schedule: 4, wantErr: ErrShiftScheduleState, wantCode: codes.FailedPrecondition},
		{name: "an unknown schedule", domain: 1, schedule: 99, wantCode: codes.NotFound},
		{name: "a schedule of another domain", domain: 1, schedule: 5, wantCode: codes.NotFound},
		{name: "this schedule from another domain", domain: 2, schedule: 1, wantCode: codes.NotFound},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := db.InTx(ctx, func(ctx context.Context, tx dbsql.Tx) error {
				_, err := deleteShifts(ctx, tx, c.domain, c.schedule, from, to, nil)

				return err
			})

			if c.wantErr != nil && !errors.Is(err, c.wantErr) {
				t.Errorf("err = %v, want %v", err, c.wantErr)
			}

			if code := errors.Code(err); code != c.wantCode {
				t.Errorf("code = %v, want %v: %v", code, c.wantCode, err)
			}
		})
	}

	if after := allShifts(t, ctx, db.Primary()); !slices.Equal(after, before) {
		t.Errorf("the refusals deleted shifts: left %v, want %v", after, before)
	}
}

func TestDeleteShiftsOfEveryWritableStateOnPostgres(t *testing.T) {
	db, conn := newPostgres(t)
	ctx := context.Background()

	if _, err := conn.Exec(ctx, "UPDATE wfm.working_schedule SET state = 2 WHERE id = 3"); err != nil {
		t.Fatal(err)
	}

	from, to := workingDate(t, "2026-11-01"), workingDate(t, "2026-11-30")

	cases := []struct {
		name     string
		schedule int64
		want     int
	}{
		{name: "active", schedule: 2, want: 2},
		{name: "pending", schedule: 3, want: 1},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := db.InTx(ctx, func(ctx context.Context, tx dbsql.Tx) error {
				got, err := deleteShifts(ctx, tx, 1, c.schedule, from, to, nil)
				if err != nil {
					return err
				}

				if len(got) != c.want {
					t.Errorf("deleted %v, want %d", got, c.want)
				}

				return errRollBack
			})
			if !errors.Is(err, errRollBack) {
				t.Fatalf("tx: %v", err)
			}
		})
	}
}

func waitForLockWait(t *testing.T, ctx context.Context, db dbsql.Store) bool {
	t.Helper()

	for range 100 {
		var waiting bool
		if err := db.Primary().Get(ctx, &waiting,
			"SELECT EXISTS (SELECT 1 FROM pg_stat_activity WHERE datname = current_database() AND wait_event_type = 'Lock')"); err != nil {
			t.Fatal(err)
		}

		if waiting {
			return true
		}

		time.Sleep(20 * time.Millisecond)
	}

	return false
}

func TestDeleteShiftsHoldsShiftWritesOnPostgres(t *testing.T) {
	db, conn := newPostgres(t)
	ctx := context.Background()
	user := &model.SignedInUser{ID: 1, DomainID: 1}

	seedShiftsToDelete(t, db)
	from, to := workingDate(t, "2026-11-01"), workingDate(t, "2026-11-30")
	later := []*model.Shift{shiftAt(t, 1, "2026-11-20 09:00+02", "2026-11-20 17:00+02")}

	deleted := make(chan struct{})
	release := make(chan struct{})
	deleteDone := make(chan error, 1)

	go func() {
		deleteDone <- db.InTx(ctx, func(ctx context.Context, tx dbsql.Tx) error {
			_, err := deleteShifts(ctx, tx, 1, 1, from, to, nil)

			close(deleted)

			if err != nil {
				return err
			}

			<-release

			return nil
		})
	}()

	<-deleted

	written := make(chan error, 1)

	go func() {
		_, err := NewAgentWorkingSchedule(db).CreateShifts(ctx, user, 1, later)
		written <- err
	}()

	if !waitForLockWait(t, ctx, db) {
		t.Error("the shift write does not wait for the delete")
	}

	var (
		writeErr   error
		wroteEarly bool
	)

	select {
	case writeErr = <-written:
		wroteEarly = true

		t.Error("the shift was written before the delete committed")
	default:
	}

	close(release)

	if err := <-deleteDone; err != nil {
		t.Fatalf("delete: %v", err)
	}

	if !wroteEarly {
		writeErr = <-written
	}

	if writeErr != nil {
		t.Fatalf("write: %v %v", writeErr, violationsOf(writeErr))
	}

	if n := shiftsIn(t, conn); n != 1 {
		t.Errorf("%d shifts in the schedule, want only the one written after the delete", n)
	}
}

func TestDeleteShiftsWaitsForAShiftWriteOnPostgres(t *testing.T) {
	db, conn := newPostgres(t)
	ctx := context.Background()

	seedShiftsToDelete(t, db)

	from, to := workingDate(t, "2026-11-01"), workingDate(t, "2026-11-30")

	writer, err := conn.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = writer.Rollback(ctx) }()

	if _, err := writer.Exec(ctx, "SELECT 1 FROM wfm.working_schedule WHERE id = 1 FOR SHARE"); err != nil {
		t.Fatal(err)
	}

	var inFlight int64
	if err := writer.QueryRow(ctx, `INSERT INTO wfm.agent_working_schedule (domain_id, created_by, working_schedule_agent_id, schedule_at, start_at, end_at)
VALUES (1, 1, 1, '2026-11-20', '2026-11-20 09:00+02', '2026-11-20 17:00+02') RETURNING id`).Scan(&inFlight); err != nil {
		t.Fatal(err)
	}

	type result struct {
		ids []int64
		err error
	}

	done := make(chan result, 1)

	go func() {
		var r result

		r.err = db.InTx(ctx, func(ctx context.Context, tx dbsql.Tx) error {
			var err error

			r.ids, err = deleteShifts(ctx, tx, 1, 1, from, to, nil)

			return err
		})
		done <- r
	}()

	if !waitForLockWait(t, ctx, db) {
		t.Error("the delete does not wait for the shift write")
	}

	select {
	case r := <-done:
		t.Fatalf("the delete ran before the write committed: %v %v", r.ids, r.err)
	default:
	}

	if err := writer.Commit(ctx); err != nil {
		t.Fatal(err)
	}

	r := <-done
	if r.err != nil {
		t.Fatalf("delete: %v", r.err)
	}

	if !slices.Contains(r.ids, inFlight) {
		t.Errorf("deleted %v, missing the shift %d written meanwhile", r.ids, inFlight)
	}

	if n := shiftsIn(t, conn); n != 0 {
		t.Errorf("%d shifts left in the schedule, want 0", n)
	}
}
