package storage

import (
	"context"
	"slices"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/webitel/webitel-go-kit/pkg/errors"

	"github.com/webitel/webitel-wfm/infra/storage/dbsql"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql/builder"
	"github.com/webitel/webitel-wfm/internal/model"
	"github.com/webitel/webitel-wfm/internal/model/options"
	"github.com/webitel/webitel-wfm/pkg/fields"
)

// ErrWorkingScheduleVersionConflict: unique and foreign-key violations are 409
// too, so the id, not the status, is what tells a client to reread.
var ErrWorkingScheduleVersionConflict = errors.Aborted("working schedule was changed concurrently", errors.WithID("storage.working_schedule.version_conflict"))

var ErrWorkingScheduleSort = errors.InvalidArgument("working schedules sort by name or state only", errors.WithID("storage.working_schedule.sort"))

var (
	ErrWorkingScheduleActive   = errors.FailedPrecondition("an active working schedule is not deleted; change its state first", errors.WithID("storage.working_schedule.active"))
	ErrWorkingScheduleArchived = errors.FailedPrecondition("an archived working schedule is read only", errors.WithID("storage.working_schedule.archived"))

	ErrWorkingScheduleActivationRefused = errors.FailedPrecondition("the schedule breaks the rules of an active schedule", errors.WithID("storage.working_schedule.activation_refused"))
)

// ReasonWorkingScheduleTeamActive: the team has another active schedule on overlapping dates; no agent or date.
const ReasonWorkingScheduleTeamActive = "storage.working_schedule.team_active"

// The sort goes into SQL, so only these; states sort as Active, Pending, Draft, Archived (US_05.01).
var workingScheduleOrder = map[string]string{
	"name":  "name",
	"state": "array_position(ARRAY [3, 2, 1, 4], state::int)",
}

type WorkingScheduleManager interface {
	CreateWorkingSchedule(ctx context.Context, user *model.SignedInUser, in *model.WorkingSchedule) (*model.WorkingSchedule, error)
	ReadWorkingSchedule(ctx context.Context, read *options.Read) (*model.WorkingSchedule, error)
	SearchWorkingSchedule(ctx context.Context, search *options.Search) ([]*model.WorkingSchedule, error)
	UpdateWorkingSchedule(ctx context.Context, user *model.SignedInUser, in *model.WorkingSchedule) (*model.WorkingSchedule, error)
	DeleteWorkingSchedule(ctx context.Context, read *options.Read) (int64, error)

	UpdateWorkingScheduleAddAgents(ctx context.Context, read *options.Read, agentIDs []int64) ([]*model.LookupItem, error)
	UpdateWorkingScheduleRemoveAgent(ctx context.Context, read *options.Read, agentID int64) (int64, error)
	UpdateWorkingScheduleBlockOutsideActivity(ctx context.Context, read *options.Read, ver int32, block bool) (*model.WorkingSchedule, error)
	UpdateWorkingScheduleState(ctx context.Context, read *options.Read, ver int32, state model.WorkingScheduleState) (*model.WorkingSchedule, error)

	DeactivatedAgents(ctx context.Context, domainID int64, agentIDs []int64) ([]int64, error)
}

type WorkingSchedule struct {
	db dbsql.Store
}

func NewWorkingSchedule(db dbsql.Store) *WorkingSchedule {
	dbsql.RegisterConstraint("working_schedule_dates_check", "start_date_at should not be after end_date_at")

	return &WorkingSchedule{db: db}
}

