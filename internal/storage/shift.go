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

type ShiftManager interface {
	// CreateShifts writes the shifts into the schedule in one transaction under the
	// agents' locks and returns their ids in the order given. It writes all or none:
	// a broken rule refuses with ErrShiftsRefused and one violation per agent, working
	// date and reason (storage.shift.<rule>).
	CreateShifts(ctx context.Context, user *model.SignedInUser, scheduleID int64, in []*model.Shift) ([]int64, error)
}

type Shift struct {
	db dbsql.Store
}

func NewShift(db dbsql.Store) *Shift {
	return &Shift{db: db}
}

func (s *Shift) CreateShifts(ctx context.Context, user *model.SignedInUser, scheduleID int64, in []*model.Shift) ([]int64, error) {
	if len(in) == 0 {
		return nil, nil
	}

	locks := make([]dbsql.LockKey, 0, len(in))
	for _, shift := range in {
		locks = append(locks, dbsql.OperatorLock(shift.AgentID))
	}

	var ids []int64

	err := s.db.InTx(ctx, func(ctx context.Context, tx dbsql.Tx) error {
		// The agents first, the schedule row second; activation (WFM-101) locks in the same order.
		if err := tx.Lock(ctx, locks...); err != nil {
			return err
		}

		schedule, err := s.lockSchedule(ctx, tx, user.DomainID, scheduleID)
		if err != nil {
			return err
		}

		// The working date is the day the shift starts on in the calendar's zone.
		zone, err := time.LoadLocation(schedule.TimeZone)
		if err != nil {
			return err
		}

		dates := make([]time.Time, 0, len(in))
		for _, shift := range in {
			y, m, d := shift.Start.In(zone).Date()
			dates = append(dates, time.Date(y, m, d, 0, 0, 0, 0, time.UTC))
		}

		violations, err := s.check(ctx, tx, scheduleID, schedule, in, dates)
		if err != nil {
			return err
		}

		if len(violations) > 0 {
			return model.Refuse(ErrShiftsRefused, violations...)
		}

		ids, err = s.insert(ctx, tx, user, scheduleID, in, dates)

		return err
	})
	if err != nil {
		return nil, err
	}

	return ids, nil
}

type shiftSchedule struct {
	State     model.WorkingScheduleState `db:"state"`
	StartDate time.Time                  `db:"start_date_at"`
	EndDate   time.Time                  `db:"end_date_at"`
	TimeZone  string                     `db:"time_zone"`
}

// lockSchedule reads the schedule FOR SHARE, so its state, dates and calendar hold
// until the write commits and an activation (WFM-101) or an edit (WFM-100) waits for it.
func (s *Shift) lockSchedule(ctx context.Context, tx dbsql.Tx, domainID, scheduleID int64) (*shiftSchedule, error) {
	sb := builder.Select("ws.state", "ws.start_date_at", "ws.end_date_at", "tz.sys_name AS time_zone").
		From("wfm.working_schedule ws").
		Join("flow.calendar c", "c.id = ws.calendar_id").
		Join("flow.calendar_timezones tz", "tz.id = c.timezone_id")
	sql, args := sb.Where(sb.Equal("ws.domain_id", domainID), sb.Equal("ws.id", scheduleID)).ForShare("ws").Build()

	var schedule shiftSchedule
	if err := tx.Primary().Get(ctx, &schedule, sql, args...); err != nil {
		if errors.Is(err, dbsql.ErrNoRows) {
			return nil, errors.Wrap(err, errors.WithID("storage.shift.schedule"))
		}

		return nil, err
	}

	writable := []model.WorkingScheduleState{model.WorkingScheduleStateDraft, model.WorkingScheduleStatePending, model.WorkingScheduleStateActive}
	if !slices.Contains(writable, schedule.State) {
		return nil, errors.Wrap(ErrShiftScheduleState, errors.WithValue("state", schedule.State.String()))
	}

	return &schedule, nil
}

// existingShift is a shift already written for one of the agents, in this schedule
// or in an active one: drafts and pending schedules of other teams block nothing.
type existingShift struct {
	AgentID    int64     `db:"agent_id"`
	ScheduleID int64     `db:"schedule_id"`
	Date       time.Time `db:"schedule_at"`
	Start      time.Time `db:"start_at"`
	End        time.Time `db:"end_at"`
}

type absence struct {
	AgentID int64     `db:"agent_id"`
	Date    time.Time `db:"absent_at"`
}

