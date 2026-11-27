package service

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/webitel/webitel-go-kit/pkg/errors"

	"github.com/webitel/webitel-wfm/infra/server/grpccontext"
	"github.com/webitel/webitel-wfm/internal/model"
	"github.com/webitel/webitel-wfm/internal/model/options"
	"github.com/webitel/webitel-wfm/internal/storage"
)

var (
	periodStart = time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	periodEnd   = time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC)
)

// scheduleStore serves one working schedule covering October 2026.
type scheduleStore struct {
	storage.WorkingScheduleManager
}

func (scheduleStore) ReadWorkingSchedule(_ context.Context, read *options.Read) (*model.WorkingSchedule, error) {
	return &model.WorkingSchedule{
		DomainRecord: model.DomainRecord{ID: read.ID()},
		Team:         model.LookupItem{ID: 5},
		StartDateAt:  pgtype.Date{Time: periodStart, Valid: true},
		EndDateAt:    pgtype.Date{Time: periodEnd, Valid: true},
	}, nil
}

// shiftStore lists every agent it holds whatever the filter, as the storage
// does when it receives no agent ids, and records the search it was given.
type shiftStore struct {
	storage.AgentWorkingScheduleManager

	agents   []*model.AgentWorkingSchedule
	holidays []*model.Holiday
	searched *model.AgentWorkingScheduleSearch
}

func (s *shiftStore) SearchAgentWorkingSchedule(_ context.Context, _ *model.SignedInUser, search *model.AgentWorkingScheduleSearch) ([]*model.AgentWorkingSchedule, error) {
	s.searched = search

	return s.agents, nil
}

func (s *shiftStore) Holidays(context.Context, *model.SignedInUser, *model.AgentWorkingScheduleSearch) ([]*model.Holiday, error) {
	return s.holidays, nil
}

func (s *shiftStore) ScheduleCalendar(context.Context, *model.SignedInUser, int64) (*model.ScheduleCalendar, error) {
	return &model.ScheduleCalendar{TimeZone: "Europe/Kyiv"}, nil
}

func signedIn() (context.Context, *model.SignedInUser) {
	user := &model.SignedInUser{ID: 1, DomainID: 2, Token: "token"}

	return grpccontext.SetUser(context.Background(), user), user
}

