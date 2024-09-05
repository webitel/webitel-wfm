package service

import (
	"context"

	"github.com/webitel/webitel-wfm/internal/model"
)

type ForecastCalculationManager interface {
	CreateForecastCalculation(ctx context.Context, user *model.SignedInUser, in *model.ForecastCalculation) (*model.ForecastCalculation, error)
	ReadForecastCalculation(ctx context.Context, user *model.SignedInUser, search *model.SearchItem) (*model.ForecastCalculation, error)
	SearchForecastCalculation(ctx context.Context, user *model.SignedInUser, search *model.SearchItem) ([]*model.ForecastCalculation, error)
	UpdateForecastCalculation(ctx context.Context, user *model.SignedInUser, in *model.ForecastCalculation) (*model.ForecastCalculation, error)
	DeleteForecastCalculation(ctx context.Context, user *model.SignedInUser, id int64) (int64, error)
	ExecuteForecastCalculation(ctx context.Context, user *model.SignedInUser, id int64) ([]*model.ForecastCalculationResult, error)
}

type ForecastCalculation struct {
	storage ForecastCalculationManager
}

func NewForecastCalculation(svc ForecastCalculationManager) *ForecastCalculation {
	return &ForecastCalculation{
		storage: svc,
	}
}

func (f *ForecastCalculation) CreateForecastCalculation(ctx context.Context, user *model.SignedInUser, in *model.ForecastCalculation) (*model.ForecastCalculation, error) {
	out, err := f.storage.CreateForecastCalculation(ctx, user, in)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (f *ForecastCalculation) ReadForecastCalculation(ctx context.Context, user *model.SignedInUser, search *model.SearchItem) (*model.ForecastCalculation, error) {
	out, err := f.storage.ReadForecastCalculation(ctx, user, search)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (f *ForecastCalculation) SearchForecastCalculation(ctx context.Context, user *model.SignedInUser, search *model.SearchItem) ([]*model.ForecastCalculation, bool, error) {
	items, err := f.storage.SearchForecastCalculation(ctx, user, search)
	if err != nil {
		return nil, false, err
	}

	var next bool
	if len(items) == int(search.Limit()) {
		next = true
		items = items[:search.Limit()-1]
	}

	return items, next, nil
}

func (f *ForecastCalculation) UpdateForecastCalculation(ctx context.Context, user *model.SignedInUser, in *model.ForecastCalculation) (*model.ForecastCalculation, error) {
	out, err := f.storage.UpdateForecastCalculation(ctx, user, in)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (f *ForecastCalculation) DeleteForecastCalculation(ctx context.Context, user *model.SignedInUser, id int64) (int64, error) {
	out, err := f.storage.DeleteForecastCalculation(ctx, user, id)
	if err != nil {
		return 0, err
	}

	return out, nil
}

func (f *ForecastCalculation) ExecuteForecastCalculation(ctx context.Context, user *model.SignedInUser, id int64) ([]*model.ForecastCalculationResult, error) {
	out, err := f.storage.ExecuteForecastCalculation(ctx, user, id)
	if err != nil {
		return nil, err
	}

	return out, nil
}
