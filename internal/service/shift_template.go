package service

import (
	"context"

	"github.com/webitel/webitel-wfm/internal/model"
	"github.com/webitel/webitel-wfm/internal/model/options"
	"github.com/webitel/webitel-wfm/internal/storage"
)

type ShiftTemplateManager interface {
	CreateShiftTemplate(ctx context.Context, read *options.Read, in *model.ShiftTemplate) (*model.ShiftTemplate, error)
	ReadShiftTemplate(ctx context.Context, read *options.Read) (*model.ShiftTemplate, error)
	SearchShiftTemplate(ctx context.Context, search *options.Search) ([]*model.ShiftTemplate, bool, error)
	UpdateShiftTemplate(ctx context.Context, read *options.Read, in *model.ShiftTemplate) (*model.ShiftTemplate, error)
	DeleteShiftTemplate(ctx context.Context, read *options.Read) (int64, error)
}

type ShiftTemplate struct {
	storage storage.ShiftTemplateManager
}

func NewShiftTemplate(storage storage.ShiftTemplateManager) *ShiftTemplate {
	return &ShiftTemplate{
		storage: storage,
	}
}

func (s *ShiftTemplate) CreateShiftTemplate(ctx context.Context, read *options.Read, in *model.ShiftTemplate) (*model.ShiftTemplate, error) {
	id, err := s.storage.CreateShiftTemplate(ctx, read.User(), in)
	if err != nil {
		return nil, err
	}

	return s.ReadShiftTemplate(ctx, read.WithID(id))
}

func (s *ShiftTemplate) ReadShiftTemplate(ctx context.Context, read *options.Read) (*model.ShiftTemplate, error) {
	out, err := s.storage.ReadShiftTemplate(ctx, read)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (s *ShiftTemplate) SearchShiftTemplate(ctx context.Context, search *options.Search) ([]*model.ShiftTemplate, bool, error) {
	out, err := s.storage.SearchShiftTemplate(ctx, search)
	if err != nil {
		return nil, false, err
	}

	next, out := model.ListResult(int32(search.Size()), out)

	return out, next, nil
}

func (s *ShiftTemplate) UpdateShiftTemplate(ctx context.Context, read *options.Read, in *model.ShiftTemplate) (*model.ShiftTemplate, error) {
	if err := s.storage.UpdateShiftTemplate(ctx, read.User(), in); err != nil {
		return nil, err
	}

	return s.ReadShiftTemplate(ctx, read)
}

func (s *ShiftTemplate) DeleteShiftTemplate(ctx context.Context, read *options.Read) (int64, error) {
	out, err := s.storage.DeleteShiftTemplate(ctx, read)
	if err != nil {
		return 0, err
	}

	return out, nil
}
