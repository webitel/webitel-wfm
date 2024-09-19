package storage

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/webitel/webitel-wfm/infra/storage/cache"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql/cluster"
	"github.com/webitel/webitel-wfm/internal/model"
	"github.com/webitel/webitel-wfm/pkg/werror"
)

const (
	forecastCalculationTable = "wfm.forecast_calculation"
	forecastCalculationView  = forecastCalculationTable + "_v"
	forecastCalculationAcl   = forecastCalculationTable + "_acl"
)

type ForecastCalculation struct {
	db         cluster.Store
	forecastDB cluster.ForecastStore
	cache      *cache.Scope[model.ForecastCalculation]
}

func NewForecastCalculation(db cluster.Store, manager cache.Manager, forecastDB cluster.ForecastStore) *ForecastCalculation {
	return &ForecastCalculation{
		db:         db,
		cache:      cache.NewScope[model.ForecastCalculation](manager, forecastCalculationTable),
		forecastDB: forecastDB,
	}
}

func (f *ForecastCalculation) CreateForecastCalculation(ctx context.Context, user *model.SignedInUser, in *model.ForecastCalculation) (*model.ForecastCalculation, error) {
	var id int64
	columns := map[string]interface{}{
		"domain_id":   user.DomainId,
		"created_by":  user.Id,
		"updated_by":  user.Id,
		"name":        in.Name,
		"description": in.Description,
		"procedure":   in.Procedure,
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
		return nil, werror.NewDBEntityConflictError("storage.forecast_calculation.read.conflict")
	}

	if len(items) == 0 {
		return nil, werror.NewDBNoRowsErr("storage.forecast_calculation.read")
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
		AddWhereClause(f.db.SQL().RBAC(user.UseRBAC, forecastCalculationAcl, 0, user.DomainId, user.Groups, user.Access)).
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
	columns := map[string]any{
		"updated_by":  user.Id,
		"name":        in.Name,
		"description": in.Description,
		"procedure":   in.Procedure,
	}

	ub := f.db.SQL().Update(forecastCalculationTable, columns)
	clauses := []string{
		ub.Equal("domain_id", user.DomainId),
		ub.Equal("id", in.Id),
	}

	sql, args := ub.Where(clauses...).AddWhereClause(f.db.SQL().RBAC(user.UseRBAC, forecastCalculationAcl, in.Id, user.DomainId, user.Groups, user.Access)).Build()
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

	sql, args := db.Where(clauses...).AddWhereClause(f.db.SQL().RBAC(user.UseRBAC, forecastCalculationAcl, id, user.DomainId, user.Groups, user.Access)).Build()
	if err := f.db.Primary().Exec(ctx, sql, args...); err != nil {
		return 0, err
	}

	return id, nil
}

func (f *ForecastCalculation) ExecuteForecastCalculation(ctx context.Context, user *model.SignedInUser, id int64, timeFilter *model.ForecastCalculationExecution) ([]*model.ForecastCalculationResult, error) {
	item, err := f.ReadForecastCalculation(ctx, user, &model.SearchItem{Id: id})
	if err != nil {
		return nil, err
	}

	var out []*model.ForecastCalculationResult
	sql, args := f.forecastDB.SQL().Format("CALL ?", interpolateProcedure(item.Procedure, timeFilter)).Build()
	if err := f.forecastDB.Alive().Select(ctx, &out, sql, args...); err != nil {
		return nil, err
	}

	return out, nil
}

// interpolateProcedure adds default time filter for SQL based on the starting and ending query time range.
//
// Example:
//
//	$__timeFrom() => 1000000000
//	$__timeTo() => 1000000000
func interpolateProcedure(proc string, timeFilter *model.ForecastCalculationExecution) string {
	if timeFilter.ForecastFrom == 0 {
		timeFilter.ForecastFrom = time.Now().Unix()
	}

	if timeFilter.ForecastTo == 0 {
		timeFilter.ForecastTo = time.Now().Unix()
	}

	proc = strings.Replace(proc, "$__timeFrom()", strconv.FormatInt(timeFilter.ForecastFrom, 10), 1)
	proc = strings.Replace(proc, "$__timeTo()", strconv.FormatInt(timeFilter.ForecastTo, 10), 1)

	return proc
}
