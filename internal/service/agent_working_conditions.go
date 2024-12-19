package service

import (
	"context"

	"github.com/webitel/webitel-wfm/infra/webitel/engine"
	"github.com/webitel/webitel-wfm/internal/model"
	"github.com/webitel/webitel-wfm/internal/storage"
)

type AgentWorkingConditionsManager interface {
	ReadAgentWorkingConditions(ctx context.Context, user *model.SignedInUser, agentId int64) (*model.AgentWorkingConditions, error)
	UpdateAgentWorkingConditions(ctx context.Context, user *model.SignedInUser, agentId int64, in *model.AgentWorkingConditions) error
}
type AgentWorkingConditions struct {
	storage storage.AgentWorkingConditionsManager
	engine  *engine.Client
}

func NewAgentWorkingConditions(storage storage.AgentWorkingConditionsManager, engine *engine.Client) *AgentWorkingConditions {
	return &AgentWorkingConditions{
		storage: storage,
		engine:  engine,
	}
}

func (a *AgentWorkingConditions) ReadAgentWorkingConditions(ctx context.Context, user *model.SignedInUser, agentId int64) (*model.AgentWorkingConditions, error) {
	_, err := a.engine.Agent(ctx, agentId)
	if err != nil {
		return nil, err
	}

	item, err := a.storage.ReadAgentWorkingConditions(ctx, user, agentId)
	if err != nil {
		return nil, err
	}

	return item, nil
}

func (a *AgentWorkingConditions) UpdateAgentWorkingConditions(ctx context.Context, user *model.SignedInUser, agentId int64, in *model.AgentWorkingConditions) error {
	_, err := a.engine.Agent(ctx, agentId)
	if err != nil {
		return err
	}

	if err := a.storage.UpdateAgentWorkingConditions(ctx, user, agentId, in); err != nil {
		return err
	}

	return nil
}
