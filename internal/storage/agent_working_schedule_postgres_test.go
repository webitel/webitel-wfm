package storage

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc/codes"

	"github.com/webitel/webitel-go-kit/pkg/errors"

	"github.com/webitel/webitel-wfm/internal/model"
)

// Schedule 1 is read: Olena has two shifts on the 2nd, a night shift into the 4th, and a
// shift on the 9th whose author was deleted; Taras is absent on the 4th; Ihor has nothing. Olena also works
// the 5th in an active schedule, the 6th in another draft and the 7th in an archived one.
// The calendar works Monday in two ranges and Friday; Saturday is disabled and Sunday's
// range is special.
const calendarSeed = `
INSERT INTO directory.wbt_domain (dc, name) VALUES (1, 'domain');
INSERT INTO directory.wbt_user (id, dc, name) VALUES (1, 1, 'admin'), (2, 1, 'Olena'), (3, 1, 'Taras'), (4, 1, 'Ihor'), (9, 1, 'gone');
INSERT INTO call_center.cc_agent (id, domain_id, user_id) VALUES (1, 1, 2), (2, 1, 3), (3, 1, 4);
INSERT INTO call_center.cc_team (id, domain_id, name) VALUES (1, 1, 'team');
INSERT INTO call_center.cc_skill (id, domain_id, name) VALUES (1, 1, 'Inbound'), (2, 1, 'Collection');
INSERT INTO call_center.cc_pause_cause (id, domain_id, name) VALUES (1, 1, 'Lunch');
INSERT INTO flow.calendar_timezones (id, sys_name) VALUES (1, 'Europe/Kyiv');
INSERT INTO flow.calendar (id, domain_id, name, timezone_id, excepts, accepts)
VALUES (1, 1, 'calendar', 1,
        ARRAY [('Corporate day', 1795125600000, false, false),
               ('Founders day', 1606255200000, true, false),
               ('Cancelled', 1795212000000, false, true)]::flow.calendar_except_date[],
        ARRAY [(false, 0, 840, 1080, false), (false, 0, 540, 780, false), (false, 4, 540, 1080, false),
               (true, 5, 540, 1080, false), (false, 6, 600, 900, true)]::flow.calendar_accept_time[]);
INSERT INTO wfm.working_schedule (id, domain_id, name, state, team_id, calendar_id, start_date_at, end_date_at)
VALUES (1, 1, 'read', 1, 1, 1, '2026-11-01', '2026-11-30'),
       (2, 1, 'active', 3, 1, 1, '2026-11-01', '2026-11-30'),
       (3, 1, 'draft', 1, 1, 1, '2026-11-01', '2026-11-30'),
       (4, 1, 'archived', 4, 1, 1, '2026-11-01', '2026-11-30');
INSERT INTO wfm.working_schedule_agent (id, domain_id, working_schedule_id, agent_id)
VALUES (1, 1, 1, 1), (2, 1, 1, 2), (3, 1, 2, 1), (4, 1, 3, 1), (5, 1, 4, 1), (6, 1, 1, 3);
INSERT INTO wfm.agent_working_schedule (id, domain_id, created_by, working_schedule_agent_id, schedule_at, start_at, end_at, source, planner_run_id)
VALUES (11, 1, 1, 1, '2026-11-02', '2026-11-02 14:00+02', '2026-11-02 18:00+02', 1, NULL),
       (10, 1, 1, 1, '2026-11-02', '2026-11-02 09:00+02', '2026-11-02 13:00+02', 1, NULL),
       (12, 1, 1, 1, '2026-11-03', '2026-11-03 22:00+02', '2026-11-04 06:00+02', 2, 44),
       (13, 1, 9, 1, '2026-11-09', '2026-11-09 09:00+02', '2026-11-09 17:00+02', 1, NULL),
       (20, 1, 1, 3, '2026-11-05', '2026-11-05 09:00+02', '2026-11-05 17:00+02', 1, NULL),
       (21, 1, 1, 4, '2026-11-06', '2026-11-06 09:00+02', '2026-11-06 17:00+02', 1, NULL),
       (22, 1, 1, 5, '2026-11-07', '2026-11-07 09:00+02', '2026-11-07 17:00+02', 1, NULL);
INSERT INTO wfm.agent_working_schedule_pause (domain_id, created_by, agent_working_schedule_id, pause_cause_id, start_at, end_at)
VALUES (1, 1, 10, 1, '2026-11-02 11:00+02', '2026-11-02 11:30+02');
INSERT INTO wfm.agent_working_schedule_skill (domain_id, agent_working_schedule_id, skill_id, capacity, enabled)
VALUES (1, 10, 1, NULL, true), (1, 11, 2, 3, false);
INSERT INTO wfm.agent_absence (domain_id, created_by, updated_by, absent_at, agent_id, absence_type_id)
VALUES (1, 1, 1, '2026-11-04', 2, 1);
DELETE FROM directory.wbt_user WHERE id = 9;`

