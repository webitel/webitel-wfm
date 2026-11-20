package storage

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/webitel/webitel-go-kit/pkg/errors"

	"github.com/webitel/webitel-wfm/infra/storage/dbsql"
	"github.com/webitel/webitel-wfm/internal/model"
	"github.com/webitel/webitel-wfm/internal/model/options"
)

func readOf(t *testing.T, ctx context.Context, id int64) *options.Read {
	t.Helper()

	read, err := options.NewRead(ctx, options.WithID(id))
	if err != nil {
		t.Fatal(err)
	}

	return read
}

func TestDeleteWorkingScheduleOnPostgres(t *testing.T) {
	db, _ := newPostgres(t, shiftSeed)
	ctx := userCtx(1, 1)
	s := &WorkingSchedule{db: db}
	seedShiftsToDelete(t, db)

	if _, err := s.DeleteWorkingSchedule(ctx, readOf(t, ctx, 2)); !errors.Is(err, ErrWorkingScheduleActive) {
		t.Errorf("active: %v, want %v", err, ErrWorkingScheduleActive)
	}

	for _, id := range []int64{99, 5} {
		if _, err := s.DeleteWorkingSchedule(ctx, readOf(t, ctx, id)); !errors.Is(err, dbsql.ErrNoRows) {
			t.Errorf("schedule %d: %v, want %v", id, err, dbsql.ErrNoRows)
		}
	}

	for _, id := range []int64{1, 4} {
		if got, err := s.DeleteWorkingSchedule(ctx, readOf(t, ctx, id)); err != nil || got != id {
			t.Errorf("schedule %d: deleted %d, %v", id, got, err)
		}
	}

	var left []int64
	if err := db.Primary().Select(ctx, &left, "SELECT id FROM wfm.working_schedule ORDER BY id"); err != nil {
		t.Fatal(err)
	}

	if want := []int64{2, 3, 5}; !slices.Equal(left, want) {
		t.Errorf("schedules left %v, want %v", left, want)
	}

	for table, want := range map[string]int{
		"wfm.agent_working_schedule":       4,
		"wfm.agent_working_schedule_pause": 0,
		"wfm.agent_working_schedule_skill": 0,
		"wfm.agent_absence":                2,
	} {
		if n := rowsOf(t, ctx, db.Primary(), table); n != want {
			t.Errorf("%s: %d rows, want %d", table, n, want)
		}
	}
}

func TestArchivedScheduleKeepsItsAgentsOnPostgres(t *testing.T) {
	db, conn := newPostgres(t, shiftSeed)
	ctx := userCtx(1, 1)
	s := &WorkingSchedule{db: db}

	// The seed sets ids by hand; a new row would take id 1 and ON CONFLICT DO NOTHING would drop it.
	if _, err := conn.Exec(ctx, "SELECT setval(pg_get_serial_sequence('wfm.working_schedule_agent', 'id'), 100)"); err != nil {
		t.Fatal(err)
	}

	if _, err := s.UpdateWorkingScheduleAddAgents(ctx, readOf(t, ctx, 4), []int64{2}); !errors.Is(err, ErrWorkingScheduleArchived) {
		t.Errorf("add: %v, want %v", err, ErrWorkingScheduleArchived)
	}

	if _, err := s.UpdateWorkingScheduleRemoveAgent(ctx, readOf(t, ctx, 4), 1); !errors.Is(err, ErrWorkingScheduleArchived) {
		t.Errorf("remove: %v, want %v", err, ErrWorkingScheduleArchived)
	}

	if _, err := s.UpdateWorkingScheduleAddAgents(ctx, readOf(t, ctx, 99), []int64{2}); !errors.Is(err, dbsql.ErrNoRows) {
		t.Errorf("missing schedule: %v, want %v", err, dbsql.ErrNoRows)
	}

	var agents int
	if err := db.Primary().Get(ctx, &agents, "SELECT count(*) FROM wfm.working_schedule_agent WHERE working_schedule_id = 4"); err != nil || agents != 1 {
		t.Errorf("archived schedule has %d agents (%v), want 1", agents, err)
	}

	for _, id := range []int64{1, 2} {
		agents, err := s.UpdateWorkingScheduleAddAgents(ctx, readOf(t, ctx, id), []int64{3})
		if err != nil || !slices.ContainsFunc(agents, func(a *model.LookupItem) bool { return a.ID == 3 }) {
			t.Fatalf("add to schedule %d: %v, %v: the read-back misses the agent just added", id, agents, err)
		}

		if _, err := s.UpdateWorkingScheduleRemoveAgent(ctx, readOf(t, ctx, id), 3); err != nil {
			t.Fatalf("remove from schedule %d: %v", id, err)
		}
	}
}

