package storage

import (
	"context"
	"slices"
	"time"

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

		timeZone, err := s.lockSchedule(ctx, tx, user.DomainID, scheduleID)
		if err != nil {
			return err
		}

		var violations []model.Violation
		if err := s.check(ctx, tx, scheduleID, in, &violations); err != nil {
			return err
		}

		if len(violations) > 0 {
			return model.Refuse(ErrShiftsRefused, violations...)
		}

		ids, err = s.insert(ctx, tx, user, scheduleID, timeZone, in)

		return err
	})
	if err != nil {
		return nil, err
	}

	return ids, nil
}

// FOR SHARE: the state, dates and calendar hold until the write commits, and an
// activation (WFM-101) or an edit of the schedule (WFM-100) waits for it.
const shiftScheduleSQL = `SELECT ws.state, tz.sys_name AS time_zone
FROM wfm.working_schedule ws
         INNER JOIN flow.calendar c ON c.id = ws.calendar_id
         INNER JOIN flow.calendar_timezones tz ON tz.id = c.timezone_id
WHERE ws.domain_id = $1
  AND ws.id = $2
FOR SHARE OF ws`

// lockSchedule holds the schedule row, refuses a state shifts can't be written to and
// returns the calendar's time zone.
func (s *Shift) lockSchedule(ctx context.Context, tx dbsql.Tx, domainID, scheduleID int64) (string, error) {
	var schedule struct {
		State    model.WorkingScheduleState `db:"state"`
		TimeZone string                     `db:"time_zone"`
	}

	if err := tx.Primary().Get(ctx, &schedule, shiftScheduleSQL, domainID, scheduleID); err != nil {
		if errors.Is(err, dbsql.ErrNoRows) {
			return "", errors.Wrap(err, errors.WithID("storage.shift.schedule"))
		}

		return "", err
	}

	writable := []model.WorkingScheduleState{model.WorkingScheduleStateDraft, model.WorkingScheduleStatePending, model.WorkingScheduleStateActive}
	if !slices.Contains(writable, schedule.State) {
		return "", errors.Wrap(ErrShiftScheduleState, errors.WithValue("state", schedule.State.String()))
	}

	return schedule.TimeZone, nil
}

