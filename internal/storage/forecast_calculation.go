package storage

import (
	"context"
	"strconv"

	"github.com/webitel/webitel-wfm/infra/storage/cache"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql"
	b "github.com/webitel/webitel-wfm/infra/storage/dbsql/builder"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql/cluster"
	"github.com/webitel/webitel-wfm/internal/model"
	"github.com/webitel/webitel-wfm/internal/model/options"
	"github.com/webitel/webitel-wfm/pkg/werror"
)

var ErrForecastProcedureNotFound = werror.NotFound("requested forecast calculation procedure does not exists", werror.WithID("storage.forecast_calculation.procedure"))

type ForecastCalculationManager interface {
	CreateForecastCalculation(ctx context.Context, read *options.Read, in *model.ForecastCalculation) (int64, error)
	ReadForecastCalculation(ctx context.Context, read *options.Read) (*model.ForecastCalculation, error)
	SearchForecastCalculation(ctx context.Context, search *options.Search) ([]*model.ForecastCalculation, error)
	UpdateForecastCalculation(ctx context.Context, read *options.Read, in *model.ForecastCalculation) error
	DeleteForecastCalculation(ctx context.Context, read *options.Read) (int64, error)

	ExecuteForecastCalculation(ctx context.Context, read *options.Read) ([]*model.ForecastCalculationResult, error)
	CheckForecastCalculationProcedure(ctx context.Context, proc string) error
}

type ForecastCalculation struct {
	db         cluster.Store
	forecastDB cluster.ForecastStore
	cache      *cache.Scope[model.ForecastCalculation]
}

func NewForecastCalculation(db cluster.Store, manager cache.Manager, forecastDB cluster.ForecastStore) *ForecastCalculation {
	return &ForecastCalculation{
		db:         db,
		cache:      cache.NewScope[model.ForecastCalculation](manager, b.ForecastCalculationTable.Name()),
		forecastDB: forecastDB,
	}
}

func (f *ForecastCalculation) CreateForecastCalculation(ctx context.Context, read *options.Read, in *model.ForecastCalculation) (int64, error) {
	columns := []map[string]any{
		{
			"domain_id":   read.User().DomainId,
			"created_by":  read.User().Id,
			"updated_by":  read.User().Id,
			"name":        in.Name,
			"description": in.Description,
			"procedure":   in.Procedure,
			"args":        in.Args,
		},
	}

	var id int64
	sql, args := b.Insert(b.ForecastCalculationTable.Name(), columns).SQL("RETURNING id").Build()
	if err := f.db.Primary().Get(ctx, &id, sql, args...); err != nil {
		return 0, err
	}

	return id, nil
}

