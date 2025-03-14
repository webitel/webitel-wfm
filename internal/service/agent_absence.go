package service

import (
	"context"

	"github.com/webitel/webitel-wfm/infra/webitel/engine"
	"github.com/webitel/webitel-wfm/infra/webitel/logger"
	"github.com/webitel/webitel-wfm/internal/model"
	"github.com/webitel/webitel-wfm/internal/storage"
	"github.com/webitel/webitel-wfm/pkg/compare"
	"github.com/webitel/webitel-wfm/pkg/werror"
)

type AgentAbsenceManager interface {
	CreateAgentAbsence(ctx context.Context, user *model.SignedInUser, in *model.AgentAbsence) (*model.AgentAbsence, error)
	UpdateAgentAbsence(ctx context.Context, user *model.SignedInUser, in *model.AgentAbsence) (*model.AgentAbsence, error)
	DeleteAgentAbsence(ctx context.Context, user *model.SignedInUser, agentId, id int64) error

	CreateAgentsAbsencesBulk(ctx context.Context, user *model.SignedInUser, agentIds []int64, in []*model.AgentAbsenceBulk) ([]*model.AgentAbsences, error)
	ReadAgentAbsences(ctx context.Context, user *model.SignedInUser, search *model.AgentAbsenceSearch) (*model.AgentAbsences, error)
	SearchAgentsAbsences(ctx context.Context, user *model.SignedInUser, search *model.AgentAbsenceSearch) ([]*model.AgentAbsences, bool, error)
}

type AgentAbsence struct {
	storage storage.AgentAbsenceManager
	audit   *logger.Audit
	engine  *engine.Client
}

func NewAgentAbsence(storage storage.AgentAbsenceManager, audit *logger.Audit, engine *engine.Client) *AgentAbsence {
	return &AgentAbsence{
		storage: storage,
		audit:   audit,
		engine:  engine,
	}
}

func (a *AgentAbsence) CreateAgentAbsence(ctx context.Context, user *model.SignedInUser, in *model.AgentAbsence) (*model.AgentAbsence, error) {
	_, err := a.engine.AgentService().Agent(ctx, in.Agent.Id)
	if err != nil {
		return nil, err
	}

	out, err := a.storage.CreateAgentAbsence(ctx, user, in)
	if err != nil {
		return nil, err
	}

	if err := a.audit.Create(ctx, user, map[int64]any{out.Absence.Id: out}); err != nil {
		return nil, err
	}

	return out, nil
}

func (a *AgentAbsence) UpdateAgentAbsence(ctx context.Context, user *model.SignedInUser, in *model.AgentAbsence) (*model.AgentAbsence, error) {
	_, err := a.engine.AgentService().Agent(ctx, in.Agent.Id)
	if err != nil {
		return nil, err
	}

	out, err := a.storage.UpdateAgentAbsence(ctx, user, in)
	if err != nil {
		return nil, err
	}

	if err := a.audit.Update(ctx, user, map[int64]any{out.Absence.Id: out}); err != nil {
		return nil, err
	}

	return out, nil
}

func (a *AgentAbsence) DeleteAgentAbsence(ctx context.Context, user *model.SignedInUser, agentId, id int64) error {
	_, err := a.engine.AgentService().Agent(ctx, agentId)
	if err != nil {
		return err
	}

	if err := a.storage.DeleteAgentAbsence(ctx, user, agentId, id); err != nil {
		return err
	}

	if err := a.audit.Delete(ctx, user, map[int64]any{id: nil}); err != nil {
		return err
	}

	return nil
}

func (a *AgentAbsence) ReadAgentAbsences(ctx context.Context, user *model.SignedInUser, search *model.AgentAbsenceSearch) (*model.AgentAbsences, error) {
	_, err := a.engine.AgentService().Agent(ctx, search.AgentIds[0])
	if err != nil {
		return nil, err
	}

	item, err := a.storage.ReadAgentAbsences(ctx, user, search)
	if err != nil {
		return nil, err
	}

	return item, nil
}

func (a *AgentAbsence) CreateAgentsAbsencesBulk(ctx context.Context, user *model.SignedInUser, agentIds []int64, in []*model.AgentAbsenceBulk) ([]*model.AgentAbsences, error) {
	agents, err := a.engine.AgentService().Agents(ctx, &model.AgentSearch{Ids: agentIds})
	if err != nil {
		return nil, err
	}

	// Checks if signed user has read access to a desired set of agents.
	if ok := compare.ElementsMatch(agents, agentIds); !ok {
		return nil, werror.Wrap(ErrAgentNotAllowed, werror.WithID("service.agent_absence.check_agents"))
	}

	out, err := a.storage.CreateAgentsAbsencesBulk(ctx, user, agents, in)
	if err != nil {
		return nil, err
	}

	records := make(map[int64]any)
	for _, item := range out {
		for _, absence := range item.Absence {
			records[absence.Id] = absence
		}
	}

	if err := a.audit.Create(ctx, user, records); err != nil {
		return nil, err
	}

	return out, nil
}

func (a *AgentAbsence) SearchAgentsAbsences(ctx context.Context, user *model.SignedInUser, search *model.AgentAbsenceSearch) ([]*model.AgentAbsences, bool, error) {
	s := &model.AgentSearch{
		SupervisorIds: search.SupervisorIds,
		TeamIds:       search.TeamIds,
		SkillIds:      search.SkillIds,
	}

	agents, err := a.engine.AgentService().Agents(ctx, s)
	if err != nil {
		return nil, false, err
	}

	search.AgentIds = agents
	out, err := a.storage.SearchAgentsAbsences(ctx, user, search)
	if err != nil {
		return nil, false, err
	}

	next, out := model.ListResult(search.SearchItem.Limit(), out)

	return out, next, nil
}
