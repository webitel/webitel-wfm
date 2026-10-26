package storage

import (
	"context"
	"fmt"

	"go.uber.org/fx"

	"github.com/webitel/webitel-go-kit/pkg/cache"
	"github.com/webitel/webitel-go-kit/pkg/errors"

	"github.com/webitel/webitel-wfm/infra/storage/dbsql"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql/builder"
	"github.com/webitel/webitel-wfm/internal/model"
	"github.com/webitel/webitel-wfm/internal/model/options"
	"github.com/webitel/webitel-wfm/pkg/fields"
)

type WorkingScheduleManager interface {
	CreateWorkingSchedule(ctx context.Context, user *model.SignedInUser, in *model.WorkingSchedule) (*model.WorkingSchedule, error)
	ReadWorkingSchedule(ctx context.Context, read *options.Read) (*model.WorkingSchedule, error)
	SearchWorkingSchedule(ctx context.Context, search *options.Search) ([]*model.WorkingSchedule, error)
	UpdateWorkingSchedule(ctx context.Context, user *model.SignedInUser, in *model.WorkingSchedule) (*model.WorkingSchedule, error)
	DeleteWorkingSchedule(ctx context.Context, read *options.Read) (int64, error)

	UpdateWorkingScheduleAddAgents(ctx context.Context, read *options.Read, agentIDs []int64) ([]*model.LookupItem, error)
	UpdateWorkingScheduleRemoveAgent(ctx context.Context, read *options.Read, agentID int64) (int64, error)
}

type WorkingSchedule struct {
	db    dbsql.Store
	items cache.Cache[string, model.WorkingSchedule]
}

func NewWorkingSchedule(lc fx.Lifecycle, db dbsql.Store, cfg cache.RistrettoConfig) (*WorkingSchedule, error) {
	dbsql.RegisterConstraint("working_schedule_check", "start_date_at should be lower that end_date_at")

	items, err := cache.New[string, model.WorkingSchedule]().Name(builder.WorkingScheduleTable.Name()).L1(cfg).Build()
	if err != nil {
		return nil, err
	}

	lc.Append(fx.Hook{OnStop: func(context.Context) error { return items.Close() }})

	return &WorkingSchedule{db: db, items: items}, nil
}