// A state change in flight decides the outcome: the write waits for it instead of reading the old state.
func TestScheduleWritesWaitForAStateChangeOnPostgres(t *testing.T) {
	cases := []struct {
		name  string
		state int
		write func(s *WorkingSchedule, ctx context.Context, read *options.Read) error
		want  error
	}{
		{
			name: "activation then delete", state: 3, want: ErrWorkingScheduleActive,
			write: func(s *WorkingSchedule, ctx context.Context, read *options.Read) error {
				_, err := s.DeleteWorkingSchedule(ctx, read)

				return err
			},
		},
		{
			name: "archiving then add agents", state: 4, want: ErrWorkingScheduleArchived,
			write: func(s *WorkingSchedule, ctx context.Context, read *options.Read) error {
				_, err := s.UpdateWorkingScheduleAddAgents(ctx, read, []int64{3})

				return err
			},
		},
		{
			name: "archiving then remove agent", state: 4, want: ErrWorkingScheduleArchived,
			write: func(s *WorkingSchedule, ctx context.Context, read *options.Read) error {
				_, err := s.UpdateWorkingScheduleRemoveAgent(ctx, read, 1)

				return err
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			db, conn := newPostgres(t, shiftSeed)
			ctx := userCtx(1, 1)
			s := &WorkingSchedule{db: db}

			tx, err := conn.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}

			if _, err := tx.Exec(ctx, "UPDATE wfm.working_schedule SET state = $1 WHERE id = 1", c.state); err != nil {
				t.Fatal(err)
			}

			read, done := readOf(t, ctx, 1), make(chan error, 1)

			go func() { done <- c.write(s, ctx, read) }()

			if !waitForLockWait(t, ctx, db) {
				t.Error("the write did not wait for the state change")
			}

			if err := tx.Commit(ctx); err != nil {
				t.Fatal(err)
			}

			if err := <-done; !errors.Is(err, c.want) {
				t.Errorf("got %v, want %v", err, c.want)
			}
		})
	}
}

// FOR UPDATE, not FOR SHARE: two deletes holding the row together would deadlock on the DELETE.
func TestConcurrentDeletesOfAScheduleOnPostgres(t *testing.T) {
	db, conn := newPostgres(t, shiftSeed)
	ctx := userCtx(1, 1)
	s := &WorkingSchedule{db: db}

	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := tx.Exec(ctx, "SELECT 1 FROM wfm.working_schedule WHERE id = 1 FOR SHARE"); err != nil {
		t.Fatal(err)
	}

	read, done := readOf(t, ctx, 1), make(chan error, 2)

	for range 2 {
		go func() {
			_, err := s.DeleteWorkingSchedule(ctx, read)
			done <- err
		}()
	}

	for waiting := 0; waiting < 2; time.Sleep(20 * time.Millisecond) {
		if err := db.Primary().Get(ctx, &waiting,
			"SELECT count(*) FROM pg_stat_activity WHERE datname = current_database() AND wait_event_type = 'Lock'"); err != nil {
			t.Fatal(err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}

	first, second := <-done, <-done
	if first != nil && second != nil || !errors.Is(first, dbsql.ErrNoRows) && !errors.Is(second, dbsql.ErrNoRows) {
		t.Errorf("deletes returned %v and %v, want one success and one %v", first, second, dbsql.ErrNoRows)
	}
}
