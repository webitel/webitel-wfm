package storage

import (
	"context"
	"fmt"

	"github.com/webitel/webitel-wfm/infra/storage/cache"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql"
	b "github.com/webitel/webitel-wfm/infra/storage/dbsql/builder"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql/cluster"
	"github.com/webitel/webitel-wfm/internal/model"
	"github.com/webitel/webitel-wfm/internal/model/options"
	"github.com/webitel/webitel-wfm/pkg/werror"
)

type WorkingScheduleManager interface {
	CreateWorkingSchedule(ctx context.Context, read *options.Read, in *model.WorkingSchedule) (int64, error)
	ReadWorkingSchedule(ctx context.Context, read *options.Read) (*model.WorkingSchedule, error)
	SearchWorkingSchedule(ctx context.Context, search *options.Search) ([]*model.WorkingSchedule, error)
	UpdateWorkingSchedule(ctx context.Context, read *options.Read, in *model.WorkingSchedule) error
	DeleteWorkingSchedule(ctx context.Context, read *options.Read) (int64, error)

	UpdateWorkingScheduleAddAgents(ctx context.Context, read *options.Read, agentIds []int64) error
	UpdateWorkingScheduleRemoveAgent(ctx context.Context, read *options.Read, agentId int64) (int64, error)
}

type WorkingSchedule struct {
	db    cluster.Store
	cache *cache.Scope[model.WorkingSchedule]
}

func NewWorkingSchedule(db cluster.Store, manager cache.Manager) *WorkingSchedule {
	dbsql.RegisterConstraint("working_schedule_check", "start_date_at should be lower that end_date_at")

	return &WorkingSchedule{
		db:    db,
		cache: cache.NewScope[model.WorkingSchedule](manager, b.WorkingScheduleTable.Name()),
	}
}

