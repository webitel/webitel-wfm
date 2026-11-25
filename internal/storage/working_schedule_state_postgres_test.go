package storage

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"

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

			if !waitForLockWait(t, ctx, db, 1) {
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

	if !waitForLockWait(t, ctx, db, 2) {
		t.Fatal("the deletes did not both wait for the row")
	}

	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}

	first, second := <-done, <-done
	if first != nil && second != nil || !errors.Is(first, dbsql.ErrNoRows) && !errors.Is(second, dbsql.ErrNoRows) {
		t.Errorf("deletes returned %v and %v, want one success and one %v", first, second, dbsql.ErrNoRows)
	}
}

func TestUpdateWorkingScheduleStateOnPostgres(t *testing.T) {
	db, conn := newPostgres(t, shiftSeed)
	ctx := userCtx(1, 1)
	s := &WorkingSchedule{db: db}

	for _, c := range []struct {
		name  string
		id    int64
		ver   int32
		state model.WorkingScheduleState
		want  error
	}{
		{name: "draft to pending", id: 1, ver: 1, state: model.WorkingScheduleStatePending},
		{name: "back to draft", id: 1, ver: 2, state: model.WorkingScheduleStateDraft},
		{name: "an older version", id: 1, ver: 2, state: model.WorkingScheduleStatePending, want: ErrWorkingScheduleVersionConflict},
		{name: "active again, its own shifts and team aside", id: 2, ver: 1, state: model.WorkingScheduleStateActive},
		{name: "active to archived", id: 2, ver: 2, state: model.WorkingScheduleStateArchived},
		{name: "out of archived", id: 4, ver: 1, state: model.WorkingScheduleStateDraft, want: ErrWorkingScheduleArchived},
		{name: "activating an archived one", id: 4, ver: 1, state: model.WorkingScheduleStateActive, want: ErrWorkingScheduleArchived},
		{name: "another domain", id: 5, ver: 1, state: model.WorkingScheduleStateDraft, want: dbsql.ErrNoRows},
		{name: "activating in another domain", id: 5, ver: 1, state: model.WorkingScheduleStateActive, want: dbsql.ErrNoRows},
		{name: "missing", id: 99, ver: 1, state: model.WorkingScheduleStateActive, want: dbsql.ErrNoRows},
	} {
		out, err := s.UpdateWorkingScheduleState(ctx, readOf(t, ctx, c.id), c.ver, c.state)
		if !errors.Is(err, c.want) || c.want == nil && (out.State != c.state || out.Ver != c.ver+1) {
			t.Errorf("%s: %+v, %v, want %v", c.name, out, err, c.want)
		}
	}

	if got, err := s.ReadWorkingSchedule(ctx, readOf(t, ctx, 4)); err != nil || got.State != model.WorkingScheduleStateArchived || got.Ver != 1 {
		t.Errorf("a refused change wrote the archived schedule: %+v, %v", got, err)
	}

	// The seed sets ids by hand; a new row would take id 1.
	if _, err := conn.Exec(ctx, "SELECT setval(pg_get_serial_sequence('wfm.working_schedule', 'id'), 100)"); err != nil {
		t.Fatal(err)
	}

	created, err := s.CreateWorkingSchedule(ctx, &model.SignedInUser{ID: 1, DomainID: 1}, &model.WorkingSchedule{
		Name: "new", State: model.WorkingScheduleStateActive, Team: model.LookupItem{ID: 1}, Calendar: model.LookupItem{ID: 1},
		StartDateAt: pgtype.Date{Time: workingDate(t, "2026-12-01"), Valid: true}, EndDateAt: pgtype.Date{Time: workingDate(t, "2026-12-31"), Valid: true},
	})
	if err != nil || created.State != model.WorkingScheduleStateDraft {
		t.Errorf("created %+v, %v; want a draft", created, err)
	}
}

