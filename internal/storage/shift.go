package storage

import (
	"cmp"
	"context"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/webitel/webitel-go-kit/pkg/errors"

	"github.com/webitel/webitel-wfm/infra/storage/dbsql"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql/builder"
	"github.com/webitel/webitel-wfm/internal/model"
)

var (
	ErrShiftScheduleState = errors.FailedPrecondition("shifts are written to a draft, pending or active schedule only", errors.WithID("storage.shift.schedule_state"))
	ErrShiftsRefused      = errors.FailedPrecondition("the shifts break the schedule rules", errors.WithID("storage.shift.refused"))
)

// CreateShifts writes the shifts with their pauses and skills in one transaction under
// the agents' locks: all of them, or none with ErrShiftsRefused and one violation per
// agent, working date and broken rule. WFM-99 and WFM-103 write through here too.
func (a *AgentWorkingSchedule) CreateShifts(ctx context.Context, user *model.SignedInUser, scheduleID int64, in []*model.Shift) ([]int64, error) {
	if len(in) == 0 {
		return nil, nil
	}

	locks := make([]dbsql.LockKey, 0, len(in))
	for _, shift := range in {
		locks = append(locks, dbsql.OperatorLock(shift.AgentID))
	}

	var ids []int64

	err := a.db.InTx(ctx, func(ctx context.Context, tx dbsql.Tx) error {
		// Agents first, schedule row second: activation (WFM-101) locks in the same order.
		if err := tx.Lock(ctx, locks...); err != nil {
			return err
		}

		schedule, err := lockSchedule(ctx, tx, user.DomainID, scheduleID)
		if err != nil {
			return err
		}

		violations, err := shiftViolations(ctx, tx, schedule, in)
		if err != nil {
			return err
		}

		if len(violations) > 0 {
			return model.Refuse(ErrShiftsRefused, violations...)
		}

		ids, err = insertShifts(ctx, tx, user, schedule, in)

		return err
	})
	if err != nil {
		return nil, err
	}

	return ids, nil
}

// shiftSchedule is the schedule row, held FOR SHARE so its state, dates and calendar
// stay put until the write commits and an activation (WFM-101) waits for it.
type shiftSchedule struct {
	ID        int64
	State     model.WorkingScheduleState `db:"state"`
	StartDate time.Time                  `db:"start_date_at"`
	EndDate   time.Time                  `db:"end_date_at"`
	TimeZone  string                     `db:"time_zone"`

	zone *time.Location
}