func (f *ForecastCalculation) ReadForecastCalculation(ctx context.Context, read *options.Read) (*model.ForecastCalculation, error) {
	search, err := options.NewSearch(ctx, options.WithID(read.ID()))
	if err != nil {
		return nil, err
	}

	items, err := f.SearchForecastCalculation(ctx, search.PopulateFromRead(read))
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

func (f *ForecastCalculation) SearchForecastCalculation(ctx context.Context, search *options.Search) ([]*model.ForecastCalculation, error) {
	const (
		linkCreatedBy = 1 << iota
		linkUpdatedBy
	)

	var (
		forecastCalculation = b.ForecastCalculationTable
		createdBy           = b.UserTable.WithAlias("crt")
		updatedBy           = b.UserTable.WithAlias("upd")
		base                = b.Select().From(forecastCalculation.String())

		join          = 0
		joinCreatedBy = func() {
			if join&linkCreatedBy != 0 {
				return
			}

			join |= linkCreatedBy
			base.JoinWithOption(
				b.LeftJoin(createdBy,
					b.JoinExpression{
						Left:  forecastCalculation.Ident("created_by"),
						Op:    "=",
						Right: createdBy.Ident("id"),
					},
				),
			)
		}

		joinUpdatedBy = func() {
			if join&linkUpdatedBy != 0 {
				return
			}

			join |= linkUpdatedBy
			base.JoinWithOption(
				b.LeftJoin(updatedBy,
					b.JoinExpression{
						Left:  forecastCalculation.Ident("updated_by"),
						Op:    "=",
						Right: updatedBy.Ident("id"),
					},
				),
			)
		}
	)

	{
		// Default fields
		for _, field := range []string{"id", "name", "created_at", "created_by", "updated_at", "updated_by"} {
			search.WithField(field)
		}

		for _, field := range search.Fields() {
			switch field {
			case "id", "domain_id", "created_at", "updated_at", "name", "description", "procedure", "args":
				field = forecastCalculation.Ident(field)

			case "created_by":
				joinCreatedBy()
				field = b.Alias(b.JSONBuildObject(b.UserLookup(createdBy)), field)

			case "updated_by":
				joinUpdatedBy()
				field = b.Alias(b.JSONBuildObject(b.UserLookup(updatedBy)), field)
			}

			base.SelectMore(field)
		}
	}

	{
		base.Where(base.EQ(forecastCalculation.Ident("domain_id"), search.User().DomainId))
		if search.Query() != "" {
			base.Where(base.ILike(forecastCalculation.Ident("name"), search.Query()))
		}

		if ids := search.IDs(); len(ids) > 0 {
			base.Where(base.In(forecastCalculation.Ident("id"), b.ConvertArgs(ids)...))
		}
	}

	{
		orderBy := search.OrderBy()
		if len(orderBy) == 0 {
			orderBy.WithOrderBy("created_at", b.OrderDirectionASC)
		}

		for field, direction := range orderBy {
			switch field {
			case "id", "name", "description", "created_at", "updated_at":
				field = b.OrderBy(forecastCalculation.Ident(field), direction)

			case "created_by":
				joinCreatedBy()
				field = b.OrderBy(createdBy.Ident("name"), direction)

			case "updated_by":
				joinUpdatedBy()
				field = b.OrderBy(updatedBy.Ident("name"), direction)
			}

			base.OrderBy(field)
		}
	}

	var items []*model.ForecastCalculation
	sql, args := base.Limit(search.Size()).Offset(search.Offset()).Build()
	if err := f.db.StandbyPreferred().Select(ctx, &items, sql, args...); err != nil {
		return nil, err
	}

	return items, nil
}

func (f *ForecastCalculation) UpdateForecastCalculation(ctx context.Context, read *options.Read, in *model.ForecastCalculation) error {
	columns := map[string]any{
		"updated_by":  read.User().Id,
		"name":        in.Name,
		"description": in.Description,
		"procedure":   in.Procedure,
		"args":        in.Args,
	}

	ub := b.Update(b.ForecastCalculationTable.Name(), columns)
	clauses := []string{
		ub.Equal("domain_id", read.User().DomainId),
		ub.Equal("id", in.Id),
	}

	sql, args := ub.Where(clauses...).Build()
	if err := f.db.Primary().Exec(ctx, sql, args...); err != nil {
		return err
	}

	return nil
}

func (f *ForecastCalculation) DeleteForecastCalculation(ctx context.Context, read *options.Read) (int64, error) {
	db := b.Delete(b.ForecastCalculationTable.Name())
	clauses := []string{
		db.Equal("domain_id", read.User().DomainId),
		db.Equal("id", read.ID()),
	}

	sql, args := db.Where(clauses...).Build()
	if err := f.db.Primary().Exec(ctx, sql, args...); err != nil {
		return 0, err
	}

	return read.ID(), nil
}

func (f *ForecastCalculation) ExecuteForecastCalculation(ctx context.Context, read *options.Read, calculation *model.ForecastCalculation) ([]*model.ForecastCalculationResult, error) {
	par, args := interpolateArguments(calculation.Args, 0, nil) // TODO: pass as derived filters in options.Read
	sql := "SELECT * FROM " + calculation.Procedure + "(" + par + ")"

	var out []*model.ForecastCalculationResult
	if err := f.forecastDB.Alive().Select(ctx, &out, sql, args...); err != nil {
		return nil, err
	}

	return out, nil
}

func (f *ForecastCalculation) CheckForecastCalculationProcedure(ctx context.Context, proc string) error {
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