// Schedule 2 is the team's active one for November; schedule 3 moved to a team of its own meets only agent 1's shifts there.
func TestActivateWorkingScheduleRefusalsOnPostgres(t *testing.T) {
	db, conn := newPostgres(t, shiftSeed+secondTeamSeed)
	ctx := userCtx(1, 1)
	s := &WorkingSchedule{db: db}

	_, err := s.UpdateWorkingScheduleState(ctx, readOf(t, ctx, 1), 1, model.WorkingScheduleStateActive)
	if vs := model.ViolationsOf(err); !errors.Is(err, ErrWorkingScheduleActivationRefused) || len(vs) != 1 || vs[0].Reason != ReasonWorkingScheduleTeamActive ||
		vs[0].AgentID != 0 || vs[0].Date.Valid || vs[0].Schedule == nil || vs[0].Schedule.ID != 2 || *vs[0].Schedule.Name != "active" {
		t.Errorf("the team's other active schedule: %v %+v", err, vs)
	}

	// Only overlapping dates count, a shared day included.
	for _, c := range []struct {
		from, to string
		refused  bool
	}{
		{from: "2026-10-01", to: "2026-10-31"},
		{from: "2026-10-01", to: "2026-11-01", refused: true},
		{from: "2026-11-30", to: "2026-12-31", refused: true},
		{from: "2026-12-01", to: "2026-12-31"},
	} {
		if _, err := conn.Exec(ctx, "UPDATE wfm.working_schedule SET state = 1, ver = 1, start_date_at = $1, end_date_at = $2 WHERE id = 1", c.from, c.to); err != nil {
			t.Fatal(err)
		}

		_, err := s.UpdateWorkingScheduleState(ctx, readOf(t, ctx, 1), 1, model.WorkingScheduleStateActive)
		if errors.Is(err, ErrWorkingScheduleActivationRefused) != c.refused || !c.refused && err != nil {
			t.Errorf("%s..%s: %v, refused %v", c.from, c.to, err, c.refused)
		}
	}

	// The 10th is agent 1's date in schedule 2, and the 16th meets its night of the 15th; draft schedule 1 does not count.
	if _, err := conn.Exec(ctx, `UPDATE wfm.working_schedule SET state = 1, ver = 1, start_date_at = '2026-10-01', end_date_at = '2026-11-30' WHERE id = 1;
UPDATE wfm.working_schedule SET team_id = 3 WHERE id = 3;
INSERT INTO wfm.agent_working_schedule (domain_id, created_by, working_schedule_agent_id, schedule_at, start_at, end_at)
VALUES (1, 1, 4, '2026-11-10', '2026-11-10 18:00+02', '2026-11-10 20:00+02'),
       (1, 1, 4, '2026-11-16', '2026-11-16 01:00+02', '2026-11-16 05:00+02'),
       (1, 1, 1, '2026-11-12', '2026-11-12 10:00+02', '2026-11-12 12:00+02');`); err != nil {
		t.Fatal(err)
	}

	var night int64
	if err := conn.QueryRow(ctx, "SELECT id FROM wfm.agent_working_schedule WHERE working_schedule_agent_id = 3 AND schedule_at = '2026-11-15'").Scan(&night); err != nil {
		t.Fatal(err)
	}

	_, err = s.UpdateWorkingScheduleState(ctx, readOf(t, ctx, 3), 1, model.WorkingScheduleStateActive)
	if got, want := violationsOf(err), []string{
		violation(1, "2026-11-10", ReasonShiftLockedDate),
		violation(1, "2026-11-16", ReasonShiftOverlap),
	}; !errors.Is(err, ErrWorkingScheduleActivationRefused) || !slices.Equal(got, want) {
		t.Fatalf("agent 1's shifts: %v %v, want %v", err, got, want)
	}

	vs := model.ViolationsOf(err)
	if by := vs[0].Schedule; by == nil || by.ID != 2 {
		t.Errorf("the locked date names %+v, want schedule 2", by)
	}

	if v := vs[1]; v.ShiftID != night || !v.StartAt.Equal(at(t, "2026-11-15 22:00+02")) || !v.EndAt.Equal(at(t, "2026-11-16 06:00+02")) {
		t.Errorf("the overlap names shift %d %v..%v, want the night shift %d", v.ShiftID, v.StartAt, v.EndAt, night)
	}

	for _, id := range []int64{1, 3} {
		if got, err := s.ReadWorkingSchedule(ctx, readOf(t, ctx, id)); err != nil || got.State != model.WorkingScheduleStateDraft || got.Ver != 1 {
			t.Errorf("a refused activation wrote schedule %d: %+v, %v", id, got, err)
		}
	}

	if _, err := conn.Exec(ctx, "DELETE FROM wfm.agent_working_schedule WHERE working_schedule_agent_id = 4 AND schedule_at IN ('2026-11-10', '2026-11-16')"); err != nil {
		t.Fatal(err)
	}

	if out, err := s.UpdateWorkingScheduleState(ctx, readOf(t, ctx, 3), 1, model.WorkingScheduleStateActive); err != nil || out.State != model.WorkingScheduleStateActive || out.Ver != 2 {
		t.Errorf("nothing in the way: %+v, %v", out, err)
	}
}

