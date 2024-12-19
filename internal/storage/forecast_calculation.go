package storage

import (
	"context"
	"strconv"

	"github.com/webitel/webitel-wfm/infra/storage/cache"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql"
	"github.com/webitel/webitel-wfm/internal/model"
	"github.com/webitel/webitel-wfm/pkg/werror"
)

const (
	forecastCalculationTable = "wfm.forecast_calculation"
	forecastCalculationView  = forecastCalculationTable + "_v"
)

var ErrForecastProcedureNotFound = werror.NotFound("", werror.WithID("storage.forecast_calculation.procedure"))

type ForecastCalculationManager interface {
	CreateForecastCalculation(ctx context.Context, user *model.SignedInUser, in *model.ForecastCalculation) (*model.ForecastCalculation, error)
	ReadForecastCalculation(ctx context.Context, user *model.SignedInUser, search *model.SearchItem) (*model.ForecastCalculation, error)
	SearchForecastCalculation(ctx context.Context, user *model.SignedInUser, search *model.SearchItem) ([]*model.ForecastCalculation, error)
	UpdateForecastCalculation(ctx context.Context, user *model.SignedInUser, in *model.ForecastCalculation) (*model.ForecastCalculation, error)
	DeleteForecastCalculation(ctx context.Context, user *model.SignedInUser, id int64) (int64, error)

	ExecuteForecastCalculation(ctx context.Context, user *model.SignedInUser, id, teamId int64, forecast *model.FilterBetween) ([]*model.ForecastCalculationResult, error)
}

type ForecastCalculation struct {
	db         dbsql.Store
	forecastDB dbsql.ForecastStore
	cache      *cache.Scope[model.ForecastCalculation]
}

func NewForecastCalculation(db dbsql.Store, manager cache.Manager, forecastDB dbsql.ForecastStore) *ForecastCalculation {
	return &ForecastCalculation{
		db:         db,
		cache:      cache.NewScope[model.ForecastCalculation](manager, forecastCalculationTable),
		forecastDB: forecastDB,
	}
}