func (w *WorkingSchedule) CreateWorkingSchedule(ctx context.Context, user *model.SignedInUser, in *model.WorkingSchedule) (*model.WorkingSchedule, error) {
	cteq := builder.CTE()
	schedule := []map[string]any{
		{
			"domain_id":              user.DomainId,
			"created_by":             user.Id,
			"updated_by":             user.Id,
			"name":                   in.Name,
			"state":                  int32(in.State),
			"team_id":                in.Team.SafeId(),
			"calendar_id":            in.Calendar.SafeId(),
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
				"domain_id":           user.DomainId,
				"working_schedule_id": builder.Format("(SELECT id FROM schedule)::bigint"),
				"skill_id":            s.SafeId(),
			}

			skills = append(skills, skill)
		}

		cteq.With(builder.With("extra_skills").As(builder.Insert(builder.WorkingScheduleExtraSkillTable.Name(), skills).SQL("RETURNING id")))
	}

	if c := len(in.Agents); c > 0 {
		agents := make([]map[string]any, 0, c)
		for _, a := range in.Agents {
			agent := map[string]any{
				"domain_id":           user.DomainId,
				"working_schedule_id": builder.Format("(SELECT id FROM schedule)::bigint"),
				"agent_id":            a.SafeId(),
			}

			agents = append(agents, agent)
		}

		cteq.With(builder.With("agents").As(builder.Insert(builder.WorkingScheduleAgentTable.Name(), agents).SQL("RETURNING id")))
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

func (w *WorkingSchedule) ReadWorkingSchedule(ctx context.Context, read *options.Read) (*model.WorkingSchedule, error) {
	return w.read(ctx, w.db.StandbyPreferred(), read)
}

func (w *WorkingSchedule) read(ctx context.Context, node dbsql.Node, read *options.Read) (*model.WorkingSchedule, error) {
	// A field mask makes the row a partial one, which must not stand in for
	// the whole entity under the same key.
	cacheable := len(read.Fields()) == 0
	key := itemKey(read.User().DomainId, read.ID())

	if cacheable {
		if out, ok, err := w.items.Get(ctx, key); err == nil && ok {
			return &out, nil
		}
	}

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

	if cacheable {
		_ = w.items.Set(ctx, key, *items[0])
	}

	return items[0], nil
}

func (w *WorkingSchedule) SearchWorkingSchedule(ctx context.Context, search *options.Search) ([]*model.WorkingSchedule, error) {
	return w.search(ctx, w.db.StandbyPreferred(), search)
}

// search is not cached: a collection key can encode neither the filters nor the
// paging of the query that produced it, and no write path could invalidate it.
func (w *WorkingSchedule) search(ctx context.Context, node dbsql.Node, search *options.Search) ([]*model.WorkingSchedule, error) {
	columns := []string{fields.Wildcard(model.WorkingSchedule{})}
	if f := search.Fields(); len(f) > 0 {
		columns = f
	}

	sb := builder.Select(columns...).From(builder.WorkingScheduleViewTable.Name())
	sb.Where(sb.Equal("domain_id", search.User().DomainId))

	if search.Query() != "" {
		sb.Where(sb.ILike("name", search.Query()))
	}

	if ids := search.IDs(); len(ids) > 0 {
		sb.Where(sb.In("id", builder.ConvertArgs(ids)...))
	}

	orderBy := search.OrderBy()
	if len(orderBy) == 0 {
		orderBy.WithOrderBy("created_at", builder.OrderDirectionASC)
	}

	for field, direction := range orderBy {
		sb.OrderBy(builder.OrderBy(field, direction))
	}

	var items []*model.WorkingSchedule

	sql, args := sb.Limit(search.Size()).Offset(search.Offset()).Build()
	if err := node.Select(ctx, &items, sql, args...); err != nil {
		return nil, err
	}

	return items, nil
}

func (w *WorkingSchedule) UpdateWorkingSchedule(ctx context.Context, user *model.SignedInUser, in *model.WorkingSchedule) (*model.WorkingSchedule, error) {
	cteq := builder.CTE()
	schedule := map[string]any{
		"updated_by":             user.Id,
		"name":                   in.Name,
		"block_outside_activity": in.BlockOutsideActivity,
	}

	cteq.With(builder.With("schedule").As(builder.Update(builder.WorkingScheduleTable.Name(), schedule).SQL("RETURNING id")))

	del := builder.Delete(builder.WorkingScheduleExtraSkillTable.Name())
	del.Where(del.Equal("domain_id", user.DomainId), del.Equal("working_schedule_id", in.Id)).SQL("RETURNING id")
	cteq.With(builder.With("del_extra_skills").As(del))

	if c := len(in.ExtraSkills); c > 0 {
		skills := make([]map[string]any, 0, len(in.ExtraSkills))
		for _, s := range in.ExtraSkills {
			skill := map[string]any{
				"domain_id":           user.DomainId,
				"working_schedule_id": builder.Format("(SELECT id FROM schedule)::bigint"),
				"skill_id":            s.SafeId(),
			}

			skills = append(skills, skill)
		}

		cteq.With(builder.With("extra_skills").As(builder.Insert(builder.WorkingScheduleExtraSkillTable.Name(), skills).SQL("RETURNING id")))
	}

	cte := cteq.Builder()
	sql, args := builder.Select("schedule.id").Distinct().With(cte).From(cte.TableNames()...).Build()

	var id int64
	if err := w.db.Primary().Get(ctx, &id, sql, args...); err != nil {
		return nil, err
	}

	_ = w.items.Delete(ctx, itemKey(user.DomainId, in.Id))

	read, err := options.NewRead(ctx, options.WithID(in.Id))
	if err != nil {
		return nil, err
	}

	return w.read(ctx, w.db.Primary(), read)
}

func (w *WorkingSchedule) DeleteWorkingSchedule(ctx context.Context, read *options.Read) (int64, error) {
	db := builder.Delete(builder.WorkingScheduleTable.Name())
	clauses := []string{
		db.Equal("domain_id", read.User().DomainId),
		db.Equal("id", read.ID()),
	}

	sql, args := db.Where(clauses...).Build()
	if err := w.db.Primary().Exec(ctx, sql, args...); err != nil {
		return 0, err
	}

	_ = w.items.Delete(ctx, itemKey(read.User().DomainId, read.ID()))

	return read.ID(), nil
}

func (w *WorkingSchedule) UpdateWorkingScheduleAddAgents(ctx context.Context, read *options.Read, agentIDs []int64) ([]*model.LookupItem, error) {
	columns := make([]map[string]any, 0, len(agentIDs))
	for _, agentID := range agentIDs {
		columns = append(columns, map[string]any{
			"domain_id":           read.User().DomainId,
			"working_schedule_id": read.ID(),
			"agent_id":            agentID,
		})
	}

	sql, args := builder.Insert(builder.WorkingScheduleAgentTable.Name(), columns).Build()
	if err := w.db.Primary().Exec(ctx, sql, args...); err != nil {
		return nil, err
	}

	_ = w.items.Delete(ctx, itemKey(read.User().DomainId, read.ID()))

	out, err := w.read(ctx, w.db.Primary(), read)
	if err != nil {
		return nil, err
	}

	return out.Agents, nil
}

func (w *WorkingSchedule) UpdateWorkingScheduleRemoveAgent(ctx context.Context, read *options.Read, agentID int64) (int64, error) {
	db := builder.Delete(builder.WorkingScheduleAgentTable.Name())
	sql, args := db.Where(
		db.Equal("domain_id", read.User().DomainId),
		db.Equal("working_schedule_id", read.ID()),
		db.Equal("agent_id", agentID),
	).Build()

	if err := w.db.Primary().Exec(ctx, sql, args...); err != nil {
		return 0, err
	}

	_ = w.items.Delete(ctx, itemKey(read.User().DomainId, read.ID()))

	return agentID, nil
}

func itemKey(domain, id int64) string {
	return fmt.Sprintf("domain-%d-id-%d", domain, id)
}
