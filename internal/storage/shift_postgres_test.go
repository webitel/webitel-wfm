package storage

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/grpc/codes"

	"github.com/webitel/webitel-go-kit/infra/pgw"
	"github.com/webitel/webitel-go-kit/pkg/errors"

	"github.com/webitel/webitel-wfm/infra/storage/dbsql"
	"github.com/webitel/webitel-wfm/internal/model"
	"github.com/webitel/webitel-wfm/migrations"
	"github.com/webitel/webitel-wfm/migrations/migrationstest"
)

// Kyiv calendar. Schedule 1 is the draft written to; 2 is active, 3 another draft, 4 archived.
// Agent 2 is absent on the 11th; agent 3 is only in schedule 3; agent 5's user is deleted.
// Agent 4 is in another domain, its user deleted, working the 10th and absent on the 11th there.
const shiftSeed = `
INSERT INTO directory.wbt_domain (dc, name) VALUES (1, 'domain');
INSERT INTO directory.wbt_user (id, dc, name, deleted_at) VALUES (1, 1, 'user', NULL), (3, 1, 'deleted', now());
INSERT INTO call_center.cc_agent (id, domain_id, user_id) VALUES (1, 1, 1), (2, 1, 1), (3, 1, 1), (5, 1, 3);
INSERT INTO call_center.cc_team (id, domain_id, name) VALUES (1, 1, 'team');
INSERT INTO call_center.cc_skill (id, domain_id, name) VALUES (1, 1, 'skill');
INSERT INTO call_center.cc_pause_cause (id, domain_id, name) VALUES (1, 1, 'lunch');
INSERT INTO flow.calendar_timezones (id, sys_name) VALUES (1, 'Europe/Kyiv');
INSERT INTO flow.calendar (id, domain_id, name, timezone_id) VALUES (1, 1, 'calendar', 1);
INSERT INTO wfm.working_schedule (id, domain_id, name, state, team_id, calendar_id, start_date_at, end_date_at)
VALUES (1, 1, 'draft', 1, 1, 1, '2026-10-01', '2026-11-30'),
       (2, 1, 'active', 3, 1, 1, '2026-11-01', '2026-11-30'),
       (3, 1, 'other draft', 1, 1, 1, '2026-11-01', '2026-11-30'),
       (4, 1, 'archived', 4, 1, 1, '2026-11-01', '2026-11-30');
INSERT INTO wfm.working_schedule_agent (id, domain_id, working_schedule_id, agent_id)
VALUES (1, 1, 1, 1), (2, 1, 1, 2), (3, 1, 2, 1), (4, 1, 3, 1), (5, 1, 4, 1), (6, 1, 3, 3), (8, 1, 1, 5);
INSERT INTO wfm.agent_working_schedule (domain_id, created_by, working_schedule_agent_id, schedule_at, start_at, end_at)
VALUES (1, 1, 3, '2026-11-10', '2026-11-10 09:00+02', '2026-11-10 17:00+02'),
       (1, 1, 3, '2026-11-15', '2026-11-15 22:00+02', '2026-11-16 06:00+02'),
       (1, 1, 4, '2026-11-12', '2026-11-12 09:00+02', '2026-11-12 17:00+02');
INSERT INTO wfm.agent_absence (domain_id, created_by, updated_by, absent_at, agent_id, absence_type_id)
VALUES (1, 1, 1, '2026-11-11', 2, 1);
INSERT INTO directory.wbt_domain (dc, name) VALUES (2, 'other');
INSERT INTO directory.wbt_user (id, dc, name, deleted_at) VALUES (2, 2, 'other', now());
INSERT INTO call_center.cc_agent (id, domain_id, user_id) VALUES (4, 2, 2);
INSERT INTO call_center.cc_team (id, domain_id, name) VALUES (2, 2, 'other');
INSERT INTO flow.calendar (id, domain_id, name, timezone_id) VALUES (2, 2, 'other', 1);
INSERT INTO wfm.working_schedule (id, domain_id, name, state, team_id, calendar_id, start_date_at, end_date_at)
VALUES (5, 2, 'other domain', 3, 2, 2, '2026-11-01', '2026-11-30');
INSERT INTO wfm.working_schedule_agent (id, domain_id, working_schedule_id, agent_id) VALUES (7, 2, 5, 4);
INSERT INTO wfm.agent_working_schedule (domain_id, created_by, working_schedule_agent_id, schedule_at, start_at, end_at)
VALUES (2, 2, 7, '2026-11-10', '2026-11-10 09:00+02', '2026-11-10 17:00+02');
INSERT INTO wfm.agent_absence (domain_id, created_by, updated_by, absent_at, agent_id, absence_type_id)
VALUES (2, 2, 2, '2026-11-11', 4, 1);`