func (f *ForecastCalculation) CreateForecastCalculation(ctx context.Context, user *model.SignedInUser, in *model.ForecastCalculation) (*model.ForecastCalculation, error) {
	if err := f.checkProcedure(ctx, in.Procedure); err != nil {
		return nil, err
	}

	var id int64
	columns := []map[string]any{
		{
			"domain_id":   user.DomainId,
			"created_by":  user.Id,
			"updated_by":  user.Id,
			"name":        in.Name,
			"description": in.Description,
			"procedure":   in.Procedure,
			"args":        in.Args,
		},
	}

	sql, args := f.db.SQL().Insert(forecastCalculationTable, columns).SQL("RETURNING id").Build()
	if err := f.db.Primary().Get(ctx, &id, sql, args...); err != nil {
		return nil, err
	}

	out, err := f.ReadForecastCalculation(ctx, user, &model.SearchItem{Id: id})
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (f *ForecastCalculation) ReadForecastCalculation(ctx context.Context, user *model.SignedInUser, search *model.SearchItem) (*model.ForecastCalculation, error) {
	items, err := f.SearchForecastCalculation(ctx, user, search)
	if err != nil {
		return nil, err
	}

	if len(items) > 1 {
		return nil, werror.Wrap(dbsql.ErrEntityConflict, werror.WithID("storage.forecast_calculation.read.conflict"))
	}

	if len(items) == 0 {
		return nil, werror.Wrap(dbsql.ErrNoRows, werror.WithID("storage.forecast_calculation.read"))
	}

	return items[0], nil
}

func (f *ForecastCalculation) SearchForecastCalculation(ctx context.Context, user *model.SignedInUser, search *model.SearchItem) ([]*model.ForecastCalculation, error) {
	var (
		items   []*model.ForecastCalculation
		columns []string
	)

	columns = []string{dbsql.Wildcard(model.ForecastCalculation{})}
	if len(search.Fields) > 0 {
		columns = search.Fields
	}

	sb := f.db.SQL().Select(columns...).From(forecastCalculationView)
	sql, args := sb.Where(sb.Equal("domain_id", user.DomainId)).
		AddWhereClause(&search.Where("name").WhereClause).
		OrderBy(search.OrderBy(forecastCalculationView)).
		Limit(int(search.Limit())).
		Offset(int(search.Offset())).
		Build()

	if err := f.db.StandbyPreferred().Select(ctx, &items, sql, args...); err != nil {
		return nil, err
	}

	return items, nil
}

func (f *ForecastCalculation) UpdateForecastCalculation(ctx context.Context, user *model.SignedInUser, in *model.ForecastCalculation) (*model.ForecastCalculation, error) {
	if err := f.checkProcedure(ctx, in.Procedure); err != nil {
		return nil, err
	}

	columns := map[string]any{
		"updated_by":  user.Id,
		"name":        in.Name,
		"description": in.Description,
		"procedure":   in.Procedure,
		"args":        in.Args,
	}

	ub := f.db.SQL().Update(forecastCalculationTable, columns)
	clauses := []string{
		ub.Equal("domain_id", user.DomainId),
		ub.Equal("id", in.Id),
	}

	sql, args := ub.Where(clauses...).Build()
	if err := f.db.Primary().Exec(ctx, sql, args...); err != nil {
		return nil, err
	}

	out, err := f.ReadForecastCalculation(ctx, user, &model.SearchItem{Id: in.Id})
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (f *ForecastCalculation) DeleteForecastCalculation(ctx context.Context, user *model.SignedInUser, id int64) (int64, error) {
	db := f.db.SQL().Delete(forecastCalculationTable)
	clauses := []string{
		db.Equal("domain_id", user.DomainId),
		db.Equal("id", id),
	}

	sql, args := db.Where(clauses...).Build()
	if err := f.db.Primary().Exec(ctx, sql, args...); err != nil {
		return 0, err
	}

	return id, nil
}

func (f *ForecastCalculation) ExecuteForecastCalculation(ctx context.Context, user *model.SignedInUser, id, teamId int64, forecast *model.FilterBetween) ([]*model.ForecastCalculationResult, error) {
	item, err := f.ReadForecastCalculation(ctx, user, &model.SearchItem{Id: id})
	if err != nil {
		return nil, err
	}

	if err := f.checkProcedure(ctx, item.Procedure); err != nil {
		return nil, err
	}

	par, args := interpolateArguments(item.Args, teamId, forecast)
	sql := "SELECT * FROM " + item.Procedure + "(" + par + ")"

	var out []*model.ForecastCalculationResult
	if err := f.forecastDB.Alive().Select(ctx, &out, sql, args...); err != nil {
		return nil, err
	}

	return out, nil
}

func (f *ForecastCalculation) checkProcedure(ctx context.Context, proc string) error {
	var exists *string
	if err := f.db.Primary().Get(ctx, &exists, "SELECT to_regproc($1)", proc); err != nil {
		return err
	}

	// to_regproc will return NULL rather than throwing an error if the name is not found or is ambiguous,
	// so we need to check this and return error if received NULL
	if exists == nil {
		return werror.Wrap(ErrForecastProcedureNotFound, werror.WithCause(dbsql.ErrNoRows),
			werror.WithValue("procedure", proc),
		)
	}

	return nil
}

// interpolateArguments generates SQL parameters list ($1, $2, ...)
// and replaces argument placeholder with value.
//
//	$__teamId() => 1
//	$__timeFrom() => 1000000000
//	$__timeTo() => 1000000001
func interpolateArguments(args []string, teamId int64, forecast *model.FilterBetween) (string, []any) {
	if len(args) == 0 {
		return "", nil
	}

	parameters := "$1"
	out := make([]any, 0, len(args))
	for i, a := range args {
		if i > 0 {
			parameters = parameters + ", $" + strconv.Itoa(i+1)
		}

		switch a {
		case "$__teamId()":
			out = append(out, teamId)
		case "$__timeFrom()":
			out = append(out, forecast.From)
		case "$__timeTo()":
			out = append(out, forecast.To)
		default:
			out = append(out, a)
		}
	}

	return parameters, out
}
