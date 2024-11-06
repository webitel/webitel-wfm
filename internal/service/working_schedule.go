package service

import (
	"context"

	"github.com/webitel/webitel-wfm/infra/webitel/engine"
	"github.com/webitel/webitel-wfm/internal/model"
)

type WorkingScheduleStorage interface {
	CreateWorkingSchedule(ctx context.Context, user *model.SignedInUser, in *model.WorkingSchedule) (*model.WorkingSchedule, error)
	ReadWorkingSchedule(ctx context.Context, user *model.SignedInUser, search *model.SearchItem) (*model.WorkingSchedule, error)
	SearchWorkingSchedule(ctx context.Context, user *model.SignedInUser, search *model.SearchItem) ([]*model.WorkingSchedule, error)
	UpdateWorkingSchedule(ctx context.Context, user *model.SignedInUser, in *model.WorkingSchedule) (*model.WorkingSchedule, error)
	DeleteWorkingSchedule(ctx context.Context, user *model.SignedInUser, id int64) (int64, error)
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
	// TODO implement me
	panic("implement me")
}

func (w *WorkingSchedule) DeleteWorkingSchedule(ctx context.Context, user *model.SignedInUser, id int64) (int64, error) {
	// TODO implement me
	panic("implement me")
}
