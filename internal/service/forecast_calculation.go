package service

import (
	"context"

	"github.com/webitel/webitel-wfm/internal/model"
	"github.com/webitel/webitel-wfm/internal/storage"
)

type ForecastCalculationManager interface {
	CreateForecastCalculation(ctx context.Context, user *model.SignedInUser, in *model.ForecastCalculation) (*model.ForecastCalculation, error)
	ReadForecastCalculation(ctx context.Context, user *model.SignedInUser, search *model.SearchItem) (*model.ForecastCalculation, error)
	SearchForecastCalculation(ctx context.Context, user *model.SignedInUser, search *model.SearchItem) ([]*model.ForecastCalculation, bool, error)
	UpdateForecastCalculation(ctx context.Context, user *model.SignedInUser, in *model.ForecastCalculation) (*model.ForecastCalculation, error)
	DeleteForecastCalculation(ctx context.Context, user *model.SignedInUser, id int64) (int64, error)

	ExecuteForecastCalculation(ctx context.Context, user *model.SignedInUser, id, teamId int64, forecast *model.FilterBetween) ([]*model.ForecastCalculationResult, error)
}
type ForecastCalculation struct {
	storage storage.ForecastCalculationManager
}

func NewForecastCalculation(svc storage.ForecastCalculationManager) *ForecastCalculation {
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
	out, err := f.storage.SearchForecastCalculation(ctx, user, search)
	if err != nil {
		return nil, false, err
	}

	next, out := model.ListResult(search.Limit(), out)

	return out, next, nil
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

func (f *ForecastCalculation) ExecuteForecastCalculation(ctx context.Context, user *model.SignedInUser, id, teamId int64, forecast *model.FilterBetween) ([]*model.ForecastCalculationResult, error) {
	out, err := f.storage.ExecuteForecastCalculation(ctx, user, id, teamId, forecast)
	if err != nil {
		return nil, err
	}

	return out, nil
}
