package handler

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	pb "github.com/webitel/webitel-wfm/gen/go/api/wfm"
	"github.com/webitel/webitel-wfm/infra/server/grpccontext"
	"github.com/webitel/webitel-wfm/internal/model"
	"github.com/webitel/webitel-wfm/internal/service"
)

type searchRecorder struct {
	service.AgentWorkingScheduleManager

	search *model.AgentWorkingScheduleSearch
	out    *model.AgentWorkingScheduleCalendar
}

func (s *searchRecorder) SearchAgentsWorkingSchedule(_ context.Context, _ *model.SignedInUser, search *model.AgentWorkingScheduleSearch) (*model.AgentWorkingScheduleCalendar, error) {
	s.search = search
	if s.out == nil {
		return &model.AgentWorkingScheduleCalendar{}, nil
	}

	return s.out, nil
}

func searchCtx() context.Context {
	return grpccontext.SetUser(context.Background(), &model.SignedInUser{ID: 1, DomainID: 2})
}

// The period is optional in a search; leaving it out must not panic, and must
// reach the service as an unset period rather than a zero one.
func TestSearchAgentsWorkingScheduleWithoutPeriod(t *testing.T) {
	svc := &searchRecorder{}

	if _, err := NewAgentWorkingSchedule(svc).SearchAgentsWorkingSchedule(searchCtx(), &pb.SearchAgentsWorkingScheduleRequest{WorkingScheduleId: 7}); err != nil {
		t.Fatal(err)
	}

	if d := svc.search.SearchItem.Date; d != nil && (d.From.Valid || d.To.Valid) {
		t.Errorf("period = %+v, want unset", d)
	}
}

// utcDay is a date as the API carries it: 00:00 UTC of the day, in epoch milliseconds.
func utcDay(y int, m time.Month, d int) int64 {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC).UnixMilli()
}

func TestSearchAgentsWorkingSchedulePeriod(t *testing.T) {
	svc := &searchRecorder{}
	req := &pb.SearchAgentsWorkingScheduleRequest{WorkingScheduleId: 7, Date: &pb.DateRange{From: utcDay(2026, 11, 1), To: utcDay(2026, 11, 30)}}

	if _, err := NewAgentWorkingSchedule(svc).SearchAgentsWorkingSchedule(searchCtx(), req); err != nil {
		t.Fatal(err)
	}

	d := svc.search.SearchItem.Date
	if want := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC); !d.From.Valid || !d.From.Time.Equal(want) {
		t.Errorf("from = %+v, want %s", d.From, want)
	}

	if want := time.Date(2026, 11, 30, 0, 0, 0, 0, time.UTC); !d.To.Valid || !d.To.Time.Equal(want) {
		t.Errorf("to = %+v, want %s", d.To, want)
	}
}

// A cell holds every shift of its working date, the calendar comes keyed by day of week.
func TestSearchAgentsWorkingScheduleResponse(t *testing.T) {
	at := func(s string) time.Time {
		v, err := time.Parse(time.RFC3339, s)
		if err != nil {
			t.Fatal(err)
		}

		return v
	}
	day := func(s string) pgtype.Date {
		return pgtype.Date{Time: at(s + "T00:00:00Z"), Valid: true}
	}
	run := int64(44)
	dayOff := model.AgentAbsenceTypeDayOff

	calendar := &model.ScheduleCalendar{TimeZone: "Europe/Kyiv"}
	for d := range calendar.Week {
		calendar.Week[d].Day = time.Weekday(d)
	}

	calendar.Week[time.Monday].Ranges = []model.CalendarRange{{Start: 540, End: 780}, {Start: 840, End: 1080}}

	svc := &searchRecorder{out: &model.AgentWorkingScheduleCalendar{
		Items: []*model.AgentWorkingSchedule{{
			Agent: model.LookupItem{ID: 12},
			Schedule: []*model.AgentSchedule{
				{Date: day("2026-11-02"), Locked: true, Shifts: []*model.AgentScheduleShift{
					{DomainRecord: model.DomainRecord{ID: 10}, StartAt: at("2026-11-02T07:00:00Z"), EndAt: at("2026-11-02T11:00:00Z"), WorkingDate: day("2026-11-02"), Source: 1},
					{DomainRecord: model.DomainRecord{ID: 11}, StartAt: at("2026-11-02T20:00:00Z"), EndAt: at("2026-11-03T04:00:00Z"), WorkingDate: day("2026-11-02"), Source: 2, PlannerRunID: &run},
				}},
				{Date: day("2026-11-04"), Absence: &dayOff},
			},
		}},
		Holidays: []*model.Holiday{{Date: day("2026-11-20"), Name: "Corporate day"}},
		Calendar: calendar,
	}}

	out, err := NewAgentWorkingSchedule(svc).SearchAgentsWorkingSchedule(searchCtx(), &pb.SearchAgentsWorkingScheduleRequest{WorkingScheduleId: 7})
	if err != nil {
		t.Fatal(err)
	}

	cells := out.GetItems()[0].GetSchedule()
	if len(cells) != 2 || cells[0].GetDate() != utcDay(2026, 11, 2) || !cells[0].GetLocked() || len(cells[0].GetShifts()) != 2 {
		t.Fatalf("cells = %v, want 2026-11-02 locked with both shifts, then 2026-11-04", cells)
	}

	night := cells[0].GetShifts()[1]
	if night.GetStartAt() != at("2026-11-02T20:00:00Z").UnixMilli() || night.GetWorkingDate() != utcDay(2026, 11, 2) ||
		night.GetSource() != pb.ShiftSource_SHIFT_SOURCE_PLANNER || night.GetPlannerRunId() != 44 {
		t.Errorf("night shift = %v", night)
	}

	if cells[1].GetDate() != utcDay(2026, 11, 4) || cells[1].GetAbsence() != pb.AbsenceType_ABSENCE_TYPE_DAYOFF || len(cells[1].GetShifts()) != 0 {
		t.Errorf("absent cell = %v", cells[1])
	}

	if h := out.GetHolidays(); len(h) != 1 || h[0].GetDate() != utcDay(2026, 11, 20) {
		t.Errorf("holidays = %v", h)
	}

	week := out.GetCalendar().GetWeek()
	if out.GetCalendar().GetTimeZone() != "Europe/Kyiv" || len(week) != 7 || len(week[0].GetRanges()) != 0 ||
		len(week[1].GetRanges()) != 2 || week[1].GetRanges()[1].GetStart() != 840 {
		t.Errorf("calendar = %v, want Kyiv with all 7 days and Monday's two ranges", out.GetCalendar())
	}
}