// An agent added while the activation waits for the row has no lock taken, so the activation asks for a reread.
func TestActivationWithAnAgentAddedMeanwhileOnPostgres(t *testing.T) {
	db, conn := newPostgres(t, shiftSeed+secondTeamSeed)
	ctx := userCtx(1, 1)
	s := &WorkingSchedule{db: db}

	if _, err := conn.Exec(ctx, "UPDATE wfm.working_schedule SET team_id = 3 WHERE id = 3"); err != nil {
		t.Fatal(err)
	}

	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := tx.Exec(ctx, "SELECT 1 FROM wfm.working_schedule WHERE id = 3 FOR UPDATE"); err != nil {
		t.Fatal(err)
	}

	read, done := readOf(t, ctx, 3), make(chan error, 1)

	go func() {
		_, err := s.UpdateWorkingScheduleState(ctx, read, 1, model.WorkingScheduleStateActive)
		done <- err
	}()

	if !waitForLockWait(t, ctx, db, 1) {
		t.Error("the activation did not wait for the row")
	}

	if _, err := tx.Exec(ctx, "INSERT INTO wfm.working_schedule_agent (id, domain_id, working_schedule_id, agent_id) VALUES (9, 1, 3, 2)"); err != nil {
		t.Fatal(err)
	}

	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}

	if err := <-done; !errors.Is(err, ErrWorkingScheduleVersionConflict) {
		t.Errorf("got %v, want %v", err, ErrWorkingScheduleVersionConflict)
	}
}

// An activation and a shift on the same date in the other active schedule at once: without the agent's lock both would pass.
func TestActivationSerializesWithShiftWritesOnPostgres(t *testing.T) {
	db, conn := newPostgres(t, shiftSeed+secondTeamSeed)
	ctx := userCtx(1, 1)
	user := &model.SignedInUser{ID: 1, DomainID: 1}
	s, shifts := &WorkingSchedule{db: db}, NewAgentWorkingSchedule(db)

	for day := 1; day <= 20; day++ {
		date := fmt.Sprintf("2026-11-%02d", day)

		if _, err := conn.Exec(ctx, `UPDATE wfm.working_schedule SET team_id = 3, state = 1, ver = 1 WHERE id = 3;
DELETE FROM wfm.agent_working_schedule WHERE working_schedule_agent_id IN (3, 4);`); err != nil {
			t.Fatal(err)
		}

		if _, err := conn.Exec(ctx, `INSERT INTO wfm.agent_working_schedule (domain_id, created_by, working_schedule_agent_id, schedule_at, start_at, end_at)
VALUES (1, 1, 4, $1::date, $1::date + time '09:00', $1::date + time '17:00')`, date); err != nil {
			t.Fatal(err)
		}

		var (
			wg                 sync.WaitGroup
			activated, written error
		)

		read := readOf(t, ctx, 3)
		shift := shiftAt(t, 1, date+" 18:00+02", date+" 20:00+02")

		wg.Go(func() { _, activated = s.UpdateWorkingScheduleState(ctx, read, 1, model.WorkingScheduleStateActive) })
		wg.Go(func() { _, written = shifts.CreateShifts(ctx, user, 2, []*model.Shift{shift}) })
		wg.Wait()

		switch {
		case activated == nil && errors.Is(written, ErrShiftsRefused):
		case written == nil && errors.Is(activated, ErrWorkingScheduleActivationRefused):
		default:
			t.Errorf("day %d: activation %v, shift %v; want exactly one", day, activated, written)
		}
	}
}

// Two schedules of one team with no agent in common, activated at once: only the team's lock keeps both from passing.
func TestActivationsOfOneTeamSerializeOnPostgres(t *testing.T) {
	db, conn := newPostgres(t, shiftSeed)
	ctx := userCtx(1, 1)
	s := &WorkingSchedule{db: db}

	if _, err := conn.Exec(ctx, "UPDATE wfm.working_schedule SET state = 4 WHERE id = 2; DELETE FROM wfm.working_schedule_agent WHERE id = 4"); err != nil {
		t.Fatal(err)
	}

	for round := range 20 {
		if _, err := conn.Exec(ctx, "UPDATE wfm.working_schedule SET state = 1, ver = 1 WHERE id IN (1, 3)"); err != nil {
			t.Fatal(err)
		}

		var wg sync.WaitGroup

		errs := make([]error, 2)

		for i, id := range []int64{1, 3} {
			read := readOf(t, ctx, id)

			wg.Go(func() { _, errs[i] = s.UpdateWorkingScheduleState(ctx, read, 1, model.WorkingScheduleStateActive) })
		}

		wg.Wait()

		if (errs[0] == nil) == (errs[1] == nil) || !errors.Is(cmp.Or(errs[0], errs[1]), ErrWorkingScheduleActivationRefused) {
			t.Errorf("round %d: %v; want one activated, one refused", round, errs)
		}
	}
}