func day(d int) time.Time { return time.Date(2026, 11, d, 0, 0, 0, 0, time.UTC) }

// cells prints the calendar one line per agent and date, times in Kyiv, sorted by agent.
func cells(t *testing.T, in []*model.AgentWorkingSchedule) []string {
	t.Helper()

	kyiv, err := time.LoadLocation("Europe/Kyiv")
	if err != nil {
		t.Fatal(err)
	}

	hm := func(at time.Time) string { return at.In(kyiv).Format("15:04") }
	name := func(l *model.LookupItem) string {
		if l == nil || l.Name == nil {
			return ""
		}

		return *l.Name
	}

	slices.SortFunc(in, func(a, b *model.AgentWorkingSchedule) int { return int(a.Agent.ID - b.Agent.ID) })

	var out []string

	for _, agent := range in {
		if len(agent.Schedule) == 0 {
			out = append(out, name(&agent.Agent)+" -")
		}

		for _, cell := range agent.Schedule {
			line := name(&agent.Agent) + " " + cell.Date.Time.Format("2006-01-02")
			if cell.Locked {
				line += " locked"
			}

			if cell.Absence != nil {
				line += fmt.Sprintf(" absent %d", *cell.Absence)
			}

			for _, s := range cell.Shifts {
				line += fmt.Sprintf(" | %d %s-%s on %s v%d src%d", s.ID, hm(s.StartAt), hm(s.EndAt), s.WorkingDate.Time.Format("01-02"), s.Ver, s.Source)
				if s.PlannerRunID != nil {
					line += fmt.Sprintf(" run%d", *s.PlannerRunID)
				}

				line += " by " + name(&s.CreatedBy)

				for _, p := range s.Pauses {
					line += fmt.Sprintf(" pause %s-%s %s", hm(p.StartAt), hm(p.EndAt), name(p.Cause))
				}

				for _, k := range s.Skills {
					line += fmt.Sprintf(" skill %s cap%v on=%v", name(&k.Skill), k.Capacity != nil && *k.Capacity == 3, k.Enabled)
				}
			}

			out = append(out, line)
		}
	}

	return out
}

func TestSearchAgentWorkingScheduleOnPostgres(t *testing.T) {
	db, _ := newPostgres(t, calendarSeed)
	ctx := context.Background()
	user := &model.SignedInUser{ID: 1, DomainID: 1}
	s := NewAgentWorkingSchedule(db)

	period := func(from, to int) *model.FilterBetween {
		return &model.FilterBetween{From: model.NewTimestamp(day(from).Unix()), To: model.NewTimestamp(day(to).Unix())}
	}

	olena := "Ole*"

	cases := []struct {
		name     string
		schedule int64
		search   model.AgentWorkingScheduleSearch
		want     []string
	}{
		{
			name:   "the month",
			search: model.AgentWorkingScheduleSearch{SearchItem: model.SearchItem{Date: period(1, 30)}},
			want: []string{
				"Olena 2026-11-02" +
					" | 10 09:00-13:00 on 11-02 v1 src1 by admin pause 11:00-11:30 Lunch skill Inbound capfalse on=true" +
					" | 11 14:00-18:00 on 11-02 v1 src1 by admin skill Collection captrue on=false",
				"Olena 2026-11-03 | 12 22:00-06:00 on 11-03 v1 src2 run44 by admin",
				"Olena 2026-11-05 locked",
				"Olena 2026-11-09 | 13 09:00-17:00 on 11-09 v1 src1 by ",
				"Taras 2026-11-04 absent 1",
				"Ihor -",
			},
		},
		{
			name:   "a period of a few days",
			search: model.AgentWorkingScheduleSearch{SearchItem: model.SearchItem{Date: period(3, 5)}},
			want: []string{
				"Olena 2026-11-03 | 12 22:00-06:00 on 11-03 v1 src2 run44 by admin",
				"Olena 2026-11-05 locked",
				"Taras 2026-11-04 absent 1",
				"Ihor -",
			},
		},
		{
			name:   "a period with nothing in it lists every agent",
			search: model.AgentWorkingScheduleSearch{SearchItem: model.SearchItem{Date: period(10, 12)}},
			want:   []string{"Olena -", "Taras -", "Ihor -"},
		},
		{
			name:   "by name",
			search: model.AgentWorkingScheduleSearch{SearchItem: model.SearchItem{Date: period(4, 5), Search: &olena}},
			want:   []string{"Olena 2026-11-05 locked"},
		},
		{
			name:     "an active schedule is not locked by its own shifts",
			schedule: 2,
			search:   model.AgentWorkingScheduleSearch{SearchItem: model.SearchItem{Date: period(1, 30)}},
			want:     []string{"Olena 2026-11-05 | 20 09:00-17:00 on 11-05 v1 src1 by admin"},
		},
		{
			name:   "by agent",
			search: model.AgentWorkingScheduleSearch{SearchItem: model.SearchItem{Date: period(1, 30)}, AgentIDs: []int64{2}},
			want:   []string{"Taras 2026-11-04 absent 1"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.search.WorkingScheduleID = cmp.Or(tc.schedule, 1)

			got, err := s.SearchAgentWorkingSchedule(ctx, user, &tc.search)
			if err != nil {
				t.Fatal(err)
			}

			if lines := cells(t, got); !slices.Equal(lines, tc.want) {
				t.Errorf("calendar:\n got %s\nwant %s", strings.Join(lines, "\n     "), strings.Join(tc.want, "\n     "))
			}
		})
	}

	other, err := s.SearchAgentWorkingSchedule(ctx, &model.SignedInUser{ID: 1, DomainID: 2}, &model.AgentWorkingScheduleSearch{WorkingScheduleID: 1, SearchItem: model.SearchItem{Date: period(1, 30)}})
	if err != nil || len(other) != 0 {
		t.Errorf("another domain: %v, %v; want nothing", cells(t, other), err)
	}
}

