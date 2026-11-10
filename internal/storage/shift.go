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
	ErrShiftDuration       = errors.InvalidArgument("invalid input: a shift should end after it starts, within 24 hours", errors.WithID("storage.shift.duration"))
	ErrShiftPauseOutside   = errors.InvalidArgument("invalid input: a pause should end after it starts, inside its shift", errors.WithID("storage.shift.pause_outside"))
	ErrShiftPauseOverlap   = errors.InvalidArgument("invalid input: the pauses of a shift should not overlap", errors.WithID("storage.shift.pause_overlap"))
	ErrShiftRequestOverlap = errors.InvalidArgument("invalid input: the shifts of an agent should not overlap", errors.WithID("storage.shift.request_overlap"))

	ErrShiftScheduleState = errors.FailedPrecondition("shifts are written to a draft, pending or active schedule only", errors.WithID("storage.shift.schedule_state"))
	ErrShiftsRefused      = errors.FailedPrecondition("the shifts break the schedule rules", errors.WithID("storage.shift.refused"))
)

// CreateShifts writes the shifts with their pauses and skills in one transaction under
// the agents' locks. A request wrong on its own is invalid before any lock is taken; one
// that breaks the schedule rules is refused whole with ErrShiftsRefused and one violation
// per agent, working date and rule. WFM-99 and WFM-103 write through here too.
func (a *AgentWorkingSchedule) CreateShifts(ctx context.Context, user *model.SignedInUser, scheduleID int64, in []*model.Shift) ([]int64, error) {
	if len(in) == 0 {
		return nil, nil
	}

	if err := validateShifts(in); err != nil {
		return nil, err
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

// validateShifts checks the request on its own: every shift ends after it starts and
// within 24 hours, its pauses lie inside it and apart, and the shifts of an agent do not
// overlap. Intervals are [start, end), so back to back is fine, and sorted by start they
// overlap only if two neighbors do.
func validateShifts(in []*model.Shift) error {
	for _, shift := range in {
		if !shift.Start.Before(shift.End) || shift.End.Sub(shift.Start) > 24*time.Hour {
			return invalidShift(ErrShiftDuration, shift)
		}

		pauses := slices.SortedFunc(slices.Values(shift.Pauses), func(a, b *model.ShiftPause) int { return a.Start.Compare(b.Start) })
		for i, pause := range pauses {
			if !pause.Start.Before(pause.End) || pause.Start.Before(shift.Start) || pause.End.After(shift.End) {
				return invalidShift(ErrShiftPauseOutside, shift)
			}

			if i > 0 && pause.Start.Before(pauses[i-1].End) {
				return invalidShift(ErrShiftPauseOverlap, shift)
			}
		}
	}

	byAgent := slices.SortedFunc(slices.Values(in), func(a, b *model.Shift) int {
		return cmp.Or(cmp.Compare(a.AgentID, b.AgentID), a.Start.Compare(b.Start))
	})
	for i := 1; i < len(byAgent); i++ {
		if byAgent[i].AgentID == byAgent[i-1].AgentID && byAgent[i].Start.Before(byAgent[i-1].End) {
			return invalidShift(ErrShiftRequestOverlap, byAgent[i])
		}
	}

	return nil
}

func invalidShift(err error, shift *model.Shift) error {
	return errors.Wrap(err, errors.WithValue("agent_id", shift.AgentID), errors.WithValue("start_at", shift.Start))
}

// shiftFactsSQL looks up, for each requested shift in order, what only the database
// knows: whether the agent is in schedule $1, absent on the working date, working that
// date in another active schedule ($6 is the active state), or already working at that
// time in this schedule or an active one. Shifts are [start, end): back to back is fine.
const shiftFactsSQL = `
SELECT EXISTS (SELECT 1
               FROM wfm.working_schedule_agent wsa
               WHERE wsa.working_schedule_id = $1
                 AND wsa.agent_id = r.agent_id) AS member,
       EXISTS (SELECT 1
               FROM wfm.agent_absence a
               WHERE a.agent_id = r.agent_id
                 AND a.absent_at = r.schedule_at) AS absent,
       EXISTS (SELECT 1
               FROM wfm.agent_working_schedule aws
                        INNER JOIN wfm.working_schedule_agent wsa ON wsa.id = aws.working_schedule_agent_id
                        INNER JOIN wfm.working_schedule ws ON ws.id = wsa.working_schedule_id
               WHERE wsa.agent_id = r.agent_id
                 AND ws.id <> $1
                 AND ws.state = $6
                 AND aws.schedule_at = r.schedule_at) AS date_taken,
       EXISTS (SELECT 1
               FROM wfm.agent_working_schedule aws
                        INNER JOIN wfm.working_schedule_agent wsa ON wsa.id = aws.working_schedule_agent_id
                        INNER JOIN wfm.working_schedule ws ON ws.id = wsa.working_schedule_id
               WHERE wsa.agent_id = r.agent_id
                 AND (ws.id = $1 OR ws.state = $6)
                 AND aws.start_at < r.end_at
                 AND aws.end_at > r.start_at) AS overlaps
FROM unnest($2::bigint[], $3::date[], $4::timestamptz[], $5::timestamptz[])
         WITH ORDINALITY AS r(agent_id, schedule_at, start_at, end_at, n)
ORDER BY r.n`

type shiftFacts struct {
	Member    bool `db:"member"`
	Absent    bool `db:"absent"`
	DateTaken bool `db:"date_taken"`
	Overlaps  bool `db:"overlaps"`
}

// shiftViolations lists every schedule rule the request breaks: the database finds the
// facts in one query, the rules turn them into reasons here. Other schedules count only
// when active (EP_05).
func shiftViolations(ctx context.Context, tx dbsql.Tx, s *shiftSchedule, in []*model.Shift) ([]model.Violation, error) {
	agents := make([]int64, 0, len(in))
	dates := make([]time.Time, 0, len(in))
	starts := make([]time.Time, 0, len(in))
	ends := make([]time.Time, 0, len(in))

	for _, shift := range in {
		agents = append(agents, shift.AgentID)
		dates = append(dates, s.date(shift))
		starts = append(starts, shift.Start)
		ends = append(ends, shift.End)
	}

	var facts []shiftFacts
	if err := tx.Primary().Select(ctx, &facts, shiftFactsSQL, s.ID, agents, dates, starts, ends, int16(model.WorkingScheduleStateActive)); err != nil {
		return nil, err
	}

	var out []model.Violation

	for i, f := range facts {
		broke := func(rule string) {
			out = append(out, model.Violation{OperatorID: agents[i], Date: pgtype.Date{Time: dates[i], Valid: true}, Reason: "storage.shift." + rule})
		}

		if !f.Member {
			broke("agent_not_in_schedule")
		}

		if dates[i].Before(s.StartDate) || dates[i].After(s.EndDate) {
			broke("outside_schedule")
		}

		if f.Absent {
			broke("absence")
		}

		if f.DateTaken {
			broke("locked_date")
		}

		if f.Overlaps {
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
