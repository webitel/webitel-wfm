package storage

import (
	"context"
	"slices"
	"testing"

	"github.com/webitel/webitel-go-kit/pkg/errors"

	"github.com/webitel/webitel-wfm/infra/storage/dbsql"
	"github.com/webitel/webitel-wfm/internal/model"
)

const secondTeamSeed = `
INSERT INTO call_center.cc_team (id, domain_id, name) VALUES (3, 1, 'second');
INSERT INTO flow.calendar (id, domain_id, name, timezone_id) VALUES (3, 1, 'second', 1);`

func draftHeader(t *testing.T, ctx context.Context, s *WorkingSchedule) *model.WorkingSchedule {
	t.Helper()

	in, err := s.ReadWorkingSchedule(ctx, readOf(t, ctx, 1))
	if err != nil {
		t.Fatal(err)
	}

	return in
}

func datesOf(t *testing.T, from, to string) func(*model.WorkingSchedule) {
	t.Helper()

	start, end := model.DateFromMillis(workingDate(t, from).UnixMilli()), model.DateFromMillis(workingDate(t, to).UnixMilli())

	return func(w *model.WorkingSchedule) { w.StartDateAt, w.EndDateAt = start, end }
}

func scheduleAgents(t *testing.T, ctx context.Context, n dbsql.Node) []int64 {
	t.Helper()

	var ids []int64
	if err := n.Select(ctx, &ids, "SELECT agent_id FROM wfm.working_schedule_agent WHERE working_schedule_id = 1 ORDER BY agent_id"); err != nil {
		t.Fatal(err)
	}

	return ids
}

func TestUpdateWorkingScheduleHeaderOnPostgres(t *testing.T) {
	all := []string{"1 on 2nd", "1 on 3rd", "1 night of 5th", "2 on 3rd", "2 on 6th", "2 early on 8th"}
	cases := []struct {
		name   string
		change func(*model.WorkingSchedule)
		keep   []string
		agents []int64
	}{
		{
			name:   "the name and a time scale past midnight keep every shift",
			change: func(w *model.WorkingSchedule) { w.Name, w.StartTimeAt, w.EndTimeAt = "renamed", 1320, 360 },
			keep:   all, agents: []int64{1, 2, 5},
		},
		{
			name:   "longer dates keep every shift",
			change: datesOf(t, "2026-09-01", "2026-12-31"),
			keep:   all, agents: []int64{1, 2, 5},
		},
		{
			name:   "shorter dates drop the shifts outside them by working date",
			change: datesOf(t, "2026-11-03", "2026-11-05"),
			keep:   []string{"1 on 3rd", "1 night of 5th", "2 on 3rd"}, agents: []int64{1, 2, 5},
		},
		{
			name:   "a later start alone",
			change: datesOf(t, "2026-11-03", "2026-11-30"),
			keep:   all[1:], agents: []int64{1, 2, 5},
		},
		{
			name:   "an earlier end alone",
			change: datesOf(t, "2026-10-01", "2026-11-05"),
			keep:   []string{"1 on 2nd", "1 on 3rd", "1 night of 5th", "2 on 3rd"}, agents: []int64{1, 2, 5},
		},
		{
			name:   "a one-day schedule",
			change: datesOf(t, "2026-11-03", "2026-11-03"),
			keep:   []string{"1 on 3rd", "2 on 3rd"}, agents: []int64{1, 2, 5},
		},
		{
			name:   "a new calendar drops every shift",
			change: func(w *model.WorkingSchedule) { w.Calendar.ID = 3 },
			agents: []int64{1, 2, 5},
		},
		{
			name: "a new calendar with shorter dates drops the shifts outside them too",
			change: func(w *model.WorkingSchedule) {
				datesOf(t, "2026-11-03", "2026-11-05")(w)
				w.Calendar.ID = 3
			},
			agents: []int64{1, 2, 5},
		},
		{
			name: "a new team replaces the agents, even one in both, with every shift",
			change: func(w *model.WorkingSchedule) {
				w.Team.ID, w.Agents = 3, []*model.LookupItem{{ID: 2}, {ID: 3}}
			},
			agents: []int64{2, 3},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			db, _ := newPostgres(t, shiftSeed+secondTeamSeed)
			ctx := userCtx(1, 1)
			s := &WorkingSchedule{db: db}
			shifts := seedShiftsToDelete(t, db)
			others := without(allShifts(t, ctx, db.Primary()), idsOf(shifts, all...))

			in := draftHeader(t, ctx, s)
			c.change(in)

			out, err := s.UpdateWorkingSchedule(ctx, &model.SignedInUser{ID: 1, DomainID: 1}, in)
			if err != nil {
				t.Fatal(err)
			}

			if out.Name != in.Name || out.Team.ID != in.Team.ID || out.Calendar.ID != in.Calendar.ID ||
				out.StartDateAt != in.StartDateAt || out.EndDateAt != in.EndDateAt ||
				out.StartTimeAt != in.StartTimeAt || out.EndTimeAt != in.EndTimeAt || out.Ver != in.Ver+1 {
				t.Errorf("header %+v, want %+v at the next version", out, in)
			}

			want := append(slices.Clone(others), idsOf(shifts, c.keep...)...)
			slices.Sort(want)

			if got := allShifts(t, ctx, db.Primary()); !slices.Equal(got, want) {
				t.Errorf("shifts %v, want %v", got, want)
			}

			// The lunch and the skill belong to the shift on the 2nd.
			children := 0
			if slices.Contains(c.keep, "1 on 2nd") {
				children = 1
			}

			for _, table := range []string{"wfm.agent_working_schedule_pause", "wfm.agent_working_schedule_skill"} {
				if n := rowsOf(t, ctx, db.Primary(), table); n != children {
					t.Errorf("%s: %d rows, want %d", table, n, children)
				}
			}

			if got := scheduleAgents(t, ctx, db.Primary()); !slices.Equal(got, c.agents) {
				t.Errorf("agents %v, want %v", got, c.agents)
			}
		})
	}
}

