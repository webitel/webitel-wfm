package service

import (
	"context"

	"github.com/webitel/webitel-wfm/internal/model"
	"github.com/webitel/webitel-wfm/pkg/werror"
)

type ShiftTemplateManager interface {
	CreateShiftTemplate(ctx context.Context, user *model.SignedInUser, in *model.ShiftTemplate) (int64, error)
	SearchShiftTemplate(ctx context.Context, user *model.SignedInUser, search *model.SearchItem) ([]*model.ShiftTemplate, error)
	UpdateShiftTemplate(ctx context.Context, user *model.SignedInUser, in *model.ShiftTemplate) error
	DeleteShiftTemplate(ctx context.Context, user *model.SignedInUser, id int64) (int64, error)

	SearchShiftTemplateTime(ctx context.Context, user *model.SignedInUser, shiftTemplateId int64, search *model.SearchItem) ([]*model.ShiftTemplateTime, error)
	UpdateShiftTemplateTimeBulk(ctx context.Context, user *model.SignedInUser, shiftTemplateId int64, in []*model.ShiftTemplateTime) error
}

type ShiftTemplate struct {
	store ShiftTemplateManager
}

func NewShiftTemplate(store ShiftTemplateManager) *ShiftTemplate {
	return &ShiftTemplate{
		store: store,
	}
}

func (s *ShiftTemplate) CreateShiftTemplate(ctx context.Context, user *model.SignedInUser, in *model.ShiftTemplate) (int64, error) {
	id, err := s.store.CreateShiftTemplate(ctx, user, in)
	if err != nil {
		return 0, err
	}

	// s.cache.DeleteMany(s.cache.KeyPrefix(search.DomainId, ShiftTemplateCacheScope, 0))

	return id, nil
}

func (s *ShiftTemplate) ReadShiftTemplate(ctx context.Context, user *model.SignedInUser, search *model.SearchItem) (*model.ShiftTemplate, error) {
	// fetchFn := func(ctx context.Context) ([]*model.ShiftTemplate, error) {
	items, err := s.store.SearchShiftTemplate(ctx, user, search)
	if err != nil {
		return nil, err
	}

	// 	return items, nil
	// }

	// items, err := cache.GetOrFetch(ctx, s.cache.Raw(), s.cache.Key(search.DomainId, ShiftTemplateCacheScope, search.Token, search), fetchFn)
	// if err != nil {
	// 	return nil, err
	// }

	if len(items) > 1 {
		return nil, werror.NewDBEntityConflictError("service.pause_template.read.conflict")
	}

	if len(items) == 0 {
		return nil, werror.NewDBNoRowsErr("service.pause_template.read")
	}

	return items[0], nil
}

func (s *ShiftTemplate) SearchShiftTemplate(ctx context.Context, user *model.SignedInUser, search *model.SearchItem) ([]*model.ShiftTemplate, bool, error) {
	// fetchFn := func(ctx context.Context) ([]*model.ShiftTemplate, error) {
	items, err := s.store.SearchShiftTemplate(ctx, user, search)
	if err != nil {
		return nil, false, err
	}

	// 	return items, nil
	// }

	// items, err := cache.GetOrFetch(ctx, s.cache.Raw(), s.cache.Key(search.DomainId, ShiftTemplateCacheScope, 0, search), fetchFn)
	// if err != nil {
	// 	return nil, false, err
	// }

	var next bool
	if len(items) == int(search.Limit()) {
		next = true
	}

	return items, next, nil
}

func (s *ShiftTemplate) UpdateShiftTemplate(ctx context.Context, user *model.SignedInUser, in *model.ShiftTemplate) error {
	if err := s.store.UpdateShiftTemplate(ctx, user, in); err != nil {
		return err
	}

	// s.cache.DeleteMany(s.cache.KeyPrefix(search.DomainId, ShiftTemplateCacheScope, 0))

	return nil
}

func (s *ShiftTemplate) DeleteShiftTemplate(ctx context.Context, user *model.SignedInUser, id int64) (int64, error) {
	out, err := s.store.DeleteShiftTemplate(ctx, user, id)
	if err != nil {
		return 0, err
	}

	// s.cache.DeleteMany(s.cache.KeyPrefix(search.DomainId, ShiftTemplateCacheScope, 0))

	return out, nil
}

func (s *ShiftTemplate) SearchShiftTemplateTime(ctx context.Context, user *model.SignedInUser, shiftTemplateId int64, search *model.SearchItem) ([]*model.ShiftTemplateTime, bool, error) {
	/*var out []*model.PauseTemplateCauseService
	if ok := p.cache.Get(p.cache.Idx(cacheIdxPauseTemplateCause, in.DomainId, in.PauseTemplateId), &out); ok {
		return out, nil
	}*/

	items, err := s.store.SearchShiftTemplateTime(ctx, user, shiftTemplateId, search)
	if err != nil {
		return nil, false, err
	}

	var next bool
	if len(items) == int(search.Limit()) {
		next = true
	}

	/*p.cache.Set(p.cache.Idx(cacheIdxPauseTemplateCause, in.DomainId, in.PauseTemplateId), items)*/

	return items, next, nil
}

func (s *ShiftTemplate) UpdateShiftTemplateTimeBulk(ctx context.Context, user *model.SignedInUser, shiftTemplateId int64, in []*model.ShiftTemplateTime) error {
	if err := s.store.UpdateShiftTemplateTimeBulk(ctx, user, shiftTemplateId, in); err != nil {
		return err
	}

	/*if len(items) > 0 {
		item := items[0]
		p.cache.Set(p.cache.Idx(cacheIdxPauseTemplateCause, item.DomainId, item.PauseTemplateId), items)
	}*/

	return nil
}