func (w *WorkingSchedule) CreateWorkingSchedule(ctx context.Context, user *model.SignedInUser, in *model.WorkingSchedule) (*model.WorkingSchedule, error) {
	cteq := builder.CTE()
	schedule := []map[string]any{
		{
			"domain_id":              user.DomainID,
			"created_by":             user.ID,
			"updated_by":             user.ID,
			"name":                   in.Name,
			"state":                  int32(model.WorkingScheduleStateDraft),
			"team_id":                in.Team.SafeID(),
			"calendar_id":            in.Calendar.SafeID(),
			"start_date_at":          in.StartDateAt,
			"end_date_at":            in.EndDateAt,
			"start_time_at":          in.StartTimeAt,
			"end_time_at":            in.EndTimeAt,
			"block_outside_activity": in.BlockOutsideActivity,
		},
	}

	cteq.With(builder.With("schedule").As(builder.Insert(builder.WorkingScheduleTable.Name(), schedule).SQL("RETURNING id")))

	if c := len(in.ExtraSkills); c > 0 {
		skills := make([]map[string]any, 0, len(in.ExtraSkills))
		for _, s := range in.ExtraSkills {
			skill := map[string]any{
				"domain_id":           user.DomainID,
				"working_schedule_id": builder.Format("(SELECT id FROM schedule)::bigint"),
				"skill_id":            s.SafeID(),
			}

			skills = append(skills, skill)
		}

		cteq.With(builder.With("extra_skills").As(builder.Insert(builder.WorkingScheduleExtraSkillTable.Name(), skills).SQL("RETURNING id")))
	}

	if c := len(in.Agents); c > 0 {
		agents := make([]map[string]any, 0, c)

		for _, a := range in.Agents {
			agent := map[string]any{
				"domain_id":           user.DomainID,
				"working_schedule_id": builder.Format("(SELECT id FROM schedule)::bigint"),
				"agent_id":            a.SafeID(),
			}

			agents = append(agents, agent)
		}

		cteq.With(builder.With("agents").As(builder.Insert(builder.WorkingScheduleAgentTable.Name(), agents).SQL(skipAgentInSchedule + " RETURNING id")))
	}

	cte := cteq.Builder()
	sql, args := builder.Select("schedule.id").Distinct().With(cte).From(cte.TableNames()...).Build()

	var id int64
	if err := w.db.Primary().Get(ctx, &id, sql, args...); err != nil {
		return nil, err
	}

	read, err := options.NewRead(ctx, options.WithID(id))
	if err != nil {
		return nil, err
	}

	// The row was just committed; a standby may not have it yet.
	return w.read(ctx, w.db.Primary(), read)
}

// ReadWorkingSchedule reads from the primary: a client rereads the schedule
// straight after saving an edit, and a standby may not have replicated it yet.
func (w *WorkingSchedule) ReadWorkingSchedule(ctx context.Context, read *options.Read) (*model.WorkingSchedule, error) {
	return w.read(ctx, w.db.Primary(), read)
}

func (w *WorkingSchedule) read(ctx context.Context, node dbsql.Node, read *options.Read) (*model.WorkingSchedule, error) {
	search, err := options.NewSearch(ctx, options.WithID(read.ID()))
	if err != nil {
		return nil, err
	}

	items, err := w.search(ctx, node, search.PopulateFromRead(read))
	if err != nil {
		return nil, err
	}

	if len(items) > 1 {
		return nil, errors.Wrap(dbsql.ErrEntityConflict, errors.WithID("storage.working_schedule.read.conflict"))
	}

	if len(items) == 0 {
		return nil, errors.Wrap(dbsql.ErrNoRows, errors.WithID("storage.working_schedule.read"))
	}

	return items[0], nil
}

func (w *WorkingSchedule) SearchWorkingSchedule(ctx context.Context, search *options.Search) ([]*model.WorkingSchedule, error) {
	return w.search(ctx, w.db.StandbyPreferred(), search)
}

func (w *WorkingSchedule) search(ctx context.Context, node dbsql.Node, search *options.Search) ([]*model.WorkingSchedule, error) {
	columns := []string{fields.Wildcard(model.WorkingSchedule{})}
	if f := search.Fields(); len(f) > 0 {
		columns = workingScheduleColumns(f)
	}

	sb := builder.Select(columns...).From(builder.WorkingScheduleViewTable.Name())
	sb.Where(sb.Equal("domain_id", search.User().DomainID))

	if search.Query() != "" {
		sb.Where(sb.ILike("name", search.Query()))
	}

	if ids := search.IDs(); len(ids) > 0 {
		sb.Where(sb.In("id", builder.ConvertArgs(ids)...))
	}

	order, direction := "created_at", builder.OrderDirectionASC

	for field, d := range search.OrderBy() {
		expr, ok := workingScheduleOrder[field]
		if !ok {
			return nil, errors.Wrap(ErrWorkingScheduleSort, errors.WithValue("sort", field))
		}

		order, direction = expr, d
	}

	sb.OrderBy(builder.OrderBy(order, direction), builder.OrderBy("id", direction))

	var items []*model.WorkingSchedule

	sql, args := sb.Limit(search.Size()).Offset(search.Offset()).Build()
	if err := node.Select(ctx, &items, sql, args...); err != nil {
		return nil, err
	}

	return items, nil
}