func TestUpdateWorkingScheduleRefusalsOnPostgres(t *testing.T) {
	db, _ := newPostgres(t, shiftSeed+secondTeamSeed)
	ctx := userCtx(1, 1)
	s := &WorkingSchedule{db: db}
	seedShiftsToDelete(t, db)
	before := allShifts(t, ctx, db.Primary())

	stale := draftHeader(t, ctx, s)
	stale.Team.ID, stale.Ver = 3, stale.Ver+1

	if _, err := s.UpdateWorkingSchedule(ctx, &model.SignedInUser{ID: 1, DomainID: 1}, stale); !errors.Is(err, ErrWorkingScheduleVersionConflict) {
		t.Errorf("another version: %v, want %v", err, ErrWorkingScheduleVersionConflict)
	}

	missing := draftHeader(t, ctx, s)
	missing.ID = 99

	if _, err := s.UpdateWorkingSchedule(ctx, &model.SignedInUser{ID: 1, DomainID: 1}, missing); !errors.Is(err, dbsql.ErrNoRows) {
		t.Errorf("missing schedule: %v, want %v", err, dbsql.ErrNoRows)
	}

	foreign := draftHeader(t, ctx, s)
	if _, err := s.UpdateWorkingSchedule(userCtx(2, 2), &model.SignedInUser{ID: 2, DomainID: 2}, foreign); !errors.Is(err, dbsql.ErrNoRows) {
		t.Errorf("another domain: %v, want %v", err, dbsql.ErrNoRows)
	}

	if got := allShifts(t, ctx, db.Primary()); !slices.Equal(got, before) {
		t.Errorf("a refused update changed the shifts: %v, want %v", got, before)
	}

	if got := scheduleAgents(t, ctx, db.Primary()); !slices.Equal(got, []int64{1, 2, 5}) {
		t.Errorf("a refused update changed the agents: %v", got)
	}
}

// A shift written while the update waits for the schedule row is still dropped when it falls outside the new dates.
func TestUpdateWorkingScheduleWaitsForAShiftWriteOnPostgres(t *testing.T) {
	db, conn := newPostgres(t, shiftSeed)
	ctx := userCtx(1, 1)
	s := &WorkingSchedule{db: db}

	in := draftHeader(t, ctx, s)
	datesOf(t, "2026-11-03", "2026-11-05")(in)

	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := tx.Exec(ctx, `SELECT 1 FROM wfm.working_schedule WHERE id = 1 FOR SHARE;
INSERT INTO wfm.agent_working_schedule (domain_id, created_by, working_schedule_agent_id, schedule_at, start_at, end_at)
VALUES (1, 1, 1, '2026-11-20', '2026-11-20 09:00+02', '2026-11-20 17:00+02')`); err != nil {
		t.Fatal(err)
	}

	done := make(chan error, 1)

	go func() {
		_, err := s.UpdateWorkingSchedule(ctx, &model.SignedInUser{ID: 1, DomainID: 1}, in)
		done <- err
	}()

	if !waitForLockWait(t, ctx, db, 1) {
		t.Error("the update did not wait for the shift write")
	}

	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}

	if err := <-done; err != nil {
		t.Fatal(err)
	}

	var left int
	if err := db.Primary().Get(ctx, &left, `SELECT count(*)
FROM wfm.agent_working_schedule aws
         INNER JOIN wfm.working_schedule_agent wsa ON wsa.id = aws.working_schedule_agent_id
WHERE wsa.working_schedule_id = 1`); err != nil {
		t.Fatal(err)
	}

	if left != 0 {
		t.Errorf("%d shifts left outside the new dates", left)
	}
}