// date is the working date of a shift: the day it starts on in the calendar's zone.
func (s *shiftSchedule) date(shift *model.Shift) time.Time {
	y, m, d := shift.Start.In(s.zone).Date()

	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func lockSchedule(ctx context.Context, tx dbsql.Tx, domainID, scheduleID int64) (*shiftSchedule, error) {
	q := builder.Select("ws.state", "ws.start_date_at", "ws.end_date_at", "tz.sys_name AS time_zone").
		From("wfm.working_schedule ws").
		Join("flow.calendar c", "c.id = ws.calendar_id").
		Join("flow.calendar_timezones tz", "tz.id = c.timezone_id")
	sql, args := q.Where(q.Equal("ws.domain_id", domainID), q.Equal("ws.id", scheduleID)).ForShare("ws").Build()

	schedule := shiftSchedule{ID: scheduleID}
	if err := tx.Primary().Get(ctx, &schedule, sql, args...); err != nil {
		return nil, err
	}

	writable := []model.WorkingScheduleState{model.WorkingScheduleStateDraft, model.WorkingScheduleStatePending, model.WorkingScheduleStateActive}
	if !slices.Contains(writable, schedule.State) {
		return nil, errors.Wrap(ErrShiftScheduleState, errors.WithValue("state", schedule.State.String()))
	}

	zone, err := time.LoadLocation(schedule.TimeZone)
	if err != nil {
		return nil, err
	}

	schedule.zone = zone

	return &schedule, nil
}

// existingShift is a shift already written for one of the agents, in this schedule or in an active one.
type existingShift struct {
	AgentID    int64     `db:"agent_id"`
	ScheduleID int64     `db:"schedule_id"`
	Date       time.Time `db:"schedule_at"`
	Start      time.Time `db:"start_at"`
	End        time.Time `db:"end_at"`
}

type agentAbsence struct {
	AgentID int64     `db:"agent_id"`
	Date    time.Time `db:"absent_at"`
}

// shiftViolations lists every rule the request breaks. Shifts are [start, end), so back
// to back is fine. Other schedules count only when active (EP_05).
func shiftViolations(ctx context.Context, tx dbsql.Tx, s *shiftSchedule, in []*model.Shift) ([]model.Violation, error) {
	agents := make([]any, 0, len(in))
	dates := make([]any, 0, len(in))

	for _, shift := range in {
		agents = append(agents, shift.AgentID)
		dates = append(dates, s.date(shift))
	}

	first := slices.MinFunc(in, func(a, b *model.Shift) int { return a.Start.Compare(b.Start) }).Start
	last := slices.MaxFunc(in, func(a, b *model.Shift) int { return a.End.Compare(b.End) }).End

	q := builder.Select("agent_id").From("wfm.working_schedule_agent")
	sql, args := q.Where(q.Equal("working_schedule_id", s.ID), q.In("agent_id", agents...)).Build()

	var members []int64
	if err := tx.Primary().Select(ctx, &members, sql, args...); err != nil {
		return nil, err
	}

	q = builder.Select("agent_id", "absent_at").From("wfm.agent_absence")
	sql, args = q.Where(q.In("agent_id", agents...), q.In("absent_at", dates...)).Build()

	var absences []agentAbsence
	if err := tx.Primary().Select(ctx, &absences, sql, args...); err != nil {
		return nil, err
	}

	q = builder.Select("wsa.agent_id", "wsa.working_schedule_id AS schedule_id", "aws.schedule_at", "aws.start_at", "aws.end_at").
		From("wfm.agent_working_schedule aws").
		Join("wfm.working_schedule_agent wsa", "wsa.id = aws.working_schedule_agent_id").
		Join("wfm.working_schedule ws", "ws.id = wsa.working_schedule_id")
	sql, args = q.Where(
		q.In("wsa.agent_id", agents...),
		q.Or(q.Equal("ws.id", s.ID), q.Equal("ws.state", int16(model.WorkingScheduleStateActive))),
		q.Or(q.In("aws.schedule_at", dates...), q.And(q.LessThan("aws.start_at", last), q.GreaterThan("aws.end_at", first))),
	).Build()

	var existing []existingShift
	if err := tx.Primary().Select(ctx, &existing, sql, args...); err != nil {
		return nil, err
	}

	var out []model.Violation

	for _, shift := range in {
		date := s.date(shift)
		broke := func(rule string) {
			out = append(out, model.Violation{OperatorID: shift.AgentID, Date: pgtype.Date{Time: date, Valid: true}, Reason: "storage.shift." + rule})
		}

		if !slices.Contains(members, shift.AgentID) {
			broke("agent_not_in_schedule")
		}

		if date.Before(s.StartDate) || date.After(s.EndDate) {
			broke("outside_schedule")
		}

		if !shift.Start.Before(shift.End) || shift.End.Sub(shift.Start) > 24*time.Hour {
			broke("duration")
		}

		if slices.ContainsFunc(shift.Pauses, func(p *model.ShiftPause) bool {
			return !p.Start.Before(p.End) || p.Start.Before(shift.Start) || p.End.After(shift.End)
		}) {
			broke("pause_outside")
		}

		if pausesOverlap(shift.Pauses) {
			broke("pause_overlap")
		}

		if slices.ContainsFunc(absences, func(a agentAbsence) bool { return a.AgentID == shift.AgentID && a.Date.Equal(date) }) {
			broke("absence")
		}

		if slices.ContainsFunc(existing, func(e existingShift) bool {
			return e.AgentID == shift.AgentID && e.ScheduleID != s.ID && e.Date.Equal(date)
		}) {
			broke("locked_date")
		}

		written := slices.ContainsFunc(existing, func(e existingShift) bool {
			return e.AgentID == shift.AgentID && overlaps(e.Start, e.End, shift.Start, shift.End)
		})
		requested := slices.ContainsFunc(in, func(o *model.Shift) bool {
			return o != shift && o.AgentID == shift.AgentID && overlaps(o.Start, o.End, shift.Start, shift.End)
		})

		if written || requested {
			broke("overlap")
		}
	}

	// One line per agent, date and rule, however many shifts break it.
	slices.SortFunc(out, func(a, b model.Violation) int {
		return cmp.Or(cmp.Compare(a.OperatorID, b.OperatorID), a.Date.Time.Compare(b.Date.Time), strings.Compare(a.Reason, b.Reason))
	})

	return slices.CompactFunc(out, func(a, b model.Violation) bool {
		return a.OperatorID == b.OperatorID && a.Date.Time.Equal(b.Date.Time) && a.Reason == b.Reason
	}), nil
}

// overlaps says whether [aStart, aEnd) and [bStart, bEnd) meet.
func overlaps(aStart, aEnd, bStart, bEnd time.Time) bool {
	return aStart.Before(bEnd) && aEnd.After(bStart)
}

func pausesOverlap(pauses []*model.ShiftPause) bool {
	for i, a := range pauses {
		for _, b := range pauses[i+1:] {
			if overlaps(a.Start, a.End, b.Start, b.End) {
				return true
			}
		}
	}

	return false
}

// insertShifts queues one statement per shift: the shift, then its pauses and skills off its id.
func insertShifts(ctx context.Context, tx dbsql.Tx, user *model.SignedInUser, s *shiftSchedule, in []*model.Shift) ([]int64, error) {
	batch := tx.Primary().Batch()

	for _, shift := range in {
		columns := []map[string]any{{
			"domain_id":                 user.DomainID,
			"created_by":                user.ID,
			"working_schedule_agent_id": builder.Format("(SELECT id::bigint FROM wfm.working_schedule_agent WHERE working_schedule_id = $? AND agent_id = $?)", s.ID, shift.AgentID),
			"schedule_at":               s.date(shift),
			"start_at":                  shift.Start,
			"end_at":                    shift.End,
		}}

		cte := builder.CTE(builder.With("shift").As(builder.Insert(agentWorkingScheduleTable, columns).SQL("RETURNING id")))

		if l := len(shift.Pauses); l > 0 {
			pauses := make([]map[string]any, 0, l)
			for _, pause := range shift.Pauses {
				pauses = append(pauses, map[string]any{
					"domain_id":                 user.DomainID,
					"created_by":                user.ID,
					"agent_working_schedule_id": builder.Format("(SELECT id::bigint FROM shift)"),
					"pause_cause_id":            pause.Cause.SafeID(),
					"start_at":                  pause.Start,
					"end_at":                    pause.End,
				})
			}

			cte.With(builder.With("pauses").As(builder.Insert(agentWorkingSchedulePauseTable, pauses)))
		}

		if l := len(shift.Skills); l > 0 {
			skills := make([]map[string]any, 0, l)
			for _, skill := range shift.Skills {
				skills = append(skills, map[string]any{
					"domain_id":                 user.DomainID,
					"agent_working_schedule_id": builder.Format("(SELECT id::bigint FROM shift)"),
					"skill_id":                  skill.Skill.ID,
					"capacity":                  skill.Capacity,
					"enabled":                   skill.Enabled,
				})
			}

			cte.With(builder.With("skills").As(builder.Insert(agentWorkingScheduleSkillTable, skills)))
		}

		sql, args := builder.Select("shift.id").With(cte.Builder()).From("shift").Build()
		batch.Queue(sql, args...)
	}

	var ids []int64
	if err := batch.Select(ctx, &ids); err != nil {
		return nil, err
	}

	return ids, nil
}
