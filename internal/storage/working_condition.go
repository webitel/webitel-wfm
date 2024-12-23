package storage

import (
	"context"

	"github.com/webitel/webitel-wfm/infra/storage/dbsql"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql/builder"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql/cluster"
	"github.com/webitel/webitel-wfm/internal/model"
	"github.com/webitel/webitel-wfm/pkg/werror"
)

const (
	workingConditionTable = "wfm.working_condition"
	workingConditionView  = workingConditionTable + "_v"
)

type WorkingConditionManager interface {
	CreateWorkingCondition(ctx context.Context, user *model.SignedInUser, in *model.WorkingCondition) (int64, error)
	ReadWorkingCondition(ctx context.Context, user *model.SignedInUser, search *model.SearchItem) (*model.WorkingCondition, error)
	SearchWorkingCondition(ctx context.Context, user *model.SignedInUser, search *model.SearchItem) ([]*model.WorkingCondition, error)
	UpdateWorkingCondition(ctx context.Context, user *model.SignedInUser, in *model.WorkingCondition) error
	DeleteWorkingCondition(ctx context.Context, user *model.SignedInUser, id int64) (int64, error)
}

type WorkingCondition struct {
	db cluster.Store
}

func NewWorkingCondition(db cluster.Store) *WorkingCondition {
	return &WorkingCondition{
		db: db,
	}
}

func (w *WorkingCondition) CreateWorkingCondition(ctx context.Context, user *model.SignedInUser, in *model.WorkingCondition) (int64, error) {
	var id int64
	columns := []map[string]any{
		{
			"domain_id":          user.DomainId,
			"created_by":         user.Id,
			"updated_by":         user.Id,
			"name":               in.Name,
			"description":        in.Description,
			"workday_hours":      in.WorkdayHours,
			"workdays_per_month": in.WorkdaysPerMonth,
			"vacation":           in.Vacation,
			"sick_leaves":        in.SickLeaves,
			"days_off":           in.DaysOff,
			"pause_duration":     in.PauseDuration,
			"pause_template_id":  in.PauseTemplate.Id,
			"shift_template_id":  in.ShiftTemplate.SafeId(),
		},
	}

	sql, args := builder.Insert(workingConditionTable, columns).SQL("RETURNING id").Build()
	if err := w.db.Primary().Get(ctx, &id, sql, args...); err != nil {
		return 0, err
	}

	return id, nil
}

func (w *WorkingCondition) ReadWorkingCondition(ctx context.Context, user *model.SignedInUser, search *model.SearchItem) (*model.WorkingCondition, error) {
	items, err := w.SearchWorkingCondition(ctx, user, search)
	if err != nil {
		return nil, err
	}

	if len(items) > 1 {
		return nil, werror.Wrap(dbsql.ErrEntityConflict, werror.WithID("storage.working_condition.read.conflict"))
	}

	if len(items) == 0 {
		return nil, werror.Wrap(dbsql.ErrNoRows, werror.WithID("storage.working_condition.read"))
	}

	return items[0], nil
}

func (w *WorkingCondition) SearchWorkingCondition(ctx context.Context, user *model.SignedInUser, search *model.SearchItem) ([]*model.WorkingCondition, error) {
	var (
		items   []*model.WorkingCondition
		columns []string
	)

	columns = []string{dbsql.Wildcard(model.WorkingCondition{})}
	if len(search.Fields) > 0 {
		columns = search.Fields
	}

	sb := builder.Select(columns...).From(workingConditionView)
	sql, args := sb.Where(sb.Equal("domain_id", user.DomainId)).
		AddWhereClause(&search.Where("name").WhereClause).
		OrderBy(search.OrderBy(workingConditionView)).
		Limit(int(search.Limit())).
		Offset(int(search.Offset())).
		Build()

	if err := w.db.StandbyPreferred().Select(ctx, &items, sql, args...); err != nil {
		return nil, err
	}

	return items, nil
}

func (w *WorkingCondition) UpdateWorkingCondition(ctx context.Context, user *model.SignedInUser, in *model.WorkingCondition) error {
	columns := map[string]any{
		"updated_by":         user.Id,
		"name":               in.Name,
		"description":        in.Description,
		"workday_hours":      in.WorkdayHours,
		"workdays_per_month": in.WorkdaysPerMonth,
		"vacation":           in.Vacation,
		"sick_leaves":        in.SickLeaves,
		"days_off":           in.DaysOff,
		"pause_duration":     in.PauseDuration,
		"pause_template_id":  in.PauseTemplate.Id,
		"shift_template_id":  in.ShiftTemplate.SafeId(),
	}

	ub := builder.Update(workingConditionTable, columns)
	clauses := []string{
		ub.Equal("domain_id", user.DomainId),
		ub.Equal("id", in.Id),
	}

	sql, args := ub.Where(clauses...).Build()
	if err := w.db.Primary().Exec(ctx, sql, args...); err != nil {
		return err
	}

	return nil
}

func (w *WorkingCondition) DeleteWorkingCondition(ctx context.Context, user *model.SignedInUser, id int64) (int64, error) {
	db := builder.Delete(workingConditionTable)
	clauses := []string{
		db.Equal("domain_id", user.DomainId),
		db.Equal("id", id),
	}

	sql, args := db.Where(clauses...).Build()
	if err := w.db.Primary().Exec(ctx, sql, args...); err != nil {
		return 0, err
	}

	return id, nil
}