func newPostgres(t *testing.T) (dbsql.Store, *pgx.Conn) {
	t.Helper()

	ctx := context.Background()
	dsn := migrationstest.Up(t, migrations.Embed)

	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}

	t.Cleanup(func() { _ = conn.Close(ctx) })

	if _, err := conn.Exec(ctx, shiftSeed); err != nil {
		t.Fatalf("seed: %v", err)
	}

	pm, err := pgw.NewPoolManager(ctx, pgw.WithPrimaryConfig(pgw.PrimaryConfig{DSN: dsn, MaxConns: 8}))
	if err != nil {
		t.Fatalf("pool manager: %v", err)
	}

	db := dbsql.New(pm)

	t.Cleanup(func() { _ = db.Close() })

	return db, conn
}

func at(t *testing.T, s string) time.Time {
	t.Helper()

	ts, err := time.Parse("2006-01-02 15:04-07", s)
	if err != nil {
		t.Fatal(err)
	}

	return ts
}

func shiftAt(t *testing.T, agent int64, start, end string, pauses ...string) *model.Shift {
	t.Helper()

	shift := &model.Shift{AgentID: agent, Start: at(t, start), End: at(t, end)}
	for i := 0; i+1 < len(pauses); i += 2 {
		shift.Pauses = append(shift.Pauses, &model.ShiftPause{Start: at(t, pauses[i]), End: at(t, pauses[i+1])})
	}

	return shift
}

func violation(agent int64, date, reason string) string {
	return fmt.Sprintf("%d %s %s", agent, date, reason)
}

func violationsOf(err error) []string {
	vs := model.ViolationsOf(err)

	out := make([]string, 0, len(vs))
	for _, v := range vs {
		out = append(out, violation(v.OperatorID, v.Date.Time.Format("2006-01-02"), v.Reason))
	}

	return out
}

func shiftsIn(t *testing.T, conn *pgx.Conn) int {
	t.Helper()

	var n int
	if err := conn.QueryRow(context.Background(),
		"SELECT count(*) FROM wfm.agent_working_schedule WHERE working_schedule_agent_id IN (1, 2)").Scan(&n); err != nil {
		t.Fatal(err)
	}

	return n
}