// One query over the request lists every broken rule. The request comes as arrays
// of shifts and of pauses numbered by shift; the working date is the day the shift
// starts on in the calendar's zone. Shifts are [start, end), so back to back is
// fine. Overlap is checked against this schedule whatever its state and against
// other active schedules; a date the agent already works in another active
// schedule is locked even with no overlap in time (EP_05). Drafts block nothing.
const shiftCheckSQL = `WITH s AS (SELECT ws.start_date_at, ws.end_date_at, tz.sys_name AS tz
           FROM wfm.working_schedule ws
                    INNER JOIN flow.calendar c ON c.id = ws.calendar_id
                    INNER JOIN flow.calendar_timezones tz ON tz.id = c.timezone_id
           WHERE ws.id = $1),
     r AS (SELECT req.n, req.agent_id, req.start_at, req.end_at, (req.start_at AT TIME ZONE s.tz)::date AS date
           FROM s,
                unnest($2::bigint[], $3::timestamptz[], $4::timestamptz[]) WITH ORDINALITY AS req(agent_id, start_at, end_at, n)),
     p AS (SELECT * FROM unnest($5::bigint[], $6::timestamptz[], $7::timestamptz[]) WITH ORDINALITY AS req(n, start_at, end_at, i)),
     o AS (SELECT x.start_at, x.end_at, x.schedule_at, xa.agent_id, xs.id AS schedule_id, xs.state
           FROM wfm.agent_working_schedule x
                    INNER JOIN wfm.working_schedule_agent xa ON xa.id = x.working_schedule_agent_id
                    INNER JOIN wfm.working_schedule xs ON xs.id = xa.working_schedule_id
           WHERE xa.agent_id = ANY ($2::bigint[]))
SELECT DISTINCT r.agent_id AS operator_id, r.date, 'storage.shift.' || v.reason AS reason
FROM r,
     LATERAL (
         SELECT 'agent_not_in_schedule'
         WHERE NOT EXISTS (SELECT 1 FROM wfm.working_schedule_agent wsa WHERE wsa.working_schedule_id = $1 AND wsa.agent_id = r.agent_id)
         UNION ALL
         SELECT 'outside_schedule'
         FROM s
         WHERE r.date < s.start_date_at OR r.date > s.end_date_at
         UNION ALL
         SELECT 'duration'
         WHERE r.start_at >= r.end_at OR r.end_at - r.start_at > INTERVAL '24 hours'
         UNION ALL
         SELECT 'pause_outside'
         WHERE EXISTS (SELECT 1 FROM p WHERE p.n = r.n AND (p.start_at >= p.end_at OR p.start_at < r.start_at OR p.end_at > r.end_at))
         UNION ALL
         SELECT 'pause_overlap'
         WHERE EXISTS (SELECT 1
                       FROM p a
                                INNER JOIN p b ON b.n = a.n AND b.i <> a.i
                       WHERE a.n = r.n AND a.start_at < b.end_at AND a.end_at > b.start_at)
         UNION ALL
         SELECT 'absence'
         WHERE EXISTS (SELECT 1 FROM wfm.agent_absence aa WHERE aa.agent_id = r.agent_id AND aa.absent_at = r.date)
         UNION ALL
         SELECT 'locked_date'
         WHERE EXISTS (SELECT 1 FROM o WHERE o.agent_id = r.agent_id AND o.schedule_id <> $1 AND o.state = $8 AND o.schedule_at = r.date)
         UNION ALL
         SELECT 'overlap'
         WHERE EXISTS (SELECT 1
                       FROM o
                       WHERE o.agent_id = r.agent_id AND (o.schedule_id = $1 OR o.state = $8)
                         AND o.start_at < r.end_at AND o.end_at > r.start_at)
         UNION ALL
         SELECT 'overlap'
         WHERE EXISTS (SELECT 1 FROM r b WHERE b.agent_id = r.agent_id AND b.n <> r.n AND b.start_at < r.end_at AND b.end_at > r.start_at)
         ) AS v(reason)
ORDER BY operator_id, r.date, reason`

func (s *Shift) check(ctx context.Context, tx dbsql.Tx, scheduleID int64, in []*model.Shift, out *[]model.Violation) error {
	agents := make([]int64, 0, len(in))
	starts := make([]time.Time, 0, len(in))
	ends := make([]time.Time, 0, len(in))

	var (
		pauseOf     []int64
		pauseStarts []time.Time
		pauseEnds   []time.Time
	)

	for i, shift := range in {
		agents = append(agents, shift.AgentID)
		starts = append(starts, shift.Start)
		ends = append(ends, shift.End)

		for _, pause := range shift.Pauses {
			pauseOf = append(pauseOf, int64(i+1))
			pauseStarts = append(pauseStarts, pause.Start)
			pauseEnds = append(pauseEnds, pause.End)
		}
	}

	return tx.Primary().Select(ctx, out, shiftCheckSQL, scheduleID, agents, starts, ends, pauseOf, pauseStarts, pauseEnds,
		int16(model.WorkingScheduleStateActive))
}

// insert queues one statement per shift: the shift, then its pauses and skills off
// the returned id. The rules are checked already, so a valid request fails no constraint.
func (s *Shift) insert(ctx context.Context, tx dbsql.Tx, user *model.SignedInUser, scheduleID int64, timeZone string, in []*model.Shift) ([]int64, error) {
	batch := tx.Primary().Batch()

	for _, shift := range in {
		columns := []map[string]any{{
			"domain_id":                 user.DomainID,
			"created_by":                user.ID,
			"working_schedule_agent_id": builder.Format("(SELECT id::bigint FROM wfm.working_schedule_agent WHERE working_schedule_id = $? AND agent_id = $?)", scheduleID, shift.AgentID),
			"schedule_at":               builder.Format("($?::timestamptz AT TIME ZONE $?)::date", shift.Start, timeZone),
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
