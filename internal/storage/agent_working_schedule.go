package storage

import (
	"cmp"
	"context"
	"slices"
	"time"

	"github.com/webitel/webitel-wfm/infra/storage/dbsql"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql/builder"
	"github.com/webitel/webitel-wfm/internal/model"
)

const (
	agentWorkingScheduleTable      = "wfm.agent_working_schedule"
	agentWorkingSchedulePauseTable = agentWorkingScheduleTable + "_pause"
	agentWorkingScheduleSkillTable = agentWorkingScheduleTable + "_skill"

	agentWorkingScheduleView         = agentWorkingScheduleTable + "_v"
	agentWorkingScheduleHolidaysView = agentWorkingScheduleTable + "_holidays_v"
)

type AgentWorkingScheduleManager interface {
	CreateShifts(ctx context.Context, user *model.SignedInUser, scheduleID int64, in []*model.Shift) ([]int64, error)
	SearchAgentWorkingSchedule(ctx context.Context, user *model.SignedInUser, search *model.AgentWorkingScheduleSearch) ([]*model.AgentWorkingSchedule, error)
	Holidays(ctx context.Context, user *model.SignedInUser, search *model.AgentWorkingScheduleSearch) ([]*model.Holiday, error)
	ScheduleCalendar(ctx context.Context, user *model.SignedInUser, scheduleID int64) (*model.ScheduleCalendar, error)
}

type AgentWorkingSchedule struct {
	db dbsql.Store
}

func NewAgentWorkingSchedule(db dbsql.Store) *AgentWorkingSchedule {
	return &AgentWorkingSchedule{
		db: db,
	}
}

func (a *AgentWorkingSchedule) SearchAgentWorkingSchedule(ctx context.Context, user *model.SignedInUser, search *model.AgentWorkingScheduleSearch) ([]*model.AgentWorkingSchedule, error) {
	sb := builder.Select()

	// The period narrows the cells aggregated for each agent, not which agents
	// are listed: an agent with nothing in the period still comes back, empty.
	cells := []string{sb.IsNotNull("date")}

	if s := search.SearchItem.Date; s != nil {
		if s.From.Valid {
			cells = append(cells, sb.GreaterEqualThan("date", s.From))
		}

		if s.To.Valid {
			cells = append(cells, sb.LessEqualThan("date", s.To))
		}
	}

	sb.Select("agent AS agent", "jsonb_agg(schedule.* ORDER BY date) FILTER (WHERE "+sb.And(cells...)+") AS schedule")

	if len(search.AgentIDs) > 0 {
		in := make([]any, 0, len(search.AgentIDs))
		for _, id := range search.AgentIDs {
			in = append(in, id)
		}

		sb.Where(sb.In("(agent ->> 'id')::bigint", in...))
	}

	if search.SearchItem.Search != nil {
		sb.Where(sb.ILike("(agent ->> 'name')::text", search.SearchItem.SearchBy()))
	}

	sql, args := sb.From(agentWorkingScheduleView+" AS schedule").
		Where(sb.Equal("domain_id", user.DomainID),
			sb.Equal("working_schedule_id", search.WorkingScheduleID),
		).
		GroupBy("agent").
		Build()

	// The calendar is reread after every edit; a standby may not have replicated it yet.
	var items []*model.AgentWorkingSchedule
	if err := a.db.Primary().Select(ctx, &items, sql, args...); err != nil {
		return nil, err
	}

	return items, nil
}

func (a *AgentWorkingSchedule) Holidays(ctx context.Context, user *model.SignedInUser, search *model.AgentWorkingScheduleSearch) ([]*model.Holiday, error) {
	sb := builder.Select("date", "name").From(agentWorkingScheduleHolidaysView)
	sql, args := sb.Where(sb.Equal("domain_id", user.DomainID),
		sb.Equal("working_schedule_id", search.WorkingScheduleID),
		sb.Between("date", search.SearchItem.Date.From, search.SearchItem.Date.To)).Build()

	// The view follows the schedule's calendar and dates, so it is read from the
	// same node as the shifts it is returned with.
	var items []*model.Holiday
	if err := a.db.Primary().Select(ctx, &items, sql, args...); err != nil {
		return nil, err
	}

	return items, nil
}

// calendarAccept is a range of flow.calendar.accepts, where day 0 is Monday.
type calendarAccept struct {
	Disabled bool  `json:"disabled"`
	Day      int   `json:"day"`
	Start    int16 `json:"start_time_of_day"`
	End      int16 `json:"end_time_of_day"`
	Special  bool  `json:"special"`
}

func (a *AgentWorkingSchedule) ScheduleCalendar(ctx context.Context, user *model.SignedInUser, scheduleID int64) (*model.ScheduleCalendar, error) {
	q := builder.Select("tz.sys_name AS time_zone", "to_jsonb(c.accepts) AS accepts").
		From(builder.WorkingScheduleTable.Name()+" ws").
		Join("flow.calendar c", "c.id = ws.calendar_id").
		Join("flow.calendar_timezones tz", "tz.id = c.timezone_id")
	sql, args := q.Where(q.Equal("ws.domain_id", user.DomainID), q.Equal("ws.id", scheduleID)).Build()

	var row struct {
		TimeZone string           `db:"time_zone"`
		Accepts  []calendarAccept `db:"accepts"`
	}

	if err := a.db.Primary().Get(ctx, &row, sql, args...); err != nil {
		return nil, err
	}

	out := &model.ScheduleCalendar{TimeZone: row.TimeZone}
	for day := range out.Week {
		out.Week[day].Day = time.Weekday(day)
	}

	for _, accept := range row.Accepts {
		if accept.Disabled || accept.Special {
			continue
		}

		day := &out.Week[(accept.Day+1)%7]
		day.Ranges = append(day.Ranges, model.CalendarRange{Start: accept.Start, End: accept.End})
	}

	for day := range out.Week {
		slices.SortFunc(out.Week[day].Ranges, func(a, b model.CalendarRange) int { return cmp.Compare(a.Start, b.Start) })
	}

	return out, nil
}