func TestCreateShiftsOnPostgres(t *testing.T) {
	db, conn := newPostgres(t)
	ctx := context.Background()
	user := &model.SignedInUser{ID: 1, DomainID: 1}
	s := NewAgentWorkingSchedule(db)

	cases := []struct {
		name  string
		given []*model.Shift
		in    []*model.Shift
		want  []string
	}{
		{
			name: "a day shift",
			in:   []*model.Shift{shiftAt(t, 1, "2026-11-02 09:00+02", "2026-11-02 17:00+02", "2026-11-02 12:00+02", "2026-11-02 13:00+02")},
		},
		{
			name: "a shift of 24 hours",
			in:   []*model.Shift{shiftAt(t, 1, "2026-11-02 08:00+02", "2026-11-03 08:00+02")},
		},
		{
			name: "an agent the schedule does not have",
			in:   []*model.Shift{shiftAt(t, 3, "2026-11-02 09:00+02", "2026-11-02 17:00+02")},
			want: []string{violation(3, "2026-11-02", "storage.shift.agent_not_in_schedule")},
		},
		{
			name: "an agent whose user is deleted",
			in:   []*model.Shift{shiftAt(t, 5, "2026-11-02 09:00+02", "2026-11-02 17:00+02")},
			want: []string{violation(5, "2026-11-02", "storage.shift.agent_deactivated")},
		},
		{
			name: "a date outside the schedule",
			in:   []*model.Shift{shiftAt(t, 1, "2026-12-01 09:00+02", "2026-12-01 17:00+02")},
			want: []string{violation(1, "2026-12-01", "storage.shift.outside_schedule")},
		},
		{
			name: "an agent of another domain, deactivated, working and absent there",
			in: []*model.Shift{
				shiftAt(t, 4, "2026-11-10 09:00+02", "2026-11-10 17:00+02"),
				shiftAt(t, 4, "2026-11-11 09:00+02", "2026-11-11 17:00+02"),
			},
			want: []string{
				violation(4, "2026-11-10", "storage.shift.agent_not_in_schedule"),
				violation(4, "2026-11-11", "storage.shift.agent_not_in_schedule"),
			},
		},
		{
			name: "the day before the schedule",
			in:   []*model.Shift{shiftAt(t, 1, "2026-09-30 09:00+03", "2026-09-30 17:00+03")},
			want: []string{violation(1, "2026-09-30", "storage.shift.outside_schedule")},
		},
		{
			name: "a date the agent is absent on",
			in:   []*model.Shift{shiftAt(t, 2, "2026-11-11 09:00+02", "2026-11-11 17:00+02")},
			want: []string{violation(2, "2026-11-11", "storage.shift.absence")},
		},
		{
			name: "the absent agent on another date",
			in:   []*model.Shift{shiftAt(t, 2, "2026-11-12 09:00+02", "2026-11-12 17:00+02")},
		},
		{
			name: "two agents on the date one of them is absent on",
			in: []*model.Shift{
				shiftAt(t, 1, "2026-11-11 09:00+02", "2026-11-11 17:00+02"),
				shiftAt(t, 2, "2026-11-11 09:00+02", "2026-11-11 17:00+02"),
			},
			want: []string{violation(2, "2026-11-11", "storage.shift.absence")},
		},
		{
			name: "the last day of the schedule",
			in:   []*model.Shift{shiftAt(t, 1, "2026-11-30 09:00+02", "2026-11-30 17:00+02")},
		},
		{
			name: "a shift starting after midnight in Kyiv on the day the agent is absent",
			in:   []*model.Shift{shiftAt(t, 2, "2026-11-11 00:30+02", "2026-11-11 08:30+02")},
			want: []string{violation(2, "2026-11-11", "storage.shift.absence")},
		},
		{
			name: "a date worked in another active schedule, with no overlap in time",
			in:   []*model.Shift{shiftAt(t, 1, "2026-11-10 20:00+02", "2026-11-10 22:00+02")},
			want: []string{violation(1, "2026-11-10", "storage.shift.locked_date")},
		},
		{
			name: "a night shift of another active schedule reaching into the date",
			in:   []*model.Shift{shiftAt(t, 1, "2026-11-16 05:00+02", "2026-11-16 13:00+02")},
			want: []string{violation(1, "2026-11-16", "storage.shift.overlap")},
		},
		{
			name: "a date worked in another draft",
			in:   []*model.Shift{shiftAt(t, 1, "2026-11-12 09:00+02", "2026-11-12 17:00+02")},
		},
		{
			name:  "a shift over one of this schedule",
			given: []*model.Shift{shiftAt(t, 1, "2026-11-02 09:00+02", "2026-11-02 17:00+02")},
			in:    []*model.Shift{shiftAt(t, 1, "2026-11-02 16:00+02", "2026-11-02 20:00+02")},
			want:  []string{violation(1, "2026-11-02", "storage.shift.overlap")},
		},
		{
			name:  "a shift back to back with one of this schedule",
			given: []*model.Shift{shiftAt(t, 1, "2026-11-02 09:00+02", "2026-11-02 17:00+02")},
			in:    []*model.Shift{shiftAt(t, 1, "2026-11-02 17:00+02", "2026-11-02 21:00+02")},
		},
		{
			name:  "a shift ending as one of this schedule starts",
			given: []*model.Shift{shiftAt(t, 1, "2026-11-02 17:00+02", "2026-11-02 21:00+02")},
			in:    []*model.Shift{shiftAt(t, 1, "2026-11-02 09:00+02", "2026-11-02 17:00+02")},
		},
		{
			name: "several agents and days, every broken rule listed once, nothing written",
			in: []*model.Shift{
				shiftAt(t, 3, "2026-12-01 09:00+02", "2026-12-01 12:00+02"),
				shiftAt(t, 3, "2026-12-01 13:00+02", "2026-12-01 17:00+02"),
				shiftAt(t, 2, "2026-11-11 09:00+02", "2026-11-11 17:00+02"),
				shiftAt(t, 1, "2026-11-16 05:00+02", "2026-11-16 13:00+02"),
				shiftAt(t, 1, "2026-11-10 20:00+02", "2026-11-10 22:00+02"),
				shiftAt(t, 1, "2026-11-02 09:00+02", "2026-11-02 17:00+02"),
			},
			want: []string{
				violation(1, "2026-11-10", "storage.shift.locked_date"),
				violation(1, "2026-11-16", "storage.shift.overlap"),
				violation(2, "2026-11-11", "storage.shift.absence"),
				violation(3, "2026-12-01", "storage.shift.agent_not_in_schedule"),
				violation(3, "2026-12-01", "storage.shift.outside_schedule"),
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Cleanup(func() {
				if _, err := conn.Exec(ctx, "DELETE FROM wfm.agent_working_schedule WHERE working_schedule_agent_id IN (1, 2)"); err != nil {
					t.Errorf("clean up: %v", err)
				}
			})

			if len(tc.given) > 0 {
				if _, err := s.CreateShifts(ctx, user, 1, tc.given); err != nil {
					t.Fatalf("given: %v", err)
				}
			}

			ids, err := s.CreateShifts(ctx, user, 1, tc.in)

			if len(tc.want) == 0 {
				if err != nil {
					t.Fatalf("refused: %v %v", err, violationsOf(err))
				}

				if len(ids) != len(tc.in) || shiftsIn(t, conn) != len(tc.in)+len(tc.given) {
					t.Errorf("ids = %v, %d shifts in the schedule; want %d written", ids, shiftsIn(t, conn), len(tc.in))
				}

				return
			}

			if !errors.Is(err, ErrShiftsRefused) || ids != nil {
				t.Fatalf("err, ids = %v, %v; want %v", err, ids, ErrShiftsRefused)
			}

			if got := violationsOf(err); !slices.Equal(got, tc.want) {
				t.Errorf("violations:\n got %v\nwant %v", got, tc.want)
			}

			if n := shiftsIn(t, conn); n != len(tc.given) {
				t.Errorf("%d shifts written by a refused request", n-len(tc.given))
			}
		})
	}
}

