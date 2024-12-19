package service

import (
	"context"

	"github.com/webitel/webitel-wfm/internal/model"
	"github.com/webitel/webitel-wfm/internal/storage"
)

type WorkingConditionManager interface {
	CreateWorkingCondition(ctx context.Context, user *model.SignedInUser, in *model.WorkingCondition) (int64, error)
	ReadWorkingCondition(ctx context.Context, user *model.SignedInUser, search *model.SearchItem) (*model.WorkingCondition, error)
	SearchWorkingCondition(ctx context.Context, user *model.SignedInUser, search *model.SearchItem) ([]*model.WorkingCondition, bool, error)
	UpdateWorkingCondition(ctx context.Context, user *model.SignedInUser, in *model.WorkingCondition) error
	DeleteWorkingCondition(ctx context.Context, user *model.SignedInUser, id int64) (int64, error)
}
type WorkingCondition struct {
	storage storage.WorkingConditionManager
}

func NewWorkingCondition(storage storage.WorkingConditionManager) *WorkingCondition {
	return &WorkingCondition{
		storage: storage,
	}
}

func (w *WorkingCondition) CreateWorkingCondition(ctx context.Context, user *model.SignedInUser, in *model.WorkingCondition) (int64, error) {
	id, err := w.storage.CreateWorkingCondition(ctx, user, in)
	if err != nil {
		return 0, err
	}

	return id, nil
}

func (w *WorkingCondition) ReadWorkingCondition(ctx context.Context, user *model.SignedInUser, search *model.SearchItem) (*model.WorkingCondition, error) {
	item, err := w.storage.ReadWorkingCondition(ctx, user, search)
	if err != nil {
		return nil, err
	}

	return item, nil
}

func (w *WorkingCondition) SearchWorkingCondition(ctx context.Context, user *model.SignedInUser, search *model.SearchItem) ([]*model.WorkingCondition, bool, error) {
	out, err := w.storage.SearchWorkingCondition(ctx, user, search)
	if err != nil {
		return nil, false, err
	}

	next, out := model.ListResult(search.Limit(), out)

	return out, next, nil
}

func (w *WorkingCondition) UpdateWorkingCondition(ctx context.Context, user *model.SignedInUser, in *model.WorkingCondition) error {
	if err := w.storage.UpdateWorkingCondition(ctx, user, in); err != nil {
		return err
	}

	return nil
}

func (w *WorkingCondition) DeleteWorkingCondition(ctx context.Context, user *model.SignedInUser, id int64) (int64, error) {
	out, err := w.storage.DeleteWorkingCondition(ctx, user, id)
	if err != nil {
		return 0, err
	}

	return out, nil
}
