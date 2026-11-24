package handler

import (
	"context"
	"testing"
	"time"

	"buf.build/go/protovalidate"
	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/grpc/codes"

	"github.com/webitel/webitel-go-kit/pkg/errors"
	"github.com/webitel/webitel-go-kit/pkg/etag"

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

func TestSearchAgentsWorkingSchedulePage(t *testing.T) {
	svc := &searchRecorder{out: &model.AgentWorkingScheduleCalendar{Total: 7, Next: true}}

	out, err := NewAgentWorkingSchedule(svc).SearchAgentsWorkingSchedule(searchCtx(), &pb.SearchAgentsWorkingScheduleRequest{WorkingScheduleId: 7, Page: ptr(int32(2)), Size: ptr(int32(20))})
	if err != nil {
		t.Fatal(err)
	}

	if item := svc.search.SearchItem; item.Page != 2 || item.Size != 20 {
		t.Errorf("page %d size %d reach the service, want 2 and 20", item.Page, item.Size)
	}

	if out.GetTotal() != 7 || !out.GetNext() {
		t.Errorf("total = %d, next = %v; want 7 and a next page", out.GetTotal(), out.GetNext())
	}
}

// page and size count from 1, as in the other WFM searches; both may be left out.
func TestValidatorTakesPagesFromOne(t *testing.T) {
	v, err := protovalidate.New()
	if err != nil {
		t.Fatal(err)
	}

	day := &pb.DateRange{From: utcDay(2026, 11, 20), To: utcDay(2026, 11, 20)}

	for name, tc := range map[string]struct {
		page, size *int32
		ok         bool
	}{
		"left out":       {ok: true},
		"the first page": {page: ptr(int32(1)), size: ptr(int32(1)), ok: true},
		"page 0":         {page: ptr(int32(0))},
		"size 0":         {size: ptr(int32(0))},
		"negative size":  {size: ptr(int32(-1))},
	} {
		err := v.Validate(&pb.SearchAgentsWorkingScheduleRequest{WorkingScheduleId: 7, Date: day, Page: tc.page, Size: tc.size})
		if (err == nil) != tc.ok {
			t.Errorf("%s: err = %v, want valid %v", name, err, tc.ok)
		}
	}
}

func ptr[T any](v T) *T { return &v }

// shiftRecorder records what an edit or a delete passes to the service.
type shiftRecorder struct {
	service.AgentWorkingScheduleManager

	calls        int
	schedule, id int64
	ver          int32
	in           *model.Shift
}

func (s *shiftRecorder) UpdateShift(_ context.Context, _ *model.SignedInUser, scheduleID, id int64, ver int32, in *model.Shift) (*model.AgentScheduleShift, error) {
	s.calls, s.schedule, s.id, s.ver, s.in = s.calls+1, scheduleID, id, ver, in

	return &model.AgentScheduleShift{DomainRecord: model.DomainRecord{ID: id}, Ver: ver + 1}, nil
}

func (s *shiftRecorder) DeleteShift(_ context.Context, _ *model.SignedInUser, scheduleID, id int64, ver int32) (int64, error) {
	s.calls, s.schedule, s.id, s.ver = s.calls+1, scheduleID, id, ver

	return id, nil
}

// An edit or a delete needs the etag of the very shift it names; without one it is a bad request, not a conflict.
func TestShiftWritesRequireTheShiftEtag(t *testing.T) {
	for name, tag := range map[string]string{
		"missing":         "",
		"malformed":       "not-an-etag",
		"a schedule's":    must(etag.EncodeEtag(etag.EtagWorkingSchedule, 7, 3)),
		"another shift's": must(etag.EncodeEtag(etag.EtagWorkingScheduleShift, 8, 3)),
	} {
		svc := &shiftRecorder{}
		h := NewAgentWorkingSchedule(svc)

		_, errUpdate := h.UpdateAgentWorkingScheduleShift(searchCtx(), &pb.UpdateAgentWorkingScheduleShiftRequest{WorkingScheduleId: 1, Shift: &pb.AgentScheduleShift{Id: 7, Etag: tag}})
		_, errDelete := h.DeleteAgentWorkingScheduleShift(searchCtx(), &pb.DeleteAgentWorkingScheduleShiftRequest{WorkingScheduleId: 1, Id: 7, Etag: tag})

		for _, err := range []error{errUpdate, errDelete} {
			if !errors.Is(err, ErrShiftEtag) || errors.Code(err) != codes.InvalidArgument {
				t.Errorf("%s: err = %v (%s), want %v", name, err, errors.Code(err), ErrShiftEtag)
			}
		}

		if svc.calls != 0 {
			t.Errorf("%s: the service was called", name)
		}
	}
}

func TestShiftWritesPassTheShift(t *testing.T) {
	svc := &shiftRecorder{}
	h := NewAgentWorkingSchedule(svc)
	own := must(etag.EncodeEtag(etag.EtagWorkingScheduleShift, 7, 3))
	start := time.Date(2026, 11, 3, 7, 0, 0, 0, time.UTC)
	ms := func(h int) int64 { return start.Add(time.Duration(h) * time.Hour).UnixMilli() }

	out, err := h.UpdateAgentWorkingScheduleShift(searchCtx(), &pb.UpdateAgentWorkingScheduleShiftRequest{WorkingScheduleId: 1, Shift: &pb.AgentScheduleShift{
		Id: 7, Etag: own, StartAt: ms(0), EndAt: ms(8),
		Pauses: []*pb.AgentScheduleShiftPause{{StartAt: ms(3), EndAt: ms(4), Cause: &pb.LookupEntity{Id: 5}}, {StartAt: ms(5), EndAt: ms(6)}},
		Skills: []*pb.AgentScheduleShiftSkill{{Skill: &pb.LookupEntity{Id: 9}, Enabled: true}},
	}})
	if err != nil {
		t.Fatal(err)
	}

	in := svc.in
	if svc.schedule != 1 || svc.id != 7 || svc.ver != 3 || !in.Start.Equal(start) || !in.End.Equal(start.Add(8*time.Hour)) ||
		len(in.Pauses) != 2 || in.Pauses[0].Cause.SafeID() == nil || *in.Pauses[0].Cause.SafeID() != 5 || in.Pauses[1].Cause != nil ||
		!in.Pauses[1].Start.Equal(start.Add(5*time.Hour)) || len(in.Skills) != 1 || in.Skills[0].Skill.ID != 9 || !in.Skills[0].Enabled {
		t.Errorf("update passed schedule %d, shift %d at %d: %+v", svc.schedule, svc.id, svc.ver, in)
	}

	if want := must(etag.EncodeEtag(etag.EtagWorkingScheduleShift, 7, 4)); out.GetShift().GetEtag() != want {
		t.Errorf("response etag = %q, want the next version's %q", out.GetShift().GetEtag(), want)
	}

	if got, err := h.DeleteAgentWorkingScheduleShift(searchCtx(), &pb.DeleteAgentWorkingScheduleShiftRequest{WorkingScheduleId: 1, Id: 7, Etag: own}); err != nil || got.GetId() != 7 || svc.ver != 3 {
		t.Errorf("delete = %v, %v at version %d", got, err, svc.ver)
	}
}