func TestCreateShiftsWritesTheRowOnPostgres(t *testing.T) {
	db, conn := newPostgres(t)
	ctx := context.Background()

	capacity := int64(3)

	// 00:30 in Kyiv on the 3rd is still the 2nd in UTC.
	shift := shiftAt(t, 1, "2026-11-03 00:30+02", "2026-11-03 08:30+02", "2026-11-03 04:00+02", "2026-11-03 04:30+02")
	shift.Pauses[0].Cause = &model.LookupItem{ID: 1}
	shift.Skills = []*model.ShiftSkill{{Skill: model.LookupItem{ID: 1}, Capacity: &capacity, Enabled: false}}

	ids, err := NewAgentWorkingSchedule(db).CreateShifts(ctx, &model.SignedInUser{ID: 1, DomainID: 1}, 1, []*model.Shift{shift})
	if err != nil {
		t.Fatalf("%v %v", err, violationsOf(err))
	}

	var (
		date              pgtype.Date
		start, end        time.Time
		ver, source       int
		plannerRun        *int64
		pauseStart        time.Time
		pauseCause, skill int64
		skillCapacity     *int64
		enabled           bool
	)

	row := conn.QueryRow(ctx, `SELECT s.schedule_at, s.start_at, s.end_at, s.ver, s.source, s.planner_run_id,
       p.start_at, p.pause_cause_id, k.skill_id, k.capacity, k.enabled
FROM wfm.agent_working_schedule s
         INNER JOIN wfm.agent_working_schedule_pause p ON p.agent_working_schedule_id = s.id
         INNER JOIN wfm.agent_working_schedule_skill k ON k.agent_working_schedule_id = s.id
WHERE s.id = $1 AND s.working_schedule_agent_id = 1 AND s.domain_id = 1 AND s.created_by = 1`, ids[0])
	if err := row.Scan(&date, &start, &end, &ver, &source, &plannerRun, &pauseStart, &pauseCause, &skill, &skillCapacity, &enabled); err != nil {
		t.Fatalf("the shift, its pause and its skill: %v", err)
	}

	if got := date.Time.Format("2006-01-02"); got != "2026-11-03" {
		t.Errorf("working date = %s, want 2026-11-03, the day in Kyiv", got)
	}

	if !start.Equal(shift.Start) || !end.Equal(shift.End) || !pauseStart.Equal(shift.Pauses[0].Start) {
		t.Errorf("times = %v, %v, %v; want %v, %v, %v", start, end, pauseStart, shift.Start, shift.End, shift.Pauses[0].Start)
	}

	if ver != 1 || source != 1 || plannerRun != nil {
		t.Errorf("ver, source, planner_run_id = %d, %d, %v; want 1, manual, none", ver, source, plannerRun)
	}

	if pauseCause != 1 || skill != 1 || skillCapacity == nil || *skillCapacity != 3 || enabled {
		t.Errorf("pause cause, skill, capacity, enabled = %d, %d, %v, %v", pauseCause, skill, skillCapacity, enabled)
	}
}

