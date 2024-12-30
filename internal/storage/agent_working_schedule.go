package storage

import (
	"context"

	"github.com/webitel/webitel-wfm/infra/storage/cache"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql/builder"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql/cluster"
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
	CreateAgentsWorkingScheduleShifts(ctx context.Context, user *model.SignedInUser, workingScheduleID int64, in []*model.AgentWorkingSchedule) ([]*model.AgentWorkingSchedule, error)
	SearchAgentWorkingSchedule(ctx context.Context, user *model.SignedInUser, search *model.AgentWorkingScheduleSearch) ([]*model.AgentWorkingSchedule, error)
	Holidays(ctx context.Context, user *model.SignedInUser, search *model.AgentWorkingScheduleSearch) ([]*model.Holiday, error)
}

type AgentWorkingSchedule struct {
	db    cluster.Store
	cache *cache.Scope[model.AgentWorkingSchedule]
}

func NewAgentWorkingSchedule(db cluster.Store, manager cache.Manager) *AgentWorkingSchedule {
	return &AgentWorkingSchedule{
		db:    db,
		cache: cache.NewScope[model.AgentWorkingSchedule](manager, agentWorkingScheduleTable),
	}
}

func (a *AgentWorkingSchedule) CreateAgentsWorkingScheduleShifts(ctx context.Context, user *model.SignedInUser, workingScheduleID int64, in []*model.AgentWorkingSchedule) ([]*model.AgentWorkingSchedule, error) {
	batch := a.db.Primary().Batch()
	for _, agent := range in {
		for _, shift := range agent.Schedule {
			columns := []map[string]any{
				{
					"domain_id":                 user.DomainId,
					"created_by":                user.Id,
					"working_schedule_agent_id": builder.Format("(SELECT id::bigint FROM wfm.working_schedule_agent WHERE domain_id = $? AND working_schedule_id = $? AND agent_id = $?)", user.DomainId, workingScheduleID, agent.Agent.Id),
					"schedule_at":               shift.Date,
					"start_min":                 shift.Shift.Start,
					"end_min":                   shift.Shift.End,
				},
			}

			cte := builder.CTE(builder.With("schedule").As(builder.Insert(agentWorkingScheduleTable, columns).SQL("RETURNING id")))
			if l := len(shift.Shift.Pauses); l > 0 {
				pauses := make([]map[string]any, l)
				for _, pause := range shift.Shift.Pauses {
					pauses = append(pauses, map[string]any{
						"domain_id":                 user.DomainId,
						"created_by":                user.Id,
						"agent_working_schedule_id": builder.Format("(SELECT id::bigint FROM schedule)"),
						"pause_cause_id":            pause.Cause.SafeId(),
						"start_min":                 pause.Start,
						"end_min":                   pause.End,
					})
				}

				cte.With(builder.With("pauses").As(builder.Insert(agentWorkingSchedulePauseTable, pauses).SQL("RETURNING id")))
			}

			if l := len(shift.Shift.Skills); l > 0 {
				skills := make([]map[string]any, l)
				for _, skill := range shift.Shift.Skills {
					skills = append(skills, map[string]any{
						"domain_id":                 user.DomainId,
						"agent_working_schedule_id": builder.Format("(SELECT id::bigint FROM schedule)"),
						"skill_id":                  skill.Skill.Id,
						"capacity":                  skill.Capacity,
					})
				}

				cte.With(builder.With("skills").As(builder.Insert(agentWorkingScheduleSkillTable, skills).SQL("RETURNING id")))
			}

			sql, args := builder.Select("distinct schedule.id").With(cte.Builder()).From(cte.Tables()...).Build()
			batch.Queue(sql, args...)
		}
	}

	var ids []int64
	if err := batch.Select(ctx, &ids); err != nil {
		return nil, err
	}

	out, err := a.SearchAgentWorkingSchedule(ctx, user, &model.AgentWorkingScheduleSearch{WorkingScheduleId: workingScheduleID, Ids: ids})
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (a *AgentWorkingSchedule) SearchAgentWorkingSchedule(ctx context.Context, user *model.SignedInUser, search *model.AgentWorkingScheduleSearch) ([]*model.AgentWorkingSchedule, error) {
	sb := builder.Select("agent AS agent", "jsonb_agg(schedule.*) FILTER (WHERE date NOTNULL) AS schedule")
	if len(search.AgentIds) > 0 {
		in := make([]any, 0, len(search.AgentIds))
		for _, id := range search.AgentIds {
			in = append(in, id)
		}

		sb.Where(sb.In("(agent ->> 'id')::bigint", in...))
	}

	if search.SearchItem.Search != nil {
		sb.Where(sb.ILike("(agent ->> 'name')::text", search.SearchItem.SearchBy()))
	}

	if len(search.Ids) > 0 {
		in := make([]any, 0, len(search.Ids))
		for _, id := range search.Ids {
			in = append(in, id)
		}

		sb.Where(sb.In("(shift ->> 'id')::bigint", in...))
	}

	if s := search.SearchItem.Date; s != nil {
		var and []string
		if s.From.Valid {
			and = append(and, sb.GreaterEqualThan("date", s.From))
		}

		if s.To.Valid {
			and = append(and, sb.LessEqualThan("date", s.To))
		}

		sb.Or(sb.IsNull("date"), sb.And(and...))
	}

	sql, args := sb.From(agentWorkingScheduleView+" AS schedule").
		Where(sb.Equal("domain_id", user.DomainId),
			sb.Equal("working_schedule_id", search.WorkingScheduleId),
		).
		GroupBy("agent").
		Build()

	var items []*model.AgentWorkingSchedule
	if err := a.db.StandbyPreferred().Select(ctx, &items, sql, args...); err != nil {
		return nil, err
	}

	return items, nil
}

func (a *AgentWorkingSchedule) Holidays(ctx context.Context, user *model.SignedInUser, search *model.AgentWorkingScheduleSearch) ([]*model.Holiday, error) {
	sb := builder.Select("date", "name").From(agentWorkingScheduleHolidaysView)
	sql, args := sb.Where(sb.Equal("domain_id", user.DomainId),
		sb.Equal("working_schedule_id", search.WorkingScheduleId),
		sb.Between("date", search.SearchItem.Date.From, search.SearchItem.Date.To)).Build()

	var items []*model.Holiday
	if err := a.db.StandbyPreferred().Select(ctx, &items, sql, args...); err != nil {
		return nil, err
	}

	return items, nil
}
