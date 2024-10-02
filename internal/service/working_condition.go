package service

import (
	"context"

	"github.com/webitel/webitel-wfm/internal/model"
	"github.com/webitel/webitel-wfm/pkg/werror"
)

type WorkingConditionManager interface {
	CreateWorkingCondition(ctx context.Context, user *model.SignedInUser, in *model.WorkingCondition) (int64, error)
	SearchWorkingCondition(ctx context.Context, user *model.SignedInUser, search *model.SearchItem) ([]*model.WorkingCondition, error)
	UpdateWorkingCondition(ctx context.Context, user *model.SignedInUser, in *model.WorkingCondition) error
	DeleteWorkingCondition(ctx context.Context, user *model.SignedInUser, id int64) (int64, error)
}

type WorkingCondition struct {
	store WorkingConditionManager
}

func NewWorkingCondition(store WorkingConditionManager) *WorkingCondition {
	return &WorkingCondition{
		store: store,
	}
}

func (w *WorkingCondition) CreateWorkingCondition(ctx context.Context, user *model.SignedInUser, in *model.WorkingCondition) (int64, error) {
	id, err := w.store.CreateWorkingCondition(ctx, user, in)
	if err != nil {
		return 0, err
	}

	// w.cache.DeleteMany(w.cache.KeyPrefix(search.DomainId, WorkingConditionCacheScope, 0))

	return id, nil
}

func (w *WorkingCondition) ReadWorkingCondition(ctx context.Context, user *model.SignedInUser, search *model.SearchItem) (*model.WorkingCondition, error) {
	// fetchFn := func(ctx context.Context) ([]*model.WorkingCondition, error) {
	items, err := w.store.SearchWorkingCondition(ctx, user, search)
	if err != nil {
		return nil, err
	}

	// 	return items, nil
	// }

	// items, err := cache.GetOrFetch(ctx, w.cache.Raw(), w.cache.Key(search.DomainId, WorkingConditionCacheScope, search.Token, search), fetchFn)
	// if err != nil {
	// 	return nil, err
	// }

	if len(items) > 1 {
		return nil, werror.NewDBEntityConflictError("service.working_condition.read.conflict")
	}

	if len(items) == 0 {
		return nil, werror.NewDBNoRowsErr("service.working_condition.read")
	}

	return items[0], nil
}

func (w *WorkingCondition) SearchWorkingCondition(ctx context.Context, user *model.SignedInUser, search *model.SearchItem) ([]*model.WorkingCondition, bool, error) {
	// fetchFn := func(ctx context.Context) ([]*model.WorkingCondition, error) {
	out, err := w.store.SearchWorkingCondition(ctx, user, search)
	if err != nil {
		return nil, false, err
	}

	// 	return items, nil
	// }

	// items, err := cache.GetOrFetch(ctx, w.cache.Raw(), w.cache.Key(search.DomainId, WorkingConditionCacheScope, 0, search), fetchFn)
	// if err != nil {
	// 	return nil, false, err
	// }

	next, out := model.ListResult(search.Limit(), out)

	return out, next, nil
}

func (w *WorkingCondition) UpdateWorkingCondition(ctx context.Context, user *model.SignedInUser, in *model.WorkingCondition) error {
	if err := w.store.UpdateWorkingCondition(ctx, user, in); err != nil {
		return err
	}

	// w.cache.DeleteMany(w.cache.KeyPrefix(search.DomainId, WorkingConditionCacheScope, 0))

	return nil
}

func (w *WorkingCondition) DeleteWorkingCondition(ctx context.Context, user *model.SignedInUser, id int64) (int64, error) {
	out, err := w.store.DeleteWorkingCondition(ctx, user, id)
	if err != nil {
		return 0, err
	}

	// w.cache.DeleteMany(w.cache.KeyPrefix(search.DomainId, WorkingConditionCacheScope, 0))

	return out, nil
}