// workingScheduleColumns turns a field mask into view columns: etag is built
// from id and ver, and scany rejects a column selected twice.
func workingScheduleColumns(mask []string) []string {
	columns := make([]string, 0, len(mask)+1)
	for _, f := range mask {
		if f == "etag" {
			columns = append(columns, "id", "ver")
		} else {
			columns = append(columns, f)
		}
	}

	slices.Sort(columns)

	return slices.Compact(columns)
}

func (w *WorkingSchedule) UpdateWorkingSchedule(ctx context.Context, user *model.SignedInUser, in *model.WorkingSchedule) (*model.WorkingSchedule, error) {
	cteq := builder.CTE()
	schedule := map[string]any{
		"updated_by":    user.ID,
		"name":          in.Name,
		"team_id":       in.Team.SafeID(),
		"calendar_id":   in.Calendar.SafeID(),
		"start_date_at": in.StartDateAt,
		"end_date_at":   in.EndDateAt,
		"start_time_at": in.StartTimeAt,
		"end_time_at":   in.EndTimeAt,
		"ver":           builder.Format("ver + 1"),
	}

	// The skills CTEs start from what the guarded schedule CTE returned, so a
	// stale version writes nothing; by in.ID they would wipe the skills and, as
	// working_schedule_id is nullable, commit orphans.
	upd := builder.Update(builder.WorkingScheduleTable.Name(), schedule)
	upd.Where(upd.Equal("domain_id", user.DomainID), upd.Equal("id", in.ID), upd.Equal("ver", in.Ver)).SQL("RETURNING id, domain_id")
	cteq.With(builder.With("schedule").As(upd))

	del := builder.Delete(builder.WorkingScheduleExtraSkillTable.Name())
	del.Where(del.Equal("domain_id", user.DomainID), "working_schedule_id IN (SELECT id FROM schedule)").SQL("RETURNING id")
	cteq.With(builder.With("del_extra_skills").As(del))

	if c := len(in.ExtraSkills); c > 0 {
		skills := make([]*int64, 0, c)
		for _, s := range in.ExtraSkills {
			skills = append(skills, s.SafeID())
		}

		ins := builder.Insert(builder.WorkingScheduleExtraSkillTable.Name(), nil).Cols("domain_id", "working_schedule_id", "skill_id").Returning("id")
		sel := ins.Select("schedule.domain_id", "schedule.id", "skill_id")
		sel.From("schedule", "unnest("+sel.Var(skills)+"::bigint[]) AS skill_id")
		cteq.With(builder.With("extra_skills").As(ins))
	}

	// Data-modifying CTEs run whether or not they are read, so only schedule is
	// selected: a cross join with an empty del_extra_skills would return no row.
	sql, args := builder.Select("schedule.id").With(cteq.Builder()).From("schedule").Build()

	read, err := options.NewRead(ctx, options.WithID(in.ID))
	if err != nil {
		return nil, err
	}

	var out *model.WorkingSchedule

	err = w.db.InTx(ctx, func(ctx context.Context, tx dbsql.Tx) error {
		old, err := lockSchedule(ctx, tx, user.DomainID, in.ID, lockUpdate)
		if err != nil {
			return err
		}

		var id int64
		if err := tx.Primary().Get(ctx, &id, sql, args...); err != nil {
			// The row is locked above, so only its version can differ.
			if errors.Is(err, dbsql.ErrNoRows) {
				return ErrWorkingScheduleVersionConflict
			}

			return err
		}

		// The time scale, the name and the skills keep the shifts.
		switch {
		case old.Team != in.Team.ID:
			// The new team's agents replace every agent; removing one takes their shifts through the foreign key.
			if err := tx.Primary().Exec(ctx, "DELETE FROM wfm.working_schedule_agent WHERE domain_id = $1 AND working_schedule_id = $2", user.DomainID, in.ID); err != nil {
				return err
			}

			agents := make([]int64, 0, len(in.Agents))
			for _, a := range in.Agents {
				agents = append(agents, a.ID)
			}

			if err := tx.Primary().Exec(ctx, `INSERT INTO wfm.working_schedule_agent (domain_id, working_schedule_id, agent_id)
SELECT $1, $2, unnest($3::int8[]) `+skipAgentInSchedule, user.DomainID, in.ID, agents); err != nil {
				return err
			}
		case old.Calendar != in.Calendar.ID:
			// A new calendar moves the working dates.
			if _, err := deleteShifts(ctx, tx, user.DomainID, in.ID, old.StartDate, old.EndDate); err != nil {
				return err
			}
		case in.StartDateAt.Time.After(old.StartDate) || in.EndDateAt.Time.Before(old.EndDate):
			// The shifts outside the new dates go; a range ending before it starts deletes nothing.
			if _, err := deleteShifts(ctx, tx, user.DomainID, in.ID, old.StartDate, in.StartDateAt.Time.AddDate(0, 0, -1)); err != nil {
				return err
			}

			if _, err := deleteShifts(ctx, tx, user.DomainID, in.ID, in.EndDateAt.Time.AddDate(0, 0, 1), old.EndDate); err != nil {
				return err
			}
		}

		out, err = w.read(ctx, tx.Primary(), read)

		return err
	})
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (w *WorkingSchedule) DeleteWorkingSchedule(ctx context.Context, read *options.Read) (int64, error) {
	err := w.db.InTx(ctx, func(ctx context.Context, tx dbsql.Tx) error {
		schedule, err := lockSchedule(ctx, tx, read.User().DomainID, read.ID(), lockUpdate)
		if err != nil {
			return err
		}

		if !schedule.State.Deletable() {
			return ErrWorkingScheduleActive
		}

		db := builder.Delete(builder.WorkingScheduleTable.Name())
		sql, args := db.Where(db.Equal("domain_id", read.User().DomainID), db.Equal("id", read.ID())).Build()

		return tx.Primary().Exec(ctx, sql, args...)
	})
	if err != nil {
		return 0, err
	}

	return read.ID(), nil
}

// writableAgents locks the schedule as shift writes do and refuses one whose agents are read only.
func writableAgents(ctx context.Context, tx dbsql.Tx, read *options.Read) error {
	schedule, err := lockSchedule(ctx, tx, read.User().DomainID, read.ID(), lockShare)
	if err != nil {
		return err
	}

	if !schedule.State.AgentsWritable() {
		return ErrWorkingScheduleArchived
	}

	return nil
}

// skipAgentInSchedule makes adding an agent the schedule already has a no-op. It names no
// columns, so it also runs where the migration adding the unique key is not applied yet.
const skipAgentInSchedule = "ON CONFLICT DO NOTHING"

func (w *WorkingSchedule) UpdateWorkingScheduleAddAgents(ctx context.Context, read *options.Read, agentIDs []int64) ([]*model.LookupItem, error) {
	columns := make([]map[string]any, 0, len(agentIDs))
	for _, agentID := range agentIDs {
		columns = append(columns, map[string]any{
			"domain_id":           read.User().DomainID,
			"working_schedule_id": read.ID(),
			"agent_id":            agentID,
		})
	}

	var out *model.WorkingSchedule

	err := w.db.InTx(ctx, func(ctx context.Context, tx dbsql.Tx) error {
		if err := writableAgents(ctx, tx, read); err != nil {
			return err
		}

		sql, args := builder.Insert(builder.WorkingScheduleAgentTable.Name(), columns).SQL(skipAgentInSchedule).Build()
		if err := tx.Primary().Exec(ctx, sql, args...); err != nil {
			return err
		}

		var err error

		out, err = w.read(ctx, tx.Primary(), read)

		return err
	})
	if err != nil {
		return nil, err
	}

	return out.Agents, nil
}

func (w *WorkingSchedule) UpdateWorkingScheduleRemoveAgent(ctx context.Context, read *options.Read, agentID int64) (int64, error) {
	err := w.db.InTx(ctx, func(ctx context.Context, tx dbsql.Tx) error {
		if err := writableAgents(ctx, tx, read); err != nil {
			return err
		}

		db := builder.Delete(builder.WorkingScheduleAgentTable.Name())
		sql, args := db.Where(
			db.Equal("domain_id", read.User().DomainID),
			db.Equal("working_schedule_id", read.ID()),
			db.Equal("agent_id", agentID),
		).Build()

		return tx.Primary().Exec(ctx, sql, args...)
	})
	if err != nil {
		return 0, err
	}

	return agentID, nil
}

func (w *WorkingSchedule) UpdateWorkingScheduleBlockOutsideActivity(ctx context.Context, read *options.Read, ver int32, block bool) (*model.WorkingSchedule, error) {
	var out *model.WorkingSchedule

	err := w.db.InTx(ctx, func(ctx context.Context, tx dbsql.Tx) error {
		schedule, err := lockSchedule(ctx, tx, read.User().DomainID, read.ID(), lockUpdate)
		if err != nil {
			return err
		}

		if schedule.Ver != ver {
			return ErrWorkingScheduleVersionConflict
		}

		if !schedule.State.BlockWritable() {
			return ErrWorkingScheduleArchived
		}

		if err := tx.Primary().Exec(ctx, "UPDATE wfm.working_schedule SET block_outside_activity = $1, updated_by = $2, ver = ver + 1 WHERE domain_id = $3 AND id = $4",
			block, read.User().ID, read.User().DomainID, read.ID()); err != nil {
			return err
		}

		out, err = w.read(ctx, tx.Primary(), read)

		return err
	})
	if err != nil {
		return nil, err
	}

	return out, nil
}

// UpdateWorkingScheduleState moves the schedule to another state; moving to active first checks the rules of an active schedule.
func (w *WorkingSchedule) UpdateWorkingScheduleState(ctx context.Context, read *options.Read, ver int32, state model.WorkingScheduleState) (*model.WorkingSchedule, error) {
	domainID, id := read.User().DomainID, read.ID()
	activate := state == model.WorkingScheduleStateActive

	var out *model.WorkingSchedule

	err := w.db.InTx(ctx, func(ctx context.Context, tx dbsql.Tx) error {
		// The team's and the agents' locks come before the row, the order shift writes take them in.
		var locked struct {
			Team   int64   `db:"team_id"`
			Agents []int64 `db:"agents"`
		}

		if activate {
			if err := tx.Primary().Get(ctx, &locked, `SELECT ws.team_id,
       array(SELECT wsa.agent_id FROM wfm.working_schedule_agent wsa WHERE wsa.domain_id = $1 AND wsa.working_schedule_id = ws.id) AS agents
FROM wfm.working_schedule ws
WHERE ws.domain_id = $1
  AND ws.id = $2`, domainID, id); err != nil {
				return err
			}

			locks := []dbsql.LockKey{dbsql.TeamLock(locked.Team)}
			for _, agent := range locked.Agents {
				locks = append(locks, dbsql.OperatorLock(agent))
			}

			if err := tx.Lock(ctx, locks...); err != nil {
				return err
			}
		}

		schedule, err := lockSchedule(ctx, tx, domainID, id, lockUpdate)
		if err != nil {
			return err
		}

		if schedule.Ver != ver {
			return ErrWorkingScheduleVersionConflict
		}

		if !schedule.State.StateWritable() {
			return ErrWorkingScheduleArchived
		}

		if activate {
			violations, err := activationViolations(ctx, tx, schedule, locked.Agents)
			if err != nil {
				return err
			}

			if len(violations) > 0 {
				return model.Refuse(ErrWorkingScheduleActivationRefused, violations...)
			}
		}

		if err := tx.Primary().Exec(ctx, "UPDATE wfm.working_schedule SET state = $1, updated_by = $2, ver = ver + 1 WHERE domain_id = $3 AND id = $4",
			int16(state), read.User().ID, domainID, id); err != nil {
			return err
		}

		out, err = w.read(ctx, tx.Primary(), read)

		return err
	})
	if err != nil {
		return nil, err
	}

	return out, nil
}

// Only active schedules count: the team's other ones on the same days, and the agents' shifts in other
// active schedules on the same working date or at the same time, a night one included.
const activationTeamSQL = `
SELECT id, name
FROM wfm.working_schedule
WHERE domain_id = $1
  AND team_id = $2
  AND id <> $3
  AND state = $4
  AND start_date_at <= $6
  AND end_date_at >= $5
ORDER BY id`

const activationShiftsSQL = `
SELECT wsa.agent_id,
       aws.schedule_at,
       ws.id                                                       AS schedule_id,
       ws.name                                                     AS schedule_name,
       other.id                                                    AS shift_id,
       other.start_at,
       other.end_at,
       other.schedule_at = aws.schedule_at                         AS same_date,
       other.start_at < aws.end_at AND other.end_at > aws.start_at AS overlaps
FROM wfm.working_schedule_agent wsa
         INNER JOIN wfm.agent_working_schedule aws ON aws.working_schedule_agent_id = wsa.id
         INNER JOIN wfm.working_schedule_agent owsa ON owsa.agent_id = wsa.agent_id
         INNER JOIN wfm.working_schedule ws ON ws.id = owsa.working_schedule_id
         INNER JOIN wfm.agent_working_schedule other ON other.working_schedule_agent_id = owsa.id
WHERE wsa.domain_id = $1
  AND wsa.working_schedule_id = $2
  AND ws.domain_id = $1
  AND ws.id <> $2
  AND ws.state = $3
  AND (other.schedule_at = aws.schedule_at OR other.start_at < aws.end_at AND other.end_at > aws.start_at)
ORDER BY wsa.agent_id, aws.schedule_at, other.start_at`

// activationViolations checks the schedule against the active ones; locked are the agents whose locks are held.
func activationViolations(ctx context.Context, tx dbsql.Tx, s *shiftSchedule, locked []int64) ([]model.Violation, error) {
	// An agent added after the locks were taken could get a shift elsewhere unseen; the client rereads.
	var added bool
	if err := tx.Primary().Get(ctx, &added, "SELECT EXISTS (SELECT 1 FROM wfm.working_schedule_agent WHERE domain_id = $1 AND working_schedule_id = $2 AND agent_id <> ALL ($3::int8[]))",
		s.DomainID, s.ID, locked); err != nil {
		return nil, err
	}

	if added {
		return nil, ErrWorkingScheduleVersionConflict
	}

	active := int16(model.WorkingScheduleStateActive)

	var teams []model.LookupItem
	if err := tx.Primary().Select(ctx, &teams, activationTeamSQL, s.DomainID, s.Team, s.ID, active, s.StartDate, s.EndDate); err != nil {
		return nil, err
	}

	var shifts []struct {
		AgentID      int64     `db:"agent_id"`
		Date         time.Time `db:"schedule_at"`
		ScheduleID   int64     `db:"schedule_id"`
		ScheduleName string    `db:"schedule_name"`
		ShiftID      int64     `db:"shift_id"`
		StartAt      time.Time `db:"start_at"`
		EndAt        time.Time `db:"end_at"`
		SameDate     bool      `db:"same_date"`
		Overlaps     bool      `db:"overlaps"`
	}
	if err := tx.Primary().Select(ctx, &shifts, activationShiftsSQL, s.DomainID, s.ID, active); err != nil {
		return nil, err
	}

	out := make([]model.Violation, 0, len(teams)+len(shifts))
	for _, team := range teams {
		out = append(out, model.Violation{Reason: ReasonWorkingScheduleTeamActive, Schedule: &team})
	}

	for _, shift := range shifts {
		at := model.Violation{AgentID: shift.AgentID, Date: pgtype.Date{Time: shift.Date, Valid: true}}

		if shift.SameDate {
			taken := at
			taken.Reason, taken.Schedule = ReasonShiftLockedDate, &model.LookupItem{ID: shift.ScheduleID, Name: &shift.ScheduleName}
			out = append(out, taken)
		}

		if !shift.Overlaps {
			continue
		}

		overlap := at
		overlap.Reason, overlap.ShiftID, overlap.StartAt, overlap.EndAt = ReasonShiftOverlap, shift.ShiftID, shift.StartAt, shift.EndAt
		out = append(out, overlap)
	}

	return compactViolations(out), nil
}

func (w *WorkingSchedule) DeactivatedAgents(ctx context.Context, domainID int64, agentIDs []int64) ([]int64, error) {
	if len(agentIDs) == 0 {
		return nil, nil
	}

	var ids []int64

	err := w.db.Primary().Select(ctx, &ids, `SELECT a.id
  FROM call_center.cc_agent a
  JOIN directory.wbt_user u ON u.id = a.user_id
 WHERE a.domain_id = $1
   AND a.id = ANY($2::int8[])
   AND u.deleted_at NOTNULL`, domainID, agentIDs)
	if err != nil {
		return nil, err
	}

	return ids, nil
}