func (w *WorkingSchedule) CreateWorkingSchedule(ctx context.Context, read *options.Read, in *model.WorkingSchedule) (int64, error) {
	cteq := b.CTE()
	schedule := []map[string]any{
		{
			"domain_id":              read.User().DomainId,
			"created_by":             read.User().Id,
			"updated_by":             read.User().Id,
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

	cteq.With(b.With("schedule").As(b.Insert(b.WorkingScheduleTable.Name(), schedule).SQL("RETURNING id")))
	if c := len(in.ExtraSkills); c > 0 {
		skills := make([]map[string]any, 0, len(in.ExtraSkills))
		for _, s := range in.ExtraSkills {
			skill := map[string]any{
				"domain_id":           read.User().DomainId,
				"working_schedule_id": b.Format("(SELECT id FROM schedule)::bigint"),
				"skill_id":            s.SafeId(),
			}

			skills = append(skills, skill)
		}

		cteq.With(b.With("extra_skills").As(b.Insert(b.WorkingScheduleExtraSkillTable.Name(), skills).SQL("RETURNING id")))
	}

	if c := len(in.Agents); c > 0 {
		agents := make([]map[string]any, 0, c)
		for _, a := range in.Agents {
			agent := map[string]any{
				"domain_id":           read.User().DomainId,
				"working_schedule_id": b.Format("(SELECT id FROM schedule)::bigint"),
				"agent_id":            a.SafeId(),
			}

			agents = append(agents, agent)
		}

		cteq.With(b.With("agents").As(b.Insert(b.WorkingScheduleAgentTable.Name(), agents).SQL("RETURNING id")))
	}

	cte := cteq.Builder()
	sql, args := b.Select("schedule.id").Distinct().With(cte).From(cte.TableNames()...).Build()
	var id int64
	if err := w.db.Primary().Get(ctx, &id, sql, args...); err != nil {
		return 0, err
	}

	return id, nil
}

func (w *WorkingSchedule) ReadWorkingSchedule(ctx context.Context, read *options.Read) (*model.WorkingSchedule, error) {
	search, err := options.NewSearch(ctx, options.WithID(read.ID()))
	if err != nil {
		return nil, err
	}

	items, err := w.SearchWorkingSchedule(ctx, search.PopulateFromRead(read))
	if err != nil {
		return nil, err
	}

	if len(items) > 1 {
		return nil, werror.Wrap(dbsql.ErrEntityConflict, werror.WithID("storage.working_schedule.read.conflict"))
	}

	if len(items) == 0 {
		return nil, werror.Wrap(dbsql.ErrNoRows, werror.WithID("storage.working_schedule.read"))
	}

	return items[0], nil
}

func (w *WorkingSchedule) SearchWorkingSchedule(ctx context.Context, search *options.Search) ([]*model.WorkingSchedule, error) {
	const (
		linkCreatedBy = 1 << iota
		linkUpdatedBy
		linkTeam
		linkCalendar
	)

	var (
		workingSchedule = b.WorkingScheduleTable
		createdBy       = b.UserTable.WithAlias("crt")
		updatedBy       = b.UserTable.WithAlias("upd")
		team            = b.TeamTable
		calendar        = b.CalendarTable
		base            = b.Select().From(workingSchedule.String())

		join          = 0
		joinCreatedBy = func() {
			if join&linkCreatedBy != 0 {
				return
			}

			join |= linkCreatedBy
			base.JoinWithOption(
				b.LeftJoin(createdBy,
					b.JoinExpression{
						Left:  workingSchedule.Ident("created_by"),
						Op:    "=",
						Right: createdBy.Ident("id"),
					},
				),
			)
		}

		joinUpdatedBy = func() {
			if join&linkUpdatedBy != 0 {
				return
			}

			join |= linkUpdatedBy
			base.JoinWithOption(
				b.LeftJoin(updatedBy,
					b.JoinExpression{
						Left:  workingSchedule.Ident("updated_by"),
						Op:    "=",
						Right: updatedBy.Ident("id"),
					},
				),
			)
		}

		joinTeam = func() {
			if join&linkTeam != 0 {
				return
			}

			join |= linkTeam
			base.JoinWithOption(
				b.LeftJoin(createdBy,
					b.JoinExpression{
						Left:  workingSchedule.Ident("team_id"),
						Op:    "=",
						Right: team.Ident("id"),
					},
				),
			)
		}

		joinCalendar = func() {
			if join&linkCalendar != 0 {
				return
			}

			join |= linkCalendar
			base.JoinWithOption(
				b.LeftJoin(createdBy,
					b.JoinExpression{
						Left:  workingSchedule.Ident("calendar_id"),
						Op:    "=",
						Right: calendar.Ident("id"),
					},
				),
			)
		}
	)

	{
		// Default fields
		for _, field := range []string{"id", "name", "state", "created_at", "created_by", "updated_at", "updated_by"} {
			search.WithField(field)
		}

		for _, field := range search.Fields() {
			switch field {
			case "id", "domain_id", "created_at", "updated_at", "name", "state", "start_date_at", "end_date_at",
				"start_time_at", "end_time_at", "block_outside_activity":
				field = workingSchedule.Ident(field)

			case "created_by":
				joinCreatedBy()
				field = b.Alias(b.JSONBuildObject(b.UserLookup(createdBy)), field)

			case "updated_by":
				joinUpdatedBy()
				field = b.Alias(b.JSONBuildObject(b.UserLookup(updatedBy)), field)

			case "team":
				joinTeam()
				field = b.Alias(b.JSONBuildObject(b.Lookup(team, "id", "name")), field)

			case "calendar":
				joinCalendar()
				field = b.Alias(b.JSONBuildObject(b.Lookup(calendar, "id", "name")), field)

			case "agents":
				{
					var (
						workingScheduleAgent = b.WorkingScheduleAgentTable
						agent                = b.AgentTable
						userAgent            = b.UserTable.WithAlias("au")
						agents               = b.Select().From(workingScheduleAgent.String())
					)

					agents.
						Select(agent.Ident("id"),
							b.Alias(b.Coalesce(userAgent.Ident("name"), userAgent.Ident("username")), "name")).
						JoinWithOption(b.LeftJoin(agent,
							b.JoinExpression{
								Left:  workingScheduleAgent.Ident("agent_id"),
								Op:    "=",
								Right: agent.Ident("id"),
							},
						)).
						JoinWithOption(b.LeftJoin(userAgent,
							b.JoinExpression{
								Left:  agent.Ident("user_id"),
								Op:    "=",
								Right: userAgent.Ident("id"),
							},
						)).
						Where(fmt.Sprintf("%s = %s", workingSchedule.Ident("id"), workingScheduleAgent.Ident("working_schedule_id")))

					// SELECT json_agg(row_to_json(agents))
					// FROM (SELECT id, name FROM ...) AS agents
					agentsJSON := b.Select("json_agg(row_to_json(agents))")
					agentsJSON.From(agentsJSON.BuilderAs(agents, field))

					field = base.BuilderAs(agentsJSON, field)
				}

			case "extra_skills":
				{
					var (
						workingScheduleExtraSkill = b.WorkingScheduleExtraSkillTable
						skill                     = b.SkillTable
						skills                    = b.Select().From(workingScheduleExtraSkill.String())
					)

					skills.
						Select(b.JSONBuildObject(b.Lookup(skill, "id", "name"))).
						JoinWithOption(b.LeftJoin(skill,
							b.JoinExpression{
								Left:  workingScheduleExtraSkill.Ident("agent_id"),
								Op:    "=",
								Right: skill.Ident("id"),
							},
						)).
						Where(fmt.Sprintf("%s = %s", workingSchedule.Ident("id"), workingScheduleExtraSkill.Ident("working_schedule_id")))

					skillsJSON := b.Select("json_agg(skills)")
					skillsJSON.From(skillsJSON.BuilderAs(skills, field))

					field = base.BuilderAs(skillsJSON, field)
				}
			}

			base.SelectMore(field)
		}
	}

	{
		base.Where(base.EQ(workingSchedule.Ident("domain_id"), search.User().DomainId))
		if search.Query() != "" {
			base.Where(base.ILike(workingSchedule.Ident("name"), search.Query()))
		}

		if ids := search.IDs(); len(ids) > 0 {
			base.Where(base.In(workingSchedule.Ident("id"), b.ConvertArgs(ids)...))
		}
	}

	{
		orderBy := search.OrderBy()
		if len(orderBy) == 0 {
			orderBy.WithOrderBy("created_at", b.OrderDirectionASC)
		}

		for field, direction := range orderBy {
			switch field {
			case "id", "name", "description", "created_at", "updated_at":
				field = b.OrderBy(workingSchedule.Ident(field), direction)

			case "created_by":
				joinCreatedBy()
				field = b.OrderBy(createdBy.Ident("name"), direction)

			case "updated_by":
				joinUpdatedBy()
				field = b.OrderBy(updatedBy.Ident("name"), direction)
			}

			base.OrderBy(field)
		}
	}

	var items []*model.WorkingSchedule
	sql, args := base.Limit(search.Size()).Offset(search.Offset()).Build()
	if err := w.db.StandbyPreferred().Select(ctx, &items, sql, args...); err != nil {
		return nil, err
	}

	return items, nil
}

func (w *WorkingSchedule) UpdateWorkingSchedule(ctx context.Context, read *options.Read, in *model.WorkingSchedule) error {
	cteq := b.CTE()
	schedule := map[string]any{
		"updated_by":             read.User().Id,
		"name":                   in.Name,
		"block_outside_activity": in.BlockOutsideActivity,
	}

	cteq.With(b.With("schedule").As(b.Update(b.WorkingScheduleTable.Name(), schedule).SQL("RETURNING id")))

	del := b.Delete(b.WorkingScheduleExtraSkillTable.Name())
	del.Where(del.Equal("domain_id", read.User().DomainId), del.Equal("working_schedule_id", in.Id)).SQL("RETURNING id")
	cteq.With(b.With("del_extra_skills").As(del))

	if c := len(in.ExtraSkills); c > 0 {
		skills := make([]map[string]any, 0, len(in.ExtraSkills))
		for _, s := range in.ExtraSkills {
			skill := map[string]any{
				"domain_id":           read.User().DomainId,
				"working_schedule_id": b.Format("(SELECT id FROM schedule)::bigint"),
				"skill_id":            s.SafeId(),
			}

			skills = append(skills, skill)
		}

		cteq.With(b.With("extra_skills").As(b.Insert(b.WorkingScheduleExtraSkillTable.Name(), skills).SQL("RETURNING id")))
	}

	cte := cteq.Builder()
	sql, args := b.Select("schedule.id").Distinct().With(cte).From(cte.TableNames()...).Build()
	if err := w.db.Primary().Exec(ctx, sql, args...); err != nil {
		return err
	}

	return nil
}

func (w *WorkingSchedule) DeleteWorkingSchedule(ctx context.Context, read *options.Read) (int64, error) {
	db := b.Delete(b.WorkingScheduleTable.Name())
	clauses := []string{
		db.Equal("domain_id", read.User().DomainId),
		db.Equal("id", read.ID()),
	}

	sql, args := db.Where(clauses...).Build()
	if err := w.db.Primary().Exec(ctx, sql, args...); err != nil {
		return 0, err
	}

	return read.ID(), nil
}

func (w *WorkingSchedule) UpdateWorkingScheduleAddAgents(ctx context.Context, user *model.SignedInUser, id int64, agentIds []int64) error {
	columns := make([]map[string]any, 0, len(agentIds))
	for _, agentId := range agentIds {
		columns = append(columns, map[string]any{
			"domain_id":           user.DomainId,
			"working_schedule_id": id,
			"agent_id":            agentId,
		})
	}

	sql, args := b.Insert(b.WorkingScheduleAgentTable.Name(), columns).Build()
	if err := w.db.Primary().Exec(ctx, sql, args...); err != nil {
		return err
	}

	return nil
}

func (w *WorkingSchedule) UpdateWorkingScheduleRemoveAgent(ctx context.Context, read *options.Read, agentId int64) (int64, error) {
	db := b.Delete(b.WorkingScheduleAgentTable.Name())
	sql, args := db.Where(db.Equal("domain_id", read.User().DomainId), db.Equal("working_schedule_id", read.ID()), db.Equal("agent_id", agentId)).Build()
	if err := w.db.Primary().Exec(ctx, sql, args...); err != nil {
		return 0, err
	}

	return agentId, nil
}
