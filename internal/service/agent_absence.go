package service

import (
	"context"

	"github.com/webitel/webitel-wfm/infra/webitel/engine"
	"github.com/webitel/webitel-wfm/internal/model"
	"github.com/webitel/webitel-wfm/pkg/compare"
	"github.com/webitel/webitel-wfm/pkg/werror"
)

type AgentAbsenceManager interface {
	CreateAgentAbsence(ctx context.Context, user *model.SignedInUser, in *model.AgentAbsence) (int64, error)
	UpdateAgentAbsence(ctx context.Context, user *model.SignedInUser, in *model.AgentAbsence) error
	DeleteAgentAbsence(ctx context.Context, user *model.SignedInUser, agentId, id int64) error

	CreateAgentsAbsencesBulk(ctx context.Context, user *model.SignedInUser, agentIds []int64, in []*model.AgentAbsenceBulk) ([]int64, error)
	SearchAgentsAbsences(ctx context.Context, user *model.SignedInUser, search *model.AgentAbsenceSearch) ([]*model.AgentAbsences, error)
}

type AgentAbsence struct {
	store  AgentAbsenceManager
	engine *engine.Client
}

func NewAgentAbsence(store AgentAbsenceManager, engine *engine.Client) *AgentAbsence {
	return &AgentAbsence{
		store:  store,
		engine: engine,
	}
}

func (a *AgentAbsence) CreateAgentAbsence(ctx context.Context, user *model.SignedInUser, in *model.AgentAbsence) (int64, error) {
	_, err := a.engine.Agent(ctx, in.Agent.Id)
	if err != nil {
		return 0, err
	}

	id, err := a.store.CreateAgentAbsence(ctx, user, in)
	if err != nil {
		return 0, err
	}

	return id, nil
}

func (a *AgentAbsence) ReadAgentAbsence(ctx context.Context, user *model.SignedInUser, agentId int64, search *model.SearchItem) (*model.AgentAbsence, error) {
	items, err := a.ReadAgentAbsences(ctx, user, &model.AgentAbsenceSearch{SearchItem: *search, AgentIds: []int64{agentId}})
	if err != nil {
		return nil, err
	}

	if len(items.Absence) > 1 {
		return nil, werror.NewDBEntityConflictError("service.agent_absence.read.conflict")
	}

	if len(items.Absence) == 0 {
		return nil, werror.NewDBNoRowsErr("service.agent_absence.read")
	}

	return &model.AgentAbsence{Agent: items.Agent, Absence: items.Absence[0]}, nil
}

func (a *AgentAbsence) UpdateAgentAbsence(ctx context.Context, user *model.SignedInUser, in *model.AgentAbsence) error {
	_, err := a.engine.Agent(ctx, in.Agent.Id)
	if err != nil {
		return err
	}

	if err := a.store.UpdateAgentAbsence(ctx, user, in); err != nil {
		return err
	}

	return nil
}

func (a *AgentAbsence) DeleteAgentAbsence(ctx context.Context, user *model.SignedInUser, agentId, id int64) error {
	_, err := a.engine.Agent(ctx, agentId)
	if err != nil {
		return err
	}

	if err := a.store.DeleteAgentAbsence(ctx, user, agentId, id); err != nil {
		return err
	}

	return nil
}

func (a *AgentAbsence) ReadAgentAbsences(ctx context.Context, user *model.SignedInUser, search *model.AgentAbsenceSearch) (*model.AgentAbsences, error) {
	_, err := a.engine.Agent(ctx, search.AgentIds[0])
	if err != nil {
		return nil, err
	}

	items, err := a.store.SearchAgentsAbsences(ctx, user, search)
	if err != nil {
		return nil, err
	}

	if len(items) > 1 {
		return nil, werror.NewDBEntityConflictError("service.agent_absence.read.conflict")
	}

	if len(items) == 0 {
		return nil, werror.NewDBNoRowsErr("service.agent_absence.read")
	}

	return items[0], nil
}

func (a *AgentAbsence) CreateAgentsAbsencesBulk(ctx context.Context, user *model.SignedInUser, agentIds []int64, in []*model.AgentAbsenceBulk) ([]int64, error) {
	agents, err := a.engine.Agents(ctx, &model.AgentSearch{Ids: agentIds})
	if err != nil {
		return nil, err
	}

	// Checks if signed user has read access to a desired set of agents.
	if ok := compare.ElementsMatch(agents, agentIds); !ok {
		return nil, werror.NewAuthForbiddenError("service.agent_absence.check_agents", "cc_agent", "read")
	}

	ids, err := a.store.CreateAgentsAbsencesBulk(ctx, user, agents, in)
	if err != nil {
		return nil, err
	}

	return ids, nil
}

func (a *AgentAbsence) SearchAgentsAbsences(ctx context.Context, user *model.SignedInUser, search *model.AgentAbsenceSearch) ([]*model.AgentAbsences, bool, error) {
	s := &model.AgentSearch{
		SupervisorIds: search.SupervisorIds,
		TeamIds:       search.TeamIds,
		SkillIds:      search.SkillIds,
	}

	agents, err := a.engine.Agents(ctx, s)
	if err != nil {
		return nil, false, err
	}

	search.AgentIds = agents
	items, err := a.store.SearchAgentsAbsences(ctx, user, search)
	if err != nil {
		return nil, false, err
	}

	var next bool
	if len(items) == int(search.SearchItem.Limit()) {
		next = true
		items = items[:search.SearchItem.Limit()-1]
	}

	return items, next, nil
}
