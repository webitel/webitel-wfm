package service

import (
	"context"

	"github.com/webitel/webitel-wfm/internal/model"
)

type ShiftTemplateManager interface {
	CreateShiftTemplate(ctx context.Context, user *model.SignedInUser, in *model.ShiftTemplate) (int64, error)
	ReadShiftTemplate(ctx context.Context, user *model.SignedInUser, id int64, fields []string) (*model.ShiftTemplate, error)
	SearchShiftTemplate(ctx context.Context, user *model.SignedInUser, search *model.SearchItem) ([]*model.ShiftTemplate, error)
	UpdateShiftTemplate(ctx context.Context, user *model.SignedInUser, in *model.ShiftTemplate) error
	DeleteShiftTemplate(ctx context.Context, user *model.SignedInUser, id int64) (int64, error)
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

	return id, nil
}

func (s *ShiftTemplate) ReadShiftTemplate(ctx context.Context, user *model.SignedInUser, id int64, fields []string) (*model.ShiftTemplate, error) {
	out, err := s.store.ReadShiftTemplate(ctx, user, id, fields)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (s *ShiftTemplate) SearchShiftTemplate(ctx context.Context, user *model.SignedInUser, search *model.SearchItem) ([]*model.ShiftTemplate, bool, error) {
	out, err := s.store.SearchShiftTemplate(ctx, user, search)
	if err != nil {
		return nil, false, err
	}

	var next bool
	if len(out) == int(search.Limit()) {
		next = true
	}

	return out, next, nil
}

func (s *ShiftTemplate) UpdateShiftTemplate(ctx context.Context, user *model.SignedInUser, in *model.ShiftTemplate) error {
	if err := s.store.UpdateShiftTemplate(ctx, user, in); err != nil {
		return err
	}

	return nil
}

func (s *ShiftTemplate) DeleteShiftTemplate(ctx context.Context, user *model.SignedInUser, id int64) (int64, error) {
	out, err := s.store.DeleteShiftTemplate(ctx, user, id)
	if err != nil {
		return 0, err
	}

	return out, nil
}
