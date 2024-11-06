package storage

import (
	"context"

	"github.com/webitel/webitel-wfm/infra/storage/cache"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql/cluster"
	"github.com/webitel/webitel-wfm/internal/model"
	"github.com/webitel/webitel-wfm/pkg/werror"
)

const (
	workingScheduleTable           = "wfm.working_schedule"
	workingScheduleView            = workingScheduleTable + "_v"
	workingScheduleExtraSkillTable = workingScheduleTable + "_extra_skill"
	workingScheduleAgentTable      = workingScheduleTable + "_agent"
)

type WorkingSchedule struct {
	db    cluster.Store
	cache *cache.Scope[model.WorkingSchedule]
}

func NewWorkingSchedule(db cluster.Store, manager cache.Manager) *WorkingSchedule {
	werror.RegisterConstraint("working_schedule_check", "start_date_at should be lower that end_date_at")

	return &WorkingSchedule{
		db:    db,
		cache: cache.NewScope[model.WorkingSchedule](manager, workingScheduleTable),
	}
}

func (w *WorkingSchedule) CreateWorkingSchedule(ctx context.Context, user *model.SignedInUser, in *model.WorkingSchedule) (*model.WorkingSchedule, error) {
	cteq := w.db.SQL().CTE()
	schedule := []map[string]any{
		{
			"domain_id":              user.DomainId,
			"created_by":             user.Id,
			"updated_by":             user.Id,
			"name":                   in.Name,
			"state":                  in.State,
			"team_id":                in.Team.SafeId(),
			"calendar_id":            in.Calendar.SafeId(),
			"start_date_at":          in.StartDateAt,
			"end_date_at":            in.EndDateAt,
			"start_time_at":          in.StartTimeAt,
			"end_time_at":            in.EndTimeAt,
			"block_outside_activity": in.BlockOutsideActivity,
		},
	}

	cteq.With(w.db.SQL().With("schedule").As(w.db.SQL().Insert(workingScheduleTable, schedule).SQL("RETURNING id")))
	if c := len(in.ExtraSkills); c > 0 {
		skills := make([]map[string]any, 0, len(in.ExtraSkills))
		for _, s := range in.ExtraSkills {
			skill := map[string]any{
				"domain_id":           user.DomainId,
				"working_schedule_id": w.db.SQL().Format("(SELECT id FROM schedule)::bigint"),
				"skill_id":            s.SafeId(),
			}

			skills = append(skills, skill)
		}

		cteq.With(w.db.SQL().With("extra_skills").As(w.db.SQL().Insert(workingScheduleExtraSkillTable, skills).SQL("RETURNING id")))
	}

	if c := len(in.Agents); c > 0 {
		agents := make([]map[string]any, 0, c)
		for _, a := range in.Agents {
			agent := map[string]any{
				"domain_id":           user.DomainId,
				"working_schedule_id": w.db.SQL().Format("(SELECT id FROM schedule)::bigint"),
				"agent_id":            a.SafeId(),
			}

			agents = append(agents, agent)
		}

		cteq.With(w.db.SQL().With("agents").As(w.db.SQL().Insert(workingScheduleAgentTable, agents).SQL("RETURNING id")))
	}

	cte := cteq.Builder()
	sql, args := w.db.SQL().Select("schedule.id").Distinct().With(cte).From(cte.TableNames()...).Build()
	var id int64
	if err := w.db.Primary().Get(ctx, &id, sql, args...); err != nil {
		return nil, err
	}

	out, err := w.ReadWorkingSchedule(ctx, user, &model.SearchItem{Id: id})
	if err != nil {
		return nil, err
	}

	w.cache.Key(user.DomainId, id).Set(ctx, *out)

	return out, nil
}

func (w *WorkingSchedule) ReadWorkingSchedule(ctx context.Context, user *model.SignedInUser, search *model.SearchItem) (*model.WorkingSchedule, error) {
	out, ok := w.cache.Key(user.DomainId, search.Id).Get(ctx)
	if ok {
		return &out, nil
	}

	items, err := w.SearchWorkingSchedule(ctx, user, search)
	if err != nil {
		return nil, err
	}

	if len(items) > 1 {
		return nil, werror.NewDBEntityConflictError("storage.working_schedule.read.conflict")
	}

	if len(items) == 0 {
		return nil, werror.NewDBNoRowsErr("storage.working_schedule.read")
	}

	w.cache.Key(user.DomainId, search.Id).Set(ctx, *items[0])

	return items[0], nil
}

func (w *WorkingSchedule) SearchWorkingSchedule(ctx context.Context, user *model.SignedInUser, search *model.SearchItem) ([]*model.WorkingSchedule, error) {
	out, ok := w.cache.Key(user.DomainId, 0).GetMany(ctx)
	if ok {
		return out, nil
	}

	var (
		items   []*model.WorkingSchedule
		columns []string
	)

	columns = []string{dbsql.Wildcard(model.WorkingSchedule{})}
	if len(search.Fields) > 0 {
		columns = search.Fields
	}

	sb := w.db.SQL().Select(columns...).From(workingScheduleView)
	sql, args := sb.Where(sb.Equal("domain_id", user.DomainId)).
		AddWhereClause(&search.Where("name").WhereClause).
		OrderBy(search.OrderBy(workingScheduleView)).
		Limit(int(search.Limit())).
		Offset(int(search.Offset())).
		Build()

	if err := w.db.StandbyPreferred().Select(ctx, &items, sql, args...); err != nil {
		return nil, err
	}

	w.cache.Key(user.DomainId, 0).SetMany(ctx, items)

	return items, nil
}

func (w *WorkingSchedule) UpdateWorkingSchedule(ctx context.Context, user *model.SignedInUser, in *model.WorkingSchedule) (*model.WorkingSchedule, error) {
	// TODO implement me
	panic("implement me")
}

func (w *WorkingSchedule) DeleteWorkingSchedule(ctx context.Context, user *model.SignedInUser, id int64) (int64, error) {
	// TODO implement me
	panic("implement me")
}
