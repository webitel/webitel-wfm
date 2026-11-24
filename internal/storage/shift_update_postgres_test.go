package storage

import (
	"cmp"
	"context"
	"slices"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/webitel/webitel-go-kit/pkg/errors"
	"github.com/webitel/webitel-go-kit/pkg/etag"

	"github.com/webitel/webitel-wfm/infra/storage/dbsql"
	"github.com/webitel/webitel-wfm/internal/model"
)

// Schedule 4 is archived, so this shift is seeded rather than written.
const archivedShiftSeed = `
INSERT INTO wfm.agent_working_schedule (id, domain_id, created_by, working_schedule_agent_id, schedule_at, start_at, end_at)
VALUES (100, 1, 1, 5, '2026-11-20', '2026-11-20 09:00+02', '2026-11-20 17:00+02');`

// editedShift writes agent 1's shift on the 3rd with a lunch and a skill, and agent 1's shift on the 4th.
func editedShift(t *testing.T, s *AgentWorkingSchedule) int64 {
	t.Helper()

	shift := shiftAt(t, 1, "2026-11-03 09:00+02", "2026-11-03 17:00+02", "2026-11-03 12:00+02", "2026-11-03 12:30+02")
	shift.Pauses[0].Cause = &model.LookupItem{ID: 1}
	shift.Skills = []*model.ShiftSkill{{Skill: model.LookupItem{ID: 1}, Enabled: true}}

	ids, err := s.CreateShifts(context.Background(), &model.SignedInUser{ID: 1, DomainID: 1}, 1,
		[]*model.Shift{shift, shiftAt(t, 1, "2026-11-04 09:00+02", "2026-11-04 17:00+02")})
	if err != nil {
		t.Fatalf("%v %v", err, violationsOf(err))
	}

	return ids[0]
}

func childrenOf(t *testing.T, conn *pgx.Conn, id int64) (pauses, skills int) {
	t.Helper()

	if err := conn.QueryRow(context.Background(), `SELECT (SELECT count(*) FROM wfm.agent_working_schedule_pause WHERE agent_working_schedule_id = $1),
       (SELECT count(*) FROM wfm.agent_working_schedule_skill WHERE agent_working_schedule_id = $1)`, id).Scan(&pauses, &skills); err != nil {
		t.Fatal(err)
	}

	return pauses, skills
}

func TestUpdateShiftOnPostgres(t *testing.T) {
	db, conn := newPostgres(t, shiftSeed)
	ctx := context.Background()
	s := NewAgentWorkingSchedule(db)
	id := editedShift(t, s)

	// Overlaps its own old time, which an edit replaces; the lunch goes, the skill switches off.
	in := shiftAt(t, 0, "2026-11-03 08:00+02", "2026-11-03 16:00+02")
	in.Skills = []*model.ShiftSkill{{Skill: model.LookupItem{ID: 1}}}

	out, err := s.UpdateShift(ctx, &model.SignedInUser{ID: 1, DomainID: 1}, 1, id, 1, in)
	if err != nil {
		t.Fatalf("%v %v", err, violationsOf(err))
	}

	if out.ID != id || out.Ver != 2 || !out.StartAt.Equal(in.Start) || !out.EndAt.Equal(in.End) ||
		out.WorkingDate.Time.Format("2006-01-02") != "2026-11-03" || len(out.Pauses) != 0 ||
		len(out.Skills) != 1 || out.Skills[0].Skill.ID != 1 || out.Skills[0].Enabled || out.UpdatedBy.ID != 1 {
		t.Errorf("shift = %+v", out)
	}

	typ, oid, ver, err := etag.DecodeEtag(out.MarshalProto().GetEtag())
	if err != nil || typ != etag.EtagWorkingScheduleShift || oid != id || ver != 2 {
		t.Errorf("etag = %v %d %d %v, want the shift at version 2", typ, oid, ver, err)
	}

	if pauses, skills := childrenOf(t, conn, id); pauses != 0 || skills != 1 {
		t.Errorf("rows: %d pauses, %d skills; want 0, 1", pauses, skills)
	}

	// A night shift moves to the working date it starts on.
	night := shiftAt(t, 0, "2026-11-05 22:00+02", "2026-11-06 06:00+02", "2026-11-06 02:00+02", "2026-11-06 02:30+02")
	if out, err = s.UpdateShift(ctx, &model.SignedInUser{ID: 1, DomainID: 1}, 1, id, 2, night); err != nil {
		t.Fatalf("%v %v", err, violationsOf(err))
	}

	if out.WorkingDate.Time.Format("2006-01-02") != "2026-11-05" || out.Ver != 3 || len(out.Pauses) != 1 || len(out.Skills) != 0 {
		t.Errorf("night shift = %+v", out)
	}
}

