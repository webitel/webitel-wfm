package service

import (
	"context"

	"github.com/webitel/webitel-wfm/internal/model"
	"github.com/webitel/webitel-wfm/internal/storage"
)

type ShiftTemplateManager interface {
	CreateShiftTemplate(ctx context.Context, user *model.SignedInUser, in *model.ShiftTemplate) (int64, error)
	ReadShiftTemplate(ctx context.Context, user *model.SignedInUser, id int64, fields []string) (*model.ShiftTemplate, error)
	SearchShiftTemplate(ctx context.Context, user *model.SignedInUser, search *model.SearchItem) ([]*model.ShiftTemplate, bool, error)
	UpdateShiftTemplate(ctx context.Context, user *model.SignedInUser, in *model.ShiftTemplate) error
	DeleteShiftTemplate(ctx context.Context, user *model.SignedInUser, id int64) (int64, error)
}

type ShiftTemplate struct {
	storage storage.ShiftTemplateManager
}

func NewShiftTemplate(storage storage.ShiftTemplateManager) *ShiftTemplate {
	return &ShiftTemplate{
		storage: storage,
	}
}

func (s *ShiftTemplate) CreateShiftTemplate(ctx context.Context, user *model.SignedInUser, in *model.ShiftTemplate) (int64, error) {
	id, err := s.storage.CreateShiftTemplate(ctx, user, in)
	if err != nil {
		return 0, err
	}

	return id, nil
}

func (s *ShiftTemplate) ReadShiftTemplate(ctx context.Context, user *model.SignedInUser, id int64, fields []string) (*model.ShiftTemplate, error) {
	out, err := s.storage.ReadShiftTemplate(ctx, user, id, fields)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (s *ShiftTemplate) SearchShiftTemplate(ctx context.Context, user *model.SignedInUser, search *model.SearchItem) ([]*model.ShiftTemplate, bool, error) {
	out, err := s.storage.SearchShiftTemplate(ctx, user, search)
	if err != nil {
		return nil, false, err
	}

	next, out := model.ListResult(search.Limit(), out)

	return out, next, nil
}

func (s *ShiftTemplate) UpdateShiftTemplate(ctx context.Context, user *model.SignedInUser, in *model.ShiftTemplate) error {
	if err := s.storage.UpdateShiftTemplate(ctx, user, in); err != nil {
		return err
	}

	return nil
}

func (s *ShiftTemplate) DeleteShiftTemplate(ctx context.Context, user *model.SignedInUser, id int64) (int64, error) {
	out, err := s.storage.DeleteShiftTemplate(ctx, user, id)
	if err != nil {
		return 0, err
	}

	return out, nil
}