// A team, skill or supervisor filter that matches no agent must list no
// agents. Left as an empty id list it would read as "no filter" and list all.
func TestSearchAgentsWorkingScheduleFilterMatchingNoAgent(t *testing.T) {
	ctx, user := signedIn()
	shifts := &shiftStore{
		agents:   []*model.AgentWorkingSchedule{{Agent: model.LookupItem{ID: 10}}, {Agent: model.LookupItem{ID: 11}}},
		holidays: []*model.Holiday{{Name: "holiday"}},
	}

	svc := NewAgentWorkingSchedule(shifts, scheduleStore{}, newEngine(t, &fakeEngine{}))

	out, err := svc.SearchAgentsWorkingSchedule(ctx, user, &model.AgentWorkingScheduleSearch{
		WorkingScheduleID: 7,
		TeamIDs:           []int64{99},
		SearchItem:        model.SearchItem{Date: &model.FilterBetween{}},
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(out.Items) != 0 {
		t.Errorf("listed %d agents, want none", len(out.Items))
	}

	if len(out.Holidays) != 1 || out.Calendar == nil {
		t.Errorf("holidays = %d, calendar = %v, want both: the calendar does not depend on the agent filter", len(out.Holidays), out.Calendar)
	}
}

// The shifts, the holidays and the calendar are read in parallel. Run with -race: both reads
// used to write one shared error variable.
func TestSearchAgentsWorkingScheduleParallelReads(t *testing.T) {
	ctx, user := signedIn()
	shifts := &shiftStore{agents: []*model.AgentWorkingSchedule{{}}, holidays: []*model.Holiday{{}}}

	svc := NewAgentWorkingSchedule(shifts, scheduleStore{}, nil)
	for range 20 {
		if _, err := svc.SearchAgentsWorkingSchedule(ctx, user, &model.AgentWorkingScheduleSearch{
			WorkingScheduleID: 7,
			SearchItem:        model.SearchItem{Date: &model.FilterBetween{}},
		}); err != nil {
			t.Fatal(err)
		}
	}
}

// A search without a period, or with one bound missing, covers the
// schedule's own period — as the schedule forecast does.
func TestSearchAgentsWorkingScheduleWithoutPeriod(t *testing.T) {
	from := model.TimestampFromMillis(time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC).UnixMilli())

	for name, tc := range map[string]struct {
		date     *model.FilterBetween
		wantFrom time.Time
	}{
		"no period":    {date: nil, wantFrom: periodStart},
		"unset period": {date: &model.FilterBetween{}, wantFrom: periodStart},
		"only from":    {date: &model.FilterBetween{From: from}, wantFrom: from.Time},
	} {
		t.Run(name, func(t *testing.T) {
			ctx, user := signedIn()
			shifts := &shiftStore{}

			svc := NewAgentWorkingSchedule(shifts, scheduleStore{}, nil)
			if _, err := svc.SearchAgentsWorkingSchedule(ctx, user, &model.AgentWorkingScheduleSearch{
				WorkingScheduleID: 7,
				SearchItem:        model.SearchItem{Date: tc.date},
			}); err != nil {
				t.Fatal(err)
			}

			got := shifts.searched.SearchItem.Date
			wall := func(at time.Time) string { return at.Format(time.DateTime) }

			// pgx sends the wall clock: a bound in the server's zone moves the day west of UTC.
			if got.From.Time.Location() != time.UTC || got.To.Time.Location() != time.UTC ||
				wall(got.From.Time) != wall(tc.wantFrom) || wall(got.To.Time) != wall(periodEnd) {
				t.Errorf("period sent as %s..%s, want %s..%s", wall(got.From.Time), wall(got.To.Time), wall(tc.wantFrom), wall(periodEnd))
			}
		})
	}
}

// The storage reads one agent more than the page; the service tells there is a next
// page by it and drops it, and takes the total from any row.
func TestSearchAgentsWorkingSchedulePage(t *testing.T) {
	ctx, user := signedIn()
	shifts := &shiftStore{agents: []*model.AgentWorkingSchedule{
		{Agent: model.LookupItem{ID: 10}, Total: 5},
		{Agent: model.LookupItem{ID: 11}, Total: 5},
	}}

	out, err := NewAgentWorkingSchedule(shifts, scheduleStore{}, nil).SearchAgentsWorkingSchedule(ctx, user, &model.AgentWorkingScheduleSearch{
		WorkingScheduleID: 7,
		SearchItem:        model.SearchItem{Date: &model.FilterBetween{}, Size: 1},
	})
	if err != nil {
		t.Fatal(err)
	}

	if !out.Next || len(out.Items) != 1 || out.Items[0].Agent.ID != 10 || out.Total != 5 {
		t.Errorf("next = %v, items = %d, total = %d; want a next page, agent 10 alone, 5 in all", out.Next, len(out.Items), out.Total)
	}
}

// templateStore records the template it was asked to write and skips nothing.
type templateStore struct {
	shiftStore

	written *model.WeekTemplate
}

func (s *templateStore) CreateTemplateShifts(_ context.Context, _ *model.SignedInUser, _ int64, tpl *model.WeekTemplate) ([]model.Violation, error) {
	s.written = tpl

	return nil, nil
}

func TestCreateAgentsWorkingScheduleShifts(t *testing.T) {
	ctx, user := signedIn()
	month := func(agents ...int64) *model.WeekTemplate {
		return &model.WeekTemplate{Agents: agents, From: periodStart, To: periodEnd}
	}

	store := &templateStore{}
	svc := NewAgentWorkingSchedule(store, nil, nil)

	if _, _, err := svc.CreateAgentsWorkingScheduleShifts(ctx, user, 7, month(3, 1, 3, 2)); err != nil {
		t.Fatal(err)
	}

	if got := store.written.Agents; len(got) != 3 || got[0] != 1 || got[2] != 3 {
		t.Errorf("agents written %v, want each once", got)
	}

	if s := store.searched; s == nil || s.WorkingScheduleID != 7 || len(s.AgentIDs) != 3 || s.SearchItem.Size < 3 ||
		!s.SearchItem.Date.From.Time.Equal(periodStart) || !s.SearchItem.Date.To.Time.Equal(periodEnd) {
		t.Errorf("cells read back with %+v, want the agents over the template's dates", s)
	}

	agents := make([]int64, 201)
	for i := range agents {
		agents[i] = int64(i + 1)
	}

	store.written = nil

	if _, _, err := svc.CreateAgentsWorkingScheduleShifts(ctx, user, 7, month(agents[:200]...)); err != nil {
		t.Errorf("200 agents over 31 days: %v", err)
	}

	store.written = nil

	if _, _, err := svc.CreateAgentsWorkingScheduleShifts(ctx, user, 7, month(agents...)); !errors.Is(err, ErrRequestTooLarge) || store.written != nil {
		t.Errorf("201 agents over 31 days: %v, written %v; want %v before the storage", err, store.written != nil, ErrRequestTooLarge)
	}

	// Rows, not only agent-days: one agent with two thousand ranges a day is refused too.
	crowded, busy := month(1), month(agents[:200]...)
	crowded.Days, busy.Days = map[time.Weekday][]model.ShiftTemplateTime{}, map[time.Weekday][]model.ShiftTemplateTime{}

	for d := time.Sunday; d <= time.Saturday; d++ {
		crowded.Days[d], busy.Days[d] = make([]model.ShiftTemplateTime, 2000), make([]model.ShiftTemplateTime, 2)
	}

	busy.Pauses = make([]model.ShiftTemplatePause, 6)
	store.written = nil

	if _, _, err := svc.CreateAgentsWorkingScheduleShifts(ctx, user, 7, crowded); !errors.Is(err, ErrRequestTooLarge) || store.written != nil {
		t.Errorf("62000 shifts for one agent: %v, written %v; want %v", err, store.written != nil, ErrRequestTooLarge)
	}

	store.written = nil

	if _, _, err := svc.CreateAgentsWorkingScheduleShifts(ctx, user, 7, busy); err != nil {
		t.Errorf("200 agents a month with two shifts and six pauses a day: %v", err)
	}
}