func TestUpdateShiftRefusalsOnPostgres(t *testing.T) {
	db, conn := newPostgres(t, shiftSeed+archivedShiftSeed)
	ctx := context.Background()
	user := &model.SignedInUser{ID: 1, DomainID: 1}
	s := NewAgentWorkingSchedule(db)
	id := editedShift(t, s)

	absent, err := s.CreateShifts(ctx, user, 1, []*model.Shift{shiftAt(t, 2, "2026-11-10 09:00+02", "2026-11-10 17:00+02")})
	if err != nil {
		t.Fatalf("%v %v", err, violationsOf(err))
	}

	same := func() *model.Shift { return shiftAt(t, 0, "2026-11-03 10:00+02", "2026-11-03 18:00+02") }

	for _, c := range []struct {
		name       string
		user       *model.SignedInUser
		schedule   int64
		id         int64
		ver        int32
		in         *model.Shift
		want       error
		violations []string
	}{
		{name: "another version", schedule: 1, id: id, ver: 2, in: same(), want: ErrShiftVersionConflict},
		{name: "invalid", schedule: 1, id: id, ver: 1, in: shiftAt(t, 0, "2026-11-03 18:00+02", "2026-11-03 10:00+02"), want: model.ErrShiftDuration},
		{
			name: "over the next shift", schedule: 1, id: id, ver: 1, in: shiftAt(t, 0, "2026-11-03 20:00+02", "2026-11-04 10:00+02"),
			want: ErrShiftsRefused, violations: []string{violation(1, "2026-11-03", ReasonShiftOverlap)},
		},
		{
			name: "a date of the active schedule", schedule: 1, id: id, ver: 1, in: shiftAt(t, 0, "2026-11-10 18:00+02", "2026-11-10 20:00+02"),
			want: ErrShiftsRefused, violations: []string{violation(1, "2026-11-10", ReasonShiftLockedDate)},
		},
		{
			name: "outside the schedule", schedule: 1, id: id, ver: 1, in: shiftAt(t, 0, "2026-12-01 09:00+02", "2026-12-01 17:00+02"),
			want: ErrShiftsRefused, violations: []string{violation(1, "2026-12-01", ReasonShiftOutsideSchedule)},
		},
		{
			name: "an absence", schedule: 1, id: absent[0], ver: 1, in: shiftAt(t, 0, "2026-11-11 09:00+02", "2026-11-11 17:00+02"),
			want: ErrShiftsRefused, violations: []string{violation(2, "2026-11-11", ReasonShiftAbsence)},
		},
		{name: "archived", schedule: 4, id: 100, ver: 1, in: shiftAt(t, 0, "2026-11-20 10:00+02", "2026-11-20 18:00+02"), want: ErrWorkingScheduleArchived},
		{name: "another schedule", schedule: 3, id: id, ver: 1, in: same(), want: dbsql.ErrNoRows},
		{name: "another domain", user: &model.SignedInUser{ID: 2, DomainID: 2}, schedule: 1, id: id, ver: 1, in: same(), want: dbsql.ErrNoRows},
		{name: "missing", schedule: 1, id: 999, ver: 1, in: same(), want: dbsql.ErrNoRows},
	} {
		if _, err := s.UpdateShift(ctx, cmp.Or(c.user, user), c.schedule, c.id, c.ver, c.in); !errors.Is(err, c.want) || !slices.Equal(violationsOf(err), c.violations) {
			t.Errorf("%s: %v %v, want %v %v", c.name, err, violationsOf(err), c.want, c.violations)
		}
	}

	var changed int
	if err := conn.QueryRow(ctx, "SELECT count(*) FROM wfm.agent_working_schedule WHERE ver <> 1").Scan(&changed); err != nil {
		t.Fatal(err)
	}

	if pauses, skills := childrenOf(t, conn, id); changed != 0 || pauses != 1 || skills != 1 {
		t.Errorf("a refused edit wrote: %d shifts changed, %d pauses, %d skills", changed, pauses, skills)
	}
}

func TestDeleteShiftOnPostgres(t *testing.T) {
	db, conn := newPostgres(t, shiftSeed+archivedShiftSeed)
	ctx := context.Background()
	user := &model.SignedInUser{ID: 1, DomainID: 1}
	s := NewAgentWorkingSchedule(db)
	id := editedShift(t, s)

	for _, c := range []struct {
		name     string
		schedule int64
		id       int64
		ver      int32
		want     error
	}{
		{name: "another version", schedule: 1, id: id, ver: 2, want: ErrShiftVersionConflict},
		{name: "archived", schedule: 4, id: 100, ver: 1, want: ErrWorkingScheduleArchived},
		{name: "another schedule", schedule: 3, id: id, ver: 1, want: dbsql.ErrNoRows},
		{name: "missing", schedule: 1, id: 999, ver: 1, want: dbsql.ErrNoRows},
	} {
		if _, err := s.DeleteShift(ctx, user, c.schedule, c.id, c.ver); !errors.Is(err, c.want) {
			t.Errorf("%s: %v, want %v", c.name, err, c.want)
		}
	}

	if _, err := s.DeleteShift(ctx, &model.SignedInUser{ID: 2, DomainID: 2}, 1, id, 1); !errors.Is(err, dbsql.ErrNoRows) {
		t.Errorf("another domain: %v, want %v", err, dbsql.ErrNoRows)
	}

	if got, err := s.DeleteShift(ctx, user, 1, id, 1); err != nil || got != id {
		t.Fatalf("delete = %d, %v", got, err)
	}

	if pauses, skills := childrenOf(t, conn, id); shiftsIn(t, conn) != 1 || pauses != 0 || skills != 0 {
		t.Errorf("left: %d shifts of schedule 1, %d pauses, %d skills; want the other shift only", shiftsIn(t, conn), pauses, skills)
	}

	if _, err := s.DeleteShift(ctx, user, 1, id, 1); !errors.Is(err, dbsql.ErrNoRows) {
		t.Errorf("deleted twice: %v, want %v", err, dbsql.ErrNoRows)
	}
}

