package service

import (
	"context"

	"github.com/webitel/webitel-wfm/infra/webitel/engine"
	"github.com/webitel/webitel-wfm/internal/model"
	"github.com/webitel/webitel-wfm/pkg/compare"
	"github.com/webitel/webitel-wfm/pkg/werror"
)

type WorkingScheduleStorage interface {
	CreateWorkingSchedule(ctx context.Context, user *model.SignedInUser, in *model.WorkingSchedule) (*model.WorkingSchedule, error)
	ReadWorkingSchedule(ctx context.Context, user *model.SignedInUser, search *model.SearchItem) (*model.WorkingSchedule, error)
	SearchWorkingSchedule(ctx context.Context, user *model.SignedInUser, search *model.SearchItem) ([]*model.WorkingSchedule, error)
	UpdateWorkingSchedule(ctx context.Context, user *model.SignedInUser, in *model.WorkingSchedule) (*model.WorkingSchedule, error)
	DeleteWorkingSchedule(ctx context.Context, user *model.SignedInUser, id int64) (int64, error)

	UpdateWorkingScheduleAddAgents(ctx context.Context, user *model.SignedInUser, id int64, agentIds []int64) ([]*model.LookupItem, error)
	UpdateWorkingScheduleRemoveAgents(ctx context.Context, user *model.SignedInUser, id int64, agentIds []int64) ([]*model.LookupItem, error)
}

type WorkingSchedule struct {
	storage WorkingScheduleStorage
	engine  *engine.Client
}

func NewWorkingSchedule(storage WorkingScheduleStorage, engine *engine.Client) *WorkingSchedule {
	return &WorkingSchedule{
		storage: storage,
		engine:  engine,
	}
}

func (w *WorkingSchedule) CreateWorkingSchedule(ctx context.Context, user *model.SignedInUser, in *model.WorkingSchedule) (*model.WorkingSchedule, error) {
	agentIds, err := w.engine.Agents(ctx, &model.AgentSearch{TeamIds: []int64{in.Team.Id}})
	if err != nil {
		return nil, err
	}

	agents := make([]*model.LookupItem, 0, len(agentIds))
	for _, a := range agentIds {
		agents = append(agents, &model.LookupItem{Id: a})
	}

	in.Agents = agents
	out, err := w.storage.CreateWorkingSchedule(ctx, user, in)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (w *WorkingSchedule) ReadWorkingSchedule(ctx context.Context, user *model.SignedInUser, search *model.SearchItem) (*model.WorkingSchedule, error) {
	out, err := w.storage.ReadWorkingSchedule(ctx, user, search)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (w *WorkingSchedule) SearchWorkingSchedule(ctx context.Context, user *model.SignedInUser, search *model.SearchItem) ([]*model.WorkingSchedule, bool, error) {
	out, err := w.storage.SearchWorkingSchedule(ctx, user, search)
	if err != nil {
		return nil, false, err
	}

	next, out := model.ListResult(search.Limit(), out)

	return out, next, nil
}

func (w *WorkingSchedule) UpdateWorkingSchedule(ctx context.Context, user *model.SignedInUser, in *model.WorkingSchedule) (*model.WorkingSchedule, error) {
	item, err := w.ReadWorkingSchedule(ctx, user, &model.SearchItem{Id: in.Id})
	if err != nil {
		return nil, err
	}

	if item.State != model.WorkingScheduleStateDraft {
		return nil, werror.NewWorkingScheduleUpdateDraftErr("service.working_schedule.state", item.State.String())
	}

	if item.Team.Id != in.Team.Id || item.Calendar.Id != in.Calendar.Id {
		if _, err := w.DeleteWorkingSchedule(ctx, user, item.Id); err != nil {
			return nil, err
		}

		out, err := w.CreateWorkingSchedule(ctx, user, item)
		if err != nil {
			return nil, err
		}

		return out, nil
	}

	out, err := w.storage.UpdateWorkingSchedule(ctx, user, in)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (w *WorkingSchedule) DeleteWorkingSchedule(ctx context.Context, user *model.SignedInUser, id int64) (int64, error) {
	out, err := w.storage.DeleteWorkingSchedule(ctx, user, id)
	if err != nil {
		return 0, err
	}

	return out, nil
}

func (w *WorkingSchedule) UpdateWorkingScheduleAddAgents(ctx context.Context, user *model.SignedInUser, id int64, agentIds []int64) ([]*model.LookupItem, error) {
	agents, err := w.engine.Agents(ctx, &model.AgentSearch{Ids: agentIds})
	if err != nil {
		return nil, err
	}

	// Checks if signed user has read access to a desired set of agents.
	if ok := compare.ElementsMatch(agents, agentIds); !ok {
		return nil, werror.NewAuthForbiddenError("service.working_schedule.check_agents", "cc_agent", "read")
	}

	out, err := w.storage.UpdateWorkingScheduleAddAgents(ctx, user, id, agentIds)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (w *WorkingSchedule) UpdateWorkingScheduleRemoveAgents(ctx context.Context, user *model.SignedInUser, id int64, agentIds []int64) ([]*model.LookupItem, error) {
	agents, err := w.engine.Agents(ctx, &model.AgentSearch{Ids: agentIds})
	if err != nil {
		return nil, err
	}

	// Checks if signed user has read access to a desired set of agents.
	if ok := compare.ElementsMatch(agents, agentIds); !ok {
		return nil, werror.NewAuthForbiddenError("service.working_schedule.check_agents", "cc_agent", "read")
	}

	out, err := w.storage.UpdateWorkingScheduleRemoveAgents(ctx, user, id, agentIds)
	if err != nil {
		return nil, err
	}

	return out, nil
}
