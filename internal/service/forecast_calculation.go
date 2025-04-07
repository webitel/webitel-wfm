package service

import (
	"context"

	"github.com/webitel/webitel-wfm/internal/model"
	"github.com/webitel/webitel-wfm/internal/model/options"
	"github.com/webitel/webitel-wfm/internal/storage"
)

type ForecastCalculationManager interface {
	CreateForecastCalculation(ctx context.Context, read *options.Read, in *model.ForecastCalculation) (*model.ForecastCalculation, error)
	ReadForecastCalculation(ctx context.Context, read *options.Read) (*model.ForecastCalculation, error)
	SearchForecastCalculation(ctx context.Context, search *options.Search) ([]*model.ForecastCalculation, bool, error)
	UpdateForecastCalculation(ctx context.Context, read *options.Read, in *model.ForecastCalculation) (*model.ForecastCalculation, error)
	DeleteForecastCalculation(ctx context.Context, read *options.Read) (int64, error)

	ExecuteForecastCalculation(ctx context.Context, read *options.Read) ([]*model.ForecastCalculationResult, error)
	CheckForecastCalculationProcedure(ctx context.Context, proc string) error
}
type ForecastCalculation struct {
	storage storage.ForecastCalculationManager
}

func NewForecastCalculation(svc storage.ForecastCalculationManager) *ForecastCalculation {
	return &ForecastCalculation{
		storage: svc,
	}
}

func (f *ForecastCalculation) CreateForecastCalculation(ctx context.Context, read *options.Read, in *model.ForecastCalculation) (*model.ForecastCalculation, error) {
	if err := f.CheckForecastCalculationProcedure(ctx, in.Procedure); err != nil {
		return nil, err
	}

	id, err := f.storage.CreateForecastCalculation(ctx, read, in)
	if err != nil {
		return nil, err
	}

	return f.ReadForecastCalculation(ctx, read.WithID(id))
}

func (f *ForecastCalculation) ReadForecastCalculation(ctx context.Context, read *options.Read) (*model.ForecastCalculation, error) {
	out, err := f.storage.ReadForecastCalculation(ctx, read)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (f *ForecastCalculation) SearchForecastCalculation(ctx context.Context, search *options.Search) ([]*model.ForecastCalculation, bool, error) {
	out, err := f.storage.SearchForecastCalculation(ctx, search)
	if err != nil {
		return nil, false, err
	}

	next, out := model.ListResult(int32(search.Size()), out)

	return out, next, nil
}

func (f *ForecastCalculation) UpdateForecastCalculation(ctx context.Context, read *options.Read, in *model.ForecastCalculation) (*model.ForecastCalculation, error) {
	if err := f.CheckForecastCalculationProcedure(ctx, in.Procedure); err != nil {
		return nil, err
	}

	if err := f.storage.UpdateForecastCalculation(ctx, read, in); err != nil {
		return nil, err
	}

	return f.ReadForecastCalculation(ctx, read)
}

func (f *ForecastCalculation) DeleteForecastCalculation(ctx context.Context, read *options.Read) (int64, error) {
	out, err := f.storage.DeleteForecastCalculation(ctx, read)
	if err != nil {
		return 0, err
	}

	return out, nil
}

func (f *ForecastCalculation) ExecuteForecastCalculation(ctx context.Context, read *options.Read) ([]*model.ForecastCalculationResult, error) {
	item, err := f.ReadForecastCalculation(ctx, read)
	if err != nil {
		return nil, err
	}

	if err := f.CheckForecastCalculationProcedure(ctx, item.Procedure); err != nil {
		return nil, err
	}

	out, err := f.storage.ExecuteForecastCalculation(ctx, read)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (f *ForecastCalculation) CheckForecastCalculationProcedure(ctx context.Context, proc string) error {
	return f.storage.CheckForecastCalculationProcedure(ctx, proc)
}
