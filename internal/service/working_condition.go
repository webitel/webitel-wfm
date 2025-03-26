package service

import (
	"context"

	"github.com/webitel/webitel-wfm/internal/model"
	"github.com/webitel/webitel-wfm/internal/model/options"
	"github.com/webitel/webitel-wfm/internal/storage"
)

type WorkingConditionManager interface {
	CreateWorkingCondition(ctx context.Context, read *options.Read, in *model.WorkingCondition) (*model.WorkingCondition, error)
	ReadWorkingCondition(ctx context.Context, read *options.Read) (*model.WorkingCondition, error)
	SearchWorkingCondition(ctx context.Context, search *options.Search) ([]*model.WorkingCondition, bool, error)
	UpdateWorkingCondition(ctx context.Context, read *options.Read, in *model.WorkingCondition) (*model.WorkingCondition, error)
	DeleteWorkingCondition(ctx context.Context, read *options.Read) (int64, error)
}
type WorkingCondition struct {
	storage storage.WorkingConditionManager
}

func NewWorkingCondition(storage storage.WorkingConditionManager) *WorkingCondition {
	return &WorkingCondition{
		storage: storage,
	}
}

func (w *WorkingCondition) CreateWorkingCondition(ctx context.Context, read *options.Read, in *model.WorkingCondition) (*model.WorkingCondition, error) {
	id, err := w.storage.CreateWorkingCondition(ctx, read.User(), in)
	if err != nil {
		return nil, err
	}

	return w.ReadWorkingCondition(ctx, read.WithID(id))
}

func (w *WorkingCondition) ReadWorkingCondition(ctx context.Context, read *options.Read) (*model.WorkingCondition, error) {
	item, err := w.storage.ReadWorkingCondition(ctx, read)
	if err != nil {
		return nil, err
	}

	return item, nil
}

func (w *WorkingCondition) SearchWorkingCondition(ctx context.Context, search *options.Search) ([]*model.WorkingCondition, bool, error) {
	out, err := w.storage.SearchWorkingCondition(ctx, search)
	if err != nil {
		return nil, false, err
	}

	next, out := model.ListResult(int32(search.Size()), out)

	return out, next, nil
}

func (w *WorkingCondition) UpdateWorkingCondition(ctx context.Context, read *options.Read, in *model.WorkingCondition) (*model.WorkingCondition, error) {
	if err := w.storage.UpdateWorkingCondition(ctx, read.User(), in); err != nil {
		return nil, err
	}

	return w.ReadWorkingCondition(ctx, read)
}

func (w *WorkingCondition) DeleteWorkingCondition(ctx context.Context, read *options.Read) (int64, error) {
	out, err := w.storage.DeleteWorkingCondition(ctx, read)
	if err != nil {
		return 0, err
	}

	return out, nil
}
