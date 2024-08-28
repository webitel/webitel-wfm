package storage

import (
	"context"

	"github.com/webitel/webitel-wfm/infra/storage/dbsql"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql/cluster"
	"github.com/webitel/webitel-wfm/internal/model"
)

const (
	agentWorkingConditionsTable = "wfm.agent_working_conditions"
	agentWorkingConditionsView  = agentWorkingConditionsTable + "_v"
)

type AgentWorkingConditions struct {
	db cluster.Store
}

func NewAgentWorkingConditions(db cluster.Store) *AgentWorkingConditions {
	return &AgentWorkingConditions{
		db: db,
	}
}

func (a *AgentWorkingConditions) ReadAgentWorkingConditions(ctx context.Context, user *model.SignedInUser, agentId int64) (*model.AgentWorkingConditions, error) {
	var item model.AgentWorkingConditions

	columns := []string{dbsql.Wildcard(model.AgentWorkingConditions{})}
	sb := a.db.SQL().Select(columns...).From(agentWorkingConditionsView)
	sql, args := sb.Where(sb.Equal("domain_id", user.DomainId), sb.Equal("agent_id", agentId)).Build()
	if err := a.db.StandbyPreferred().Get(ctx, &item, sql, args...); err != nil {
		return nil, err
	}

	return &item, nil
}

func (a *AgentWorkingConditions) UpdateAgentWorkingConditions(ctx context.Context, user *model.SignedInUser, agentId int64, in *model.AgentWorkingConditions) error {
	columns := map[string]any{
		"domain_id":            user.DomainId,
		"updated_by":           user.Id,
		"agent_id":             agentId,
		"working_condition_id": in.WorkingCondition.Id,
		"pause_template_id":    in.PauseTemplate.Id,
	}

	sql, args := a.db.SQL().Insert(agentWorkingConditionsTable, columns).Build()
	if err := a.db.Primary().Exec(ctx, sql, args...); err != nil {
		return err
	}

	return nil
}
