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
	ErrShiftsRefused        = errors.FailedPrecondition("the shifts break the schedule rules", errors.WithID("storage.shift.refused"))
	ErrShiftVersionConflict = errors.Aborted("the shift was changed concurrently", errors.WithID("storage.shift.version_conflict"))
)

// The reasons of the violations in ErrShiftsRefused: the front tells refusals apart by them.
const (
	ReasonShiftAgentNotInSchedule = "storage.shift.agent_not_in_schedule"
	ReasonShiftAgentDeactivated   = "storage.shift.agent_deactivated"
	ReasonShiftOutsideSchedule    = "storage.shift.outside_schedule"
	ReasonShiftAbsence            = "storage.shift.absence"
	ReasonShiftLockedDate         = "storage.shift.locked_date"
	ReasonShiftOverlap            = "storage.shift.overlap"
)

func (a *AgentWorkingSchedule) CreateShifts(ctx context.Context, user *model.SignedInUser, scheduleID int64, in []*model.Shift) ([]int64, error) {
	if len(in) == 0 {
		return nil, nil
	}

	if err := model.ValidateShifts(in); err != nil {
		return nil, err
	}

	agents := make([]int64, 0, len(in))
	for _, shift := range in {
		agents = append(agents, shift.AgentID)
	}

	var ids []int64

	err := a.db.InTx(ctx, func(ctx context.Context, tx dbsql.Tx) error {
		schedule, err := lockForShifts(ctx, tx, user.DomainID, scheduleID, agents)
		if err != nil {
			return err
		}

		violations, err := shiftViolations(ctx, tx, schedule, in, 0)
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

// CreateTemplateShifts writes a weekly template for several agents. A day with an absence or a date
// another active schedule holds is skipped and returned; any other broken rule refuses the request.
func (a *AgentWorkingSchedule) CreateTemplateShifts(ctx context.Context, user *model.SignedInUser, scheduleID int64, tpl *model.WeekTemplate) ([]model.Violation, error) {
	var skipped []model.Violation

	err := a.db.InTx(ctx, func(ctx context.Context, tx dbsql.Tx) error {
		schedule, err := lockForShifts(ctx, tx, user.DomainID, scheduleID, tpl.Agents)
		if err != nil {
			return err
		}

		in, err := tpl.Shifts(schedule.zone)
		if err != nil {
			return err
		}

		if err := model.ValidateShifts(in); err != nil {
			return err
		}

		violations, err := shiftViolations(ctx, tx, schedule, in, 0)
		if err != nil {
			return err
		}

		type day struct {
			agent int64
			date  time.Time
		}

		off := make(map[day]bool)

		for _, v := range violations {
			if v.Reason == ReasonShiftAbsence || v.Reason == ReasonShiftLockedDate {
				off[day{v.AgentID, v.Date.Time}] = true
				skipped = append(skipped, v)
			}
		}

		refused := slices.DeleteFunc(violations, func(v model.Violation) bool { return off[day{v.AgentID, v.Date.Time}] })
		if len(refused) > 0 {
			return model.Refuse(ErrShiftsRefused, refused...)
		}

		in = slices.DeleteFunc(in, func(s *model.Shift) bool { return off[day{s.AgentID, schedule.date(s)}] })
		if len(in) == 0 {
			return nil
		}

		_, err = insertShifts(ctx, tx, user, schedule, in)

		return err
	})
	if err != nil {
		return nil, err
	}

	return skipped, nil
}

// lockForShifts takes the agents' locks, then the schedule's row, the order activation locks in, and refuses an archived schedule.
func lockForShifts(ctx context.Context, tx dbsql.Tx, domainID, scheduleID int64, agents []int64) (*shiftSchedule, error) {
	locks := make([]dbsql.LockKey, 0, len(agents))
	for _, agent := range agents {
		locks = append(locks, dbsql.OperatorLock(agent))
	}

	if err := tx.Lock(ctx, locks...); err != nil {
		return nil, err
	}

	schedule, err := lockSchedule(ctx, tx, domainID, scheduleID, lockShare)
	if err != nil {
		return nil, err
	}

	if !schedule.State.ShiftsWritable() {
		return nil, errors.Wrap(ErrWorkingScheduleArchived, errors.WithValue("state", schedule.State.String()))
	}

	return schedule, nil
}

// UpdateShift replaces a shift whole, its pauses and skills included, under the rules of CreateShifts.
func (a *AgentWorkingSchedule) UpdateShift(ctx context.Context, user *model.SignedInUser, scheduleID, id int64, ver int32, in *model.Shift) (*model.AgentScheduleShift, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}

	var out *model.AgentScheduleShift

	err := a.db.InTx(ctx, func(ctx context.Context, tx dbsql.Tx) error {
		schedule, agent, err := lockShift(ctx, tx, user.DomainID, scheduleID, id, ver)
		if err != nil {
			return err
		}

		in.AgentID = agent

		violations, err := shiftViolations(ctx, tx, schedule, []*model.Shift{in}, id)
		if err != nil {
			return err
		}

		if len(violations) > 0 {
			return model.Refuse(ErrShiftsRefused, violations...)
		}

		for _, table := range []string{agentWorkingSchedulePauseTable, agentWorkingScheduleSkillTable} {
			if err := tx.Primary().Exec(ctx, "DELETE FROM "+table+" WHERE domain_id = $1 AND agent_working_schedule_id = $2", user.DomainID, id); err != nil {
				return err
			}
		}

		upd := builder.Update(agentWorkingScheduleTable, map[string]any{
			"updated_by":  user.ID,
			"updated_at":  builder.Format("now()"),
			"schedule_at": schedule.date(in),
			"start_at":    in.Start,
			"end_at":      in.End,
			"ver":         builder.Format("ver + 1"),
		})
		upd.Where(upd.Equal("domain_id", user.DomainID), upd.Equal("id", id)).SQL("RETURNING id")

		cte := withChildren(builder.CTE(builder.With("shift").As(upd)), user, in)
		sql, args := builder.Select("shift.id").With(cte.Builder()).From("shift").Build()

		if err := tx.Primary().Exec(ctx, sql, args...); err != nil {
			return err
		}

		// The calendar's shape: the view builds the shift with its lookups.
		var shifts []*model.AgentScheduleShift
		if err := tx.Primary().Get(ctx, &shifts, `SELECT shifts
FROM wfm.agent_working_schedule_v
WHERE domain_id = $1
  AND working_schedule_id = $2
  AND (agent ->> 'id')::bigint = $3
  AND date = $4`, user.DomainID, scheduleID, agent, schedule.date(in)); err != nil {
			return err
		}

		i := slices.IndexFunc(shifts, func(s *model.AgentScheduleShift) bool { return s.ID == id })
		if i < 0 {
			return dbsql.ErrNoRows
		}

		out = shifts[i]

		return nil
	})
	if err != nil {
		return nil, err
	}

	return out, nil
}

// DeleteShift deletes a shift; its pauses and skills go through the foreign key.
func (a *AgentWorkingSchedule) DeleteShift(ctx context.Context, user *model.SignedInUser, scheduleID, id int64, ver int32) (int64, error) {
	err := a.db.InTx(ctx, func(ctx context.Context, tx dbsql.Tx) error {
		if _, _, err := lockShift(ctx, tx, user.DomainID, scheduleID, id, ver); err != nil {
			return err
		}

		return tx.Primary().Exec(ctx, "DELETE FROM "+agentWorkingScheduleTable+" WHERE domain_id = $1 AND id = $2", user.DomainID, id)
	})
	if err != nil {
		return 0, err
	}

	return id, nil
}

// lockShift locks as CreateShifts does, then the shift's row, and refuses another version or an archived schedule.
func lockShift(ctx context.Context, tx dbsql.Tx, domainID, scheduleID, id int64, ver int32) (*shiftSchedule, int64, error) {
	var agent int64
	if err := tx.Primary().Get(ctx, &agent, `SELECT wsa.agent_id
FROM wfm.agent_working_schedule aws
         INNER JOIN wfm.working_schedule_agent wsa ON wsa.id = aws.working_schedule_agent_id
WHERE aws.domain_id = $1
  AND wsa.working_schedule_id = $2
  AND aws.id = $3`, domainID, scheduleID, id); err != nil {
		return nil, 0, err
	}

	if err := tx.Lock(ctx, dbsql.OperatorLock(agent)); err != nil {
		return nil, 0, err
	}

	schedule, err := lockSchedule(ctx, tx, domainID, scheduleID, lockShare)
	if err != nil {
		return nil, 0, err
	}

	// Gone while this waited for the locks: no_rows.
	var current int32
	if err := tx.Primary().Get(ctx, &current, "SELECT ver FROM "+agentWorkingScheduleTable+" WHERE domain_id = $1 AND id = $2 FOR UPDATE", domainID, id); err != nil {
		return nil, 0, err
	}

	if current != ver {
		return nil, 0, ErrShiftVersionConflict
	}

	if !schedule.State.ShiftsWritable() {
		return nil, 0, ErrWorkingScheduleArchived
	}

	return schedule, agent, nil
}

type shiftSchedule struct {
	ID        int64
	DomainID  int64
	State     model.WorkingScheduleState `db:"state"`
	Ver       int32                      `db:"ver"`
	Team      int64                      `db:"team_id"`
	Calendar  int64                      `db:"calendar_id"`
	StartDate time.Time                  `db:"start_date_at"`
	EndDate   time.Time                  `db:"end_date_at"`
	TimeZone  string                     `db:"time_zone"`

	zone *time.Location
}

// date is the working date: the day the shift starts on in the calendar's zone.
func (s *shiftSchedule) date(shift *model.Shift) time.Time {
	y, m, d := shift.Start.In(s.zone).Date()

	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// rowLock is how a write locks the schedule row, so a state change waits for the write's transaction.
type rowLock string

const (
	// lockShare: shift and agent writes run side by side.
	lockShare rowLock = "SHARE"
	// lockNoKeyUpdate: deleting shifts waits for shift writes, not for inserts that only reference the schedule.
	lockNoKeyUpdate rowLock = "NO KEY UPDATE"
	// lockUpdate: a header update or a delete; two deletes sharing the row would deadlock on the DELETE.
	lockUpdate rowLock = "UPDATE"
)

// lockSchedule reads the schedule under a row lock; the caller checks its state.
func lockSchedule(ctx context.Context, tx dbsql.Tx, domainID, scheduleID int64, lock rowLock) (*shiftSchedule, error) {
	q := builder.Select("ws.state", "ws.ver", "ws.team_id", "ws.calendar_id", "ws.start_date_at", "ws.end_date_at", "tz.sys_name AS time_zone").
		From("wfm.working_schedule ws").
		Join("flow.calendar c", "c.id = ws.calendar_id").
		Join("flow.calendar_timezones tz", "tz.id = c.timezone_id")
	sql, args := q.Where(q.Equal("ws.domain_id", domainID), q.Equal("ws.id", scheduleID)).SQL("FOR " + string(lock) + " OF ws").Build()

	schedule := shiftSchedule{ID: scheduleID, DomainID: domainID}
	if err := tx.Primary().Get(ctx, &schedule, sql, args...); err != nil {
		return nil, err
	}

	zone, err := time.LoadLocation(schedule.TimeZone)
	if err != nil {
		return nil, err
	}

	schedule.zone = zone

	return &schedule, nil
}

const deleteShiftsSQL = `
WITH deleted AS (
    DELETE
    FROM wfm.agent_working_schedule aws
        USING wfm.working_schedule_agent wsa
    WHERE wsa.id = aws.working_schedule_agent_id
      AND wsa.working_schedule_id = $1
      AND aws.schedule_at BETWEEN $2::date AND $3::date
    RETURNING aws.id)
SELECT id
FROM deleted
ORDER BY id`

// deleteShifts deletes the schedule's shifts by working date; removing an agent takes theirs through the foreign key.
// A caller that also takes agents' locks takes them first, as CreateShifts does.
func deleteShifts(ctx context.Context, tx dbsql.Tx, domainID, scheduleID int64, from, to time.Time) ([]int64, error) {
	schedule, err := lockSchedule(ctx, tx, domainID, scheduleID, lockNoKeyUpdate)
	if err != nil {
		return nil, err
	}

	if !schedule.State.ShiftsWritable() {
		return nil, errors.Wrap(ErrWorkingScheduleArchived, errors.WithValue("state", schedule.State.String()))
	}

	var ids []int64
	if err := tx.Primary().Select(ctx, &ids, deleteShiftsSQL, scheduleID, from, to); err != nil {
		return nil, err
	}

	return ids, nil
}

// Other schedules count only when active (EP_05).
const shiftFactsSQL = `
SELECT EXISTS (SELECT 1
               FROM wfm.working_schedule_agent wsa
               WHERE wsa.working_schedule_id = $1
                 AND wsa.agent_id = r.agent_id) AS member,
       EXISTS (SELECT 1
               FROM call_center.cc_agent ca
                        INNER JOIN directory.wbt_user u ON u.id = ca.user_id
               WHERE ca.domain_id = $7
                 AND ca.id = r.agent_id
                 AND u.deleted_at IS NOT NULL) AS deactivated,
       EXISTS (SELECT 1
               FROM wfm.agent_absence a
               WHERE a.domain_id = $7
                 AND a.agent_id = r.agent_id
                 AND a.absent_at = r.schedule_at) AS absent,
       taken.id                       AS taken_id,
       taken.name                     AS taken_name,
       overlap.id                     AS overlap_id,
       overlap.start_at               AS overlap_start,
       overlap.end_at                 AS overlap_end
FROM unnest($2::bigint[], $3::date[], $4::timestamptz[], $5::timestamptz[])
         WITH ORDINALITY AS r(agent_id, schedule_at, start_at, end_at, n)
         LEFT JOIN LATERAL (SELECT ws.id, ws.name
                            FROM wfm.agent_working_schedule aws
                                     INNER JOIN wfm.working_schedule_agent wsa ON wsa.id = aws.working_schedule_agent_id
                                     INNER JOIN wfm.working_schedule ws ON ws.id = wsa.working_schedule_id
                            WHERE wsa.agent_id = r.agent_id
                              AND ws.domain_id = $7
                              AND ws.id <> $1
                              AND ws.state = $6
                              AND aws.schedule_at = r.schedule_at
                            LIMIT 1) taken ON true
         LEFT JOIN LATERAL (SELECT aws.id, aws.start_at, aws.end_at
                            FROM wfm.agent_working_schedule aws
                                     INNER JOIN wfm.working_schedule_agent wsa ON wsa.id = aws.working_schedule_agent_id
                                     INNER JOIN wfm.working_schedule ws ON ws.id = wsa.working_schedule_id
                            WHERE wsa.agent_id = r.agent_id
                              AND ws.domain_id = $7
                              AND (ws.id = $1 OR ws.state = $6)
                              AND aws.id <> $8
                              AND aws.start_at < r.end_at
                              AND aws.end_at > r.start_at
                            ORDER BY aws.start_at
                            LIMIT 1) overlap ON true
ORDER BY r.n`

// shiftFacts: taken is the other active schedule holding the date, overlap the first shift in the way.
type shiftFacts struct {
	Member       bool       `db:"member"`
	Deactivated  bool       `db:"deactivated"`
	Absent       bool       `db:"absent"`
	TakenID      *int64     `db:"taken_id"`
	TakenName    *string    `db:"taken_name"`
	OverlapID    *int64     `db:"overlap_id"`
	OverlapStart *time.Time `db:"overlap_start"`
	OverlapEnd   *time.Time `db:"overlap_end"`
}

// shiftViolations checks the shifts against the schedule; except is the shift an edit replaces, 0 for none.
func shiftViolations(ctx context.Context, tx dbsql.Tx, s *shiftSchedule, in []*model.Shift, except int64) ([]model.Violation, error) {
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
	if err := tx.Primary().Select(ctx, &facts, shiftFactsSQL, s.ID, agents, dates, starts, ends, int16(model.WorkingScheduleStateActive), s.DomainID, except); err != nil {
		return nil, err
	}

	var out []model.Violation

	for i, f := range facts {
		at := model.Violation{AgentID: agents[i], Date: pgtype.Date{Time: dates[i], Valid: true}}
		broke := func(v model.Violation, reason string) {
			v.Reason = reason
			out = append(out, v)
		}

		if !f.Member {
			broke(at, ReasonShiftAgentNotInSchedule)
		}

		if f.Deactivated {
			broke(at, ReasonShiftAgentDeactivated)
		}

		if dates[i].Before(s.StartDate) || dates[i].After(s.EndDate) {
			broke(at, ReasonShiftOutsideSchedule)
		}

		if f.Absent {
			broke(at, ReasonShiftAbsence)
		}

		if f.TakenID != nil {
			taken := at
			taken.Schedule = &model.LookupItem{ID: *f.TakenID, Name: f.TakenName}

			broke(taken, ReasonShiftLockedDate)
		}

		if f.OverlapID == nil {
			continue
		}

		overlap := at
		overlap.ShiftID, overlap.StartAt, overlap.EndAt = *f.OverlapID, *f.OverlapStart, *f.OverlapEnd

		broke(overlap, ReasonShiftOverlap)
	}

	return compactViolations(out), nil
}

// compactViolations orders violations by agent, date and reason, keeping the first of each.
func compactViolations(out []model.Violation) []model.Violation {
	slices.SortStableFunc(out, func(a, b model.Violation) int {
		return cmp.Or(cmp.Compare(a.AgentID, b.AgentID), a.Date.Time.Compare(b.Date.Time), strings.Compare(a.Reason, b.Reason))
	})

	return slices.CompactFunc(out, func(a, b model.Violation) bool {
		return a.AgentID == b.AgentID && a.Date.Time.Equal(b.Date.Time) && a.Reason == b.Reason
	})
}

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

		cte := withChildren(builder.CTE(builder.With("shift").As(builder.Insert(agentWorkingScheduleTable, columns).SQL("RETURNING id"))), user, shift)

		sql, args := builder.Select("shift.id").With(cte.Builder()).From("shift").Build()
		batch.Queue(sql, args...)
	}

	var ids []int64
	if err := batch.Select(ctx, &ids); err != nil {
		return nil, err
	}

	return ids, nil
}

// withChildren adds the pauses and skills of the shift that the CTE's shift statement returns.
func withChildren(cte *builder.CTEQuery, user *model.SignedInUser, shift *model.Shift) *builder.CTEQuery {
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

	return cte
}
