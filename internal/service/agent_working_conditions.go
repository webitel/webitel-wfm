package service

import (
	"context"

	"github.com/webitel/webitel-wfm/infra/webitel/engine"
	"github.com/webitel/webitel-wfm/internal/model"
	"github.com/webitel/webitel-wfm/internal/model/options"
	"github.com/webitel/webitel-wfm/internal/storage"
)

type AgentWorkingConditionsManager interface {
	ReadAgentWorkingConditions(ctx context.Context, read *options.Read) (*model.AgentWorkingConditions, error)
	UpdateAgentWorkingConditions(ctx context.Context, read *options.Read, in *model.AgentWorkingConditions) (*model.AgentWorkingConditions, error)
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

func (a *AgentWorkingConditions) ReadAgentWorkingConditions(ctx context.Context, read *options.Read) (*model.AgentWorkingConditions, error) {
	_, err := a.engine.AgentService().Agent(ctx, read.ID())
	if err != nil {
		return nil, err
	}

	item, err := a.storage.ReadAgentWorkingConditions(ctx, read)
	if err != nil {
		return nil, err
	}

	return item, nil
}

func (a *AgentWorkingConditions) UpdateAgentWorkingConditions(ctx context.Context, read *options.Read, in *model.AgentWorkingConditions) (*model.AgentWorkingConditions, error) {
	_, err := a.engine.AgentService().Agent(ctx, read.ID())
	if err != nil {
		return nil, err
	}

	if err := a.storage.UpdateAgentWorkingConditions(ctx, read, in); err != nil {
		return nil, err
	}

	out, err := a.ReadAgentWorkingConditions(ctx, read)
	if err != nil {
		return nil, err
	}

	return out, nil
}