func TestHolidaysOnPostgres(t *testing.T) {
	db, _ := newPostgres(t, calendarSeed)
	search := &model.AgentWorkingScheduleSearch{
		WorkingScheduleID: 1,
		SearchItem: model.SearchItem{Date: &model.FilterBetween{
			From: model.NewTimestamp(day(1).Unix()),
			To:   model.NewTimestamp(day(30).Unix()),
		}},
	}

	holidays, err := NewAgentWorkingSchedule(db).Holidays(context.Background(), &model.SignedInUser{ID: 1, DomainID: 1}, search)
	if err != nil {
		t.Fatal(err)
	}

	got := make([]string, 0, len(holidays))
	for _, h := range holidays {
		got = append(got, h.Date.Time.Format("2006-01-02")+" "+h.Name)
	}

	slices.Sort(got)

	// The 20th starts at 22:00 UTC on the 19th; the 25th repeats from 2020; the 21st is disabled.
	if want := []string{"2026-11-20 Corporate day", "2026-11-25 Founders day"}; !slices.Equal(got, want) {
		t.Errorf("holidays = %v, want %v", got, want)
	}
}

func TestScheduleCalendarOnPostgres(t *testing.T) {
	db, _ := newPostgres(t, calendarSeed)
	ctx := context.Background()
	s := NewAgentWorkingSchedule(db)

	c, err := s.ScheduleCalendar(ctx, &model.SignedInUser{ID: 1, DomainID: 1}, 1)
	if err != nil {
		t.Fatal(err)
	}

	week := make([]string, 0, len(c.Week))
	for _, d := range c.Week {
		week = append(week, fmt.Sprint(d.Day, d.Ranges))
	}

	want := []string{"Sunday []", "Monday [{540 780} {840 1080}]", "Tuesday []", "Wednesday []", "Thursday []", "Friday [{540 1080}]", "Saturday []"}
	if c.TimeZone != "Europe/Kyiv" || !slices.Equal(week, want) {
		t.Errorf("calendar = %s %v, want Europe/Kyiv %v", c.TimeZone, week, want)
	}

	for _, user := range []*model.SignedInUser{{ID: 1, DomainID: 2}} {
		if _, err := s.ScheduleCalendar(ctx, user, 1); errors.Code(err) != codes.NotFound {
			t.Errorf("another domain: err = %v, want not found", err)
		}
	}

	if _, err := s.ScheduleCalendar(ctx, &model.SignedInUser{ID: 1, DomainID: 1}, 99); errors.Code(err) != codes.NotFound {
		t.Errorf("unknown schedule: err = %v, want not found", err)
	}
}
