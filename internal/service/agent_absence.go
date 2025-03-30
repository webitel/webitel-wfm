package service

import (
	"context"

	"github.com/webitel/webitel-wfm/infra/webitel/engine"
	"github.com/webitel/webitel-wfm/infra/webitel/logger"
	"github.com/webitel/webitel-wfm/internal/model"
	"github.com/webitel/webitel-wfm/internal/model/options"
	"github.com/webitel/webitel-wfm/internal/storage"
)

type AgentAbsenceManager interface {
	CreateAgentAbsence(ctx context.Context, read *options.Read, in *model.Absence) (*model.Absence, error)
	ReadAgentAbsence(ctx context.Context, read *options.Read) (*model.Absence, error)
	SearchAgentAbsence(ctx context.Context, search *options.Search) ([]*model.Absence, error)
	UpdateAgentAbsence(ctx context.Context, read *options.Read, in *model.Absence) (*model.Absence, error)
	DeleteAgentAbsence(ctx context.Context, read *options.Read) error

	CreateAgentsAbsences(ctx context.Context, search *options.Search, in []*model.AgentAbsences) ([]*model.AgentAbsences, error)
	SearchAgentsAbsences(ctx context.Context, search *options.Search) ([]*model.AgentAbsences, bool, error)
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

func (a *AgentAbsence) CreateAgentAbsence(ctx context.Context, read *options.Read, in *model.Absence) (*model.Absence, error) {
	id, err := a.storage.CreateAgentAbsence(ctx, read, in)
	if err != nil {
		return nil, err
	}

	read.WithID(id)

	return a.ReadAgentAbsence(ctx, read)
}

func (a *AgentAbsence) ReadAgentAbsence(ctx context.Context, read *options.Read) (*model.Absence, error) {
	return a.storage.ReadAgentAbsence(ctx, read)
}

func (a *AgentAbsence) SearchAgentAbsence(ctx context.Context, search *options.Search) ([]*model.Absence, error) {
	return a.storage.SearchAgentAbsence(ctx, search)
}

func (a *AgentAbsence) UpdateAgentAbsence(ctx context.Context, read *options.Read, in *model.Absence) (*model.Absence, error) {
	if err := a.storage.UpdateAgentAbsence(ctx, read, in); err != nil {
		return nil, err
	}

	return a.ReadAgentAbsence(ctx, read)
}

func (a *AgentAbsence) DeleteAgentAbsence(ctx context.Context, read *options.Read) error {
	if err := a.storage.DeleteAgentAbsence(ctx, read); err != nil {
		return err
	}

	return nil
}

func (a *AgentAbsence) CreateAgentsAbsences(ctx context.Context, search *options.Search, in []*model.AgentAbsences) ([]*model.AgentAbsences, error) {
	ids, err := a.storage.CreateAgentsAbsences(ctx, search, in)
	if err != nil {
		return nil, err
	}

	search.WithIDs(ids)
	out, _, err := a.SearchAgentsAbsences(ctx, search)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (a *AgentAbsence) SearchAgentsAbsences(ctx context.Context, search *options.Search) ([]*model.AgentAbsences, bool, error) {
	out, err := a.storage.SearchAgentsAbsences(ctx, search)
	if err != nil {
		return nil, false, err
	}

	next, out := model.ListResult(int32(search.Size()), out)

	return out, next, nil
}