// check reads what the rules need under the locks and lists every broken rule.
func (s *Shift) check(ctx context.Context, tx dbsql.Tx, scheduleID int64, schedule *shiftSchedule, in []*model.Shift, dates []time.Time) ([]model.Violation, error) {
	agents := make([]any, 0, len(in))
	for _, shift := range in {
		agents = append(agents, shift.AgentID)
	}

	members := builder.Select("agent_id").From("wfm.working_schedule_agent")
	members.Where(members.Equal("working_schedule_id", scheduleID), members.In("agent_id", agents...))

	var inSchedule []int64
	if err := selectInto(ctx, tx, &inSchedule, members); err != nil {
		return nil, err
	}

	absent := builder.Select("agent_id", "absent_at").From("wfm.agent_absence")
	absent.Where(absent.In("agent_id", agents...))

	var absences []absence
	if err := selectInto(ctx, tx, &absences, absent); err != nil {
		return nil, err
	}

	written := builder.Select("wsa.agent_id", "wsa.working_schedule_id AS schedule_id", "aws.schedule_at", "aws.start_at", "aws.end_at").
		From("wfm.agent_working_schedule aws").
		Join("wfm.working_schedule_agent wsa", "wsa.id = aws.working_schedule_agent_id").
		Join("wfm.working_schedule ws", "ws.id = wsa.working_schedule_id")
	written.Where(written.In("wsa.agent_id", agents...),
		written.Or(written.Equal("ws.id", scheduleID), written.Equal("ws.state", int16(model.WorkingScheduleStateActive))))

	var existing []existingShift
	if err := selectInto(ctx, tx, &existing, written); err != nil {
		return nil, err
	}

	var violations []model.Violation

	for i, shift := range in {
		date := dates[i]
		refuse := func(reason string) {
			violations = append(violations, model.Violation{OperatorID: shift.AgentID, Date: pgtype.Date{Time: date, Valid: true}, Reason: "storage.shift." + reason})
		}

		if !slices.Contains(inSchedule, shift.AgentID) {
			refuse("agent_not_in_schedule")
		}

		if date.Before(schedule.StartDate) || date.After(schedule.EndDate) {
			refuse("outside_schedule")
		}

		if !shift.Start.Before(shift.End) || shift.End.Sub(shift.Start) > 24*time.Hour {
			refuse("duration")
		}

		if slices.ContainsFunc(shift.Pauses, func(p *model.ShiftPause) bool {
			return !p.Start.Before(p.End) || p.Start.Before(shift.Start) || p.End.After(shift.End)
		}) {
			refuse("pause_outside")
		}

		if pausesOverlap(shift.Pauses) {
			refuse("pause_overlap")
		}

		if slices.ContainsFunc(absences, func(a absence) bool { return a.AgentID == shift.AgentID && a.Date.Equal(date) }) {
			refuse("absence")
		}

		// A date the agent works in another active schedule is taken even with no overlap in time (EP_05).
		if slices.ContainsFunc(existing, func(e existingShift) bool {
			return e.AgentID == shift.AgentID && e.ScheduleID != scheduleID && e.Date.Equal(date)
		}) {
			refuse("locked_date")
		}

		overlapsWritten := slices.ContainsFunc(existing, func(e existingShift) bool {
			return e.AgentID == shift.AgentID && overlaps(e.Start, e.End, shift.Start, shift.End)
		})
		overlapsRequested := slices.ContainsFunc(in, func(o *model.Shift) bool {
			return o != shift && o.AgentID == shift.AgentID && overlaps(o.Start, o.End, shift.Start, shift.End)
		})

		if overlapsWritten || overlapsRequested {
			refuse("overlap")
		}
	}

	slices.SortFunc(violations, func(a, b model.Violation) int {
		return cmp.Or(cmp.Compare(a.OperatorID, b.OperatorID), a.Date.Time.Compare(b.Date.Time), strings.Compare(a.Reason, b.Reason))
	})

	// Two shifts of one agent on one date that break the same rule are one violation.
	return slices.CompactFunc(violations, func(a, b model.Violation) bool {
		return a.OperatorID == b.OperatorID && a.Date.Time.Equal(b.Date.Time) && a.Reason == b.Reason
	}), nil
}

func selectInto(ctx context.Context, tx dbsql.Tx, dest any, sb interface{ Build() (string, []any) }) error {
	sql, args := sb.Build()

	return tx.Primary().Select(ctx, dest, sql, args...)
}

// overlaps says whether [aStart, aEnd) and [bStart, bEnd) meet: back to back is fine.
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

// insert queues one statement per shift: the shift, then its pauses and skills off
// the returned id. The rules are checked already, so a valid request fails no constraint.
func (s *Shift) insert(ctx context.Context, tx dbsql.Tx, user *model.SignedInUser, scheduleID int64, in []*model.Shift, dates []time.Time) ([]int64, error) {
	batch := tx.Primary().Batch()

	for i, shift := range in {
		columns := []map[string]any{{
			"domain_id":                 user.DomainID,
			"created_by":                user.ID,
			"working_schedule_agent_id": builder.Format("(SELECT id::bigint FROM wfm.working_schedule_agent WHERE working_schedule_id = $? AND agent_id = $?)", scheduleID, shift.AgentID),
			"schedule_at":               dates[i],
			"start_at":                  shift.Start,
			"end_at":                    shift.End,
			"source":                    int16(model.ShiftSourceManual),
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