func TestCreateShiftsIntoAnActiveScheduleOnPostgres(t *testing.T) {
	db, _ := newPostgres(t)
	in := []*model.Shift{shiftAt(t, 1, "2026-11-10 20:00+02", "2026-11-10 22:00+02")}

	if _, err := NewAgentWorkingSchedule(db).CreateShifts(context.Background(), &model.SignedInUser{ID: 1, DomainID: 1}, 2, in); err != nil {
		t.Fatalf("%v %v", err, violationsOf(err))
	}
}

func TestCreateShiftsRefusesTheScheduleOnPostgres(t *testing.T) {
	db, _ := newPostgres(t)
	ctx := context.Background()
	user := &model.SignedInUser{ID: 1, DomainID: 1}
	in := []*model.Shift{shiftAt(t, 1, "2026-11-02 09:00+02", "2026-11-02 17:00+02")}

	if _, err := NewAgentWorkingSchedule(db).CreateShifts(ctx, user, 4, in); !errors.Is(err, ErrShiftScheduleState) {
		t.Errorf("archived: err = %v, want %v", err, ErrShiftScheduleState)
	}

	if _, err := NewAgentWorkingSchedule(db).CreateShifts(ctx, user, 99, in); errors.Code(err) != codes.NotFound {
		t.Errorf("unknown: err = %v, want not found", err)
	}

	if _, err := NewAgentWorkingSchedule(db).CreateShifts(ctx, &model.SignedInUser{ID: 1, DomainID: 2}, 1, in); errors.Code(err) != codes.NotFound {
		t.Errorf("another domain: err = %v, want not found", err)
	}
}

// Without the agent's lock both writers would pass the checks and both be written.
func TestCreateShiftsSerializeOnTheAgentOnPostgres(t *testing.T) {
	db, conn := newPostgres(t)
	ctx := context.Background()
	user := &model.SignedInUser{ID: 1, DomainID: 1}
	s := NewAgentWorkingSchedule(db)

	const rounds = 30

	var (
		mu      sync.Mutex
		written int
		refused int
	)

	for day := 1; day <= rounds; day++ {
		date := time.Date(2026, 10, day, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
		pair := []*model.Shift{
			shiftAt(t, 1, date+" 09:00+03", date+" 17:00+03"),
			shiftAt(t, 1, date+" 12:00+03", date+" 20:00+03"),
		}

		var wg sync.WaitGroup

		for _, shift := range pair {
			wg.Go(func() {
				_, err := s.CreateShifts(ctx, user, 1, []*model.Shift{shift})

				mu.Lock()
				defer mu.Unlock()

				switch {
				case err == nil:
					written++
				case errors.Is(err, ErrShiftsRefused) && slices.Equal(violationsOf(err), []string{violation(1, date, "storage.shift.overlap")}):
					refused++
				default:
					t.Errorf("day %d: %v %v", day, err, violationsOf(err))
				}
			})
		}

		wg.Wait()
	}

	if written != rounds || refused != rounds {
		t.Errorf("written, refused = %d, %d; want %d each", written, refused, rounds)
	}

	if n := shiftsIn(t, conn); n != rounds {
		t.Errorf("%d shifts in the schedule, want %d: one per day", n, rounds)
	}
}
