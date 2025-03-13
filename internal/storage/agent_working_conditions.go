package storage

import (
	"context"

	"github.com/webitel/webitel-wfm/infra/storage/dbsql/builder"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql/cluster"
	"github.com/webitel/webitel-wfm/internal/model"
	"github.com/webitel/webitel-wfm/pkg/fields"
)

const (
	agentWorkingConditionsTable = "wfm.agent_working_conditions"
	agentWorkingConditionsView  = agentWorkingConditionsTable + "_v"
)

type AgentWorkingConditionsManager interface {
	ReadAgentWorkingConditions(ctx context.Context, user *model.SignedInUser, agentId int64) (*model.AgentWorkingConditions, error)
	UpdateAgentWorkingConditions(ctx context.Context, user *model.SignedInUser, agentId int64, in *model.AgentWorkingConditions) error
}

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

	columns := []string{fields.Wildcard(model.AgentWorkingConditions{})}
	sb := builder.Select(columns...).From(agentWorkingConditionsView)
	sql, args := sb.Where(sb.Equal("domain_id", user.DomainId), sb.Equal("(agent ->> 'id')::bigint", agentId)).Build()
	if err := a.db.StandbyPreferred().Get(ctx, &item, sql, args...); err != nil {
		return nil, err
	}

	return &item, nil
}

func (a *AgentWorkingConditions) UpdateAgentWorkingConditions(ctx context.Context, user *model.SignedInUser, agentId int64, in *model.AgentWorkingConditions) error {
	columns := []map[string]any{
		{
			"domain_id":            user.DomainId,
			"updated_by":           user.Id,
			"agent_id":             agentId,
			"working_condition_id": in.WorkingCondition.Id,
			"pause_template_id":    in.PauseTemplate.SafeId(),
		},
	}

	sql, args := builder.Insert(agentWorkingConditionsTable, columns).Build()
	if err := a.db.Primary().Exec(ctx, sql, args...); err != nil {
		return err
	}

	return nil
}
