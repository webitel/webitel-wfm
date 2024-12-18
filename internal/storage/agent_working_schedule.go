package storage

import (
	"context"

	"github.com/webitel/webitel-wfm/infra/storage/cache"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql"
	"github.com/webitel/webitel-wfm/internal/model"
)

const (
	agentWorkingScheduleTable        = "wfm.agent_working_schedule"
	agentWorkingScheduleView         = agentWorkingScheduleTable + "_v"
	agentWorkingScheduleHolidaysView = agentWorkingScheduleTable + "_holidays_v"
)

type AgentWorkingSchedule struct {
	db    dbsql.Store
	cache *cache.Scope[model.AgentWorkingSchedule]
}

func NewAgentWorkingSchedule(db dbsql.Store, manager cache.Manager) *AgentWorkingSchedule {
	return &AgentWorkingSchedule{
		db:    db,
		cache: cache.NewScope[model.AgentWorkingSchedule](manager, agentWorkingScheduleTable),
	}
}

func (a *AgentWorkingSchedule) SearchAgentWorkingSchedule(ctx context.Context, user *model.SignedInUser, search *model.AgentWorkingScheduleSearch) ([]*model.AgentWorkingSchedule, error) {
	sb := a.db.SQL().Select("agent AS agent", "jsonb_agg(schedule.json) AS schedule")
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

	sql, args := sb.From(agentWorkingScheduleView,
		sb.LateralAs(
			a.db.SQL().Select("jsonb_build_object('date', date, 'type', type, 'absence', absence, 'shifts', shifts) as json"),
			"schedule",
		)).
		Where(sb.Equal("domain_id", user.DomainId),
			sb.Equal("working_schedule_id", search.WorkingScheduleId),
			sb.Between("date", search.SearchItem.Date.From, search.SearchItem.Date.To),
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
	sb := a.db.SQL().Select("date", "name").From(agentWorkingScheduleHolidaysView)
	sql, args := sb.Where(sb.Equal("domain_id", user.DomainId),
		sb.Equal("working_schedule_id", search.WorkingScheduleId),
		sb.Between("date", search.SearchItem.Date.From, search.SearchItem.Date.To)).Build()

	var items []*model.Holiday
	if err := a.db.StandbyPreferred().Select(ctx, &items, sql, args...); err != nil {
		return nil, err
	}

	return items, nil
}