// Two edits from the same read: without the row lock and the version check both would be written.
func TestUpdateShiftOnceFromOneReadOnPostgres(t *testing.T) {
	db, conn := newPostgres(t, shiftSeed)
	ctx := context.Background()
	user := &model.SignedInUser{ID: 1, DomainID: 1}
	s := NewAgentWorkingSchedule(db)

	for round := range 20 {
		id := editedShift(t, s)

		var (
			wg       sync.WaitGroup
			mu       sync.Mutex
			written  int
			conflict int
		)

		for _, start := range []string{"2026-11-03 07:00+02", "2026-11-03 10:00+02"} {
			wg.Go(func() {
				_, err := s.UpdateShift(ctx, user, 1, id, 1, shiftAt(t, 0, start, "2026-11-03 18:00+02"))

				mu.Lock()
				defer mu.Unlock()

				switch {
				case err == nil:
					written++
				case errors.Is(err, ErrShiftVersionConflict):
					conflict++
				default:
					t.Errorf("round %d: %v", round, err)
				}
			})
		}

		wg.Wait()

		if written != 1 || conflict != 1 {
			t.Fatalf("round %d: %d written, %d conflicts; want one each", round, written, conflict)
		}

		if _, err := conn.Exec(ctx, "DELETE FROM wfm.agent_working_schedule WHERE working_schedule_agent_id = 1"); err != nil {
			t.Fatal(err)
		}
	}
}

// An edit moving into a time that a create writes at once: without the agent's lock both would pass the checks.
func TestUpdateShiftSerializesWithCreateOnPostgres(t *testing.T) {
	db, conn := newPostgres(t, shiftSeed)
	ctx := context.Background()
	user := &model.SignedInUser{ID: 1, DomainID: 1}
	s := NewAgentWorkingSchedule(db)

	for round := range 20 {
		id := editedShift(t, s)

		var wg sync.WaitGroup

		errs := make([]error, 2)

		wg.Go(func() {
			_, errs[0] = s.UpdateShift(ctx, user, 1, id, 1, shiftAt(t, 0, "2026-11-03 18:00+02", "2026-11-03 22:00+02"))
		})
		wg.Go(func() {
			_, errs[1] = s.CreateShifts(ctx, user, 1, []*model.Shift{shiftAt(t, 1, "2026-11-03 19:00+02", "2026-11-03 21:00+02")})
		})
		wg.Wait()

		if (errs[0] == nil) == (errs[1] == nil) || !errors.Is(cmp.Or(errs[0], errs[1]), ErrShiftsRefused) {
			t.Fatalf("round %d: edit %v, create %v; want exactly one refused for the overlap", round, errs[0], errs[1])
		}

		if _, err := conn.Exec(ctx, "DELETE FROM wfm.agent_working_schedule WHERE working_schedule_agent_id = 1"); err != nil {
			t.Fatal(err)
		}
	}
}

// Removing the agent takes the shift through the foreign key, without the agent's lock: an edit waiting for it finds the shift gone.
func TestUpdateShiftWaitsForAShiftDeleteOnPostgres(t *testing.T) {
	db, conn := newPostgres(t, shiftSeed)
	ctx := context.Background()
	s := NewAgentWorkingSchedule(db)
	id := editedShift(t, s)

	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := tx.Exec(ctx, "DELETE FROM wfm.working_schedule_agent WHERE id = 1"); err != nil {
		t.Fatal(err)
	}

	done := make(chan error, 1)

	go func() {
		_, err := s.UpdateShift(ctx, &model.SignedInUser{ID: 1, DomainID: 1}, 1, id, 1,
			shiftAt(t, 0, "2026-11-03 10:00+02", "2026-11-03 18:00+02", "2026-11-03 12:00+02", "2026-11-03 12:30+02"))
		done <- err
	}()

	if !waitForLockWait(t, ctx, db, 1) {
		t.Error("the edit did not wait for the delete")
	}

	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}

	if err := <-done; !errors.Is(err, dbsql.ErrNoRows) {
		t.Errorf("err = %v, want %v", err, dbsql.ErrNoRows)
	}
}
