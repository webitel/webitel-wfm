package storage

import (
	"context"

	"github.com/webitel/webitel-wfm/infra/storage/dbsql"
	b "github.com/webitel/webitel-wfm/infra/storage/dbsql/builder"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql/cluster"
	"github.com/webitel/webitel-wfm/internal/model"
	"github.com/webitel/webitel-wfm/internal/model/options"
	"github.com/webitel/webitel-wfm/pkg/werror"
)

type WorkingConditionManager interface {
	CreateWorkingCondition(ctx context.Context, user *model.SignedInUser, in *model.WorkingCondition) (int64, error)
	ReadWorkingCondition(ctx context.Context, read *options.Read) (*model.WorkingCondition, error)
	SearchWorkingCondition(ctx context.Context, search *options.Search) ([]*model.WorkingCondition, error)
	UpdateWorkingCondition(ctx context.Context, user *model.SignedInUser, in *model.WorkingCondition) error
	DeleteWorkingCondition(ctx context.Context, read *options.Read) (int64, error)
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

	sql, args := b.Insert(b.WorkingConditionTable.Name(), columns).SQL("RETURNING id").Build()
	if err := w.db.Primary().Get(ctx, &id, sql, args...); err != nil {
		return 0, err
	}

	return id, nil
}

func (w *WorkingCondition) ReadWorkingCondition(ctx context.Context, read *options.Read) (*model.WorkingCondition, error) {
	search, err := options.NewSearch(ctx, options.WithID(read.ID()))
	if err != nil {
		return nil, err
	}

	items, err := w.SearchWorkingCondition(ctx, search.PopulateFromRead(read))
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

func (w *WorkingCondition) SearchWorkingCondition(ctx context.Context, search *options.Search) ([]*model.WorkingCondition, error) {
	var (
		workingCondition = b.WorkingConditionTable
		createdBy        = b.UserTable.WithAlias("crt")
		updatedBy        = b.UserTable.WithAlias("upd")
		pauseTemplate    = b.PauseTemplateTable
		shiftTemplates   = b.ShiftTemplateTable
	)

	joins := b.NewJoinRegistry()
	base := b.Select().From(workingCondition.String())
	for _, field := range search.Fields() {
		switch field {
		case "id", "domain_id", "created_at", "updated_at", "name", "description", "workday_hours", "workdays_per_month",
			"vacation", "sick_leaves", "days_off", "pause_duration":
			base.SelectMore(workingCondition.Ident(field))

		case "created_by":
			joins.Register(createdBy)
			base.SelectMore(b.Alias(b.JSONBuildObject(createdBy, "id", "name"), field)).JoinWithOption(
				b.LeftJoin(createdBy,
					b.JoinExpression{
						Left:  workingCondition.Ident("created_by"),
						Op:    "=",
						Right: createdBy.Ident("id"),
					},
				),
			)

		case "updated_by":
			joins.Register(updatedBy)
			base.SelectMore(b.Alias(b.JSONBuildObject(updatedBy, "id", "name"), field)).JoinWithOption(
				b.LeftJoin(updatedBy,
					b.JoinExpression{
						Left:  workingCondition.Ident("updated_by"),
						Op:    "=",
						Right: updatedBy.Ident("id"),
					},
				),
			)

		case "pause_template":
			joins.Register(pauseTemplate)
			base.SelectMore(b.Alias(b.JSONBuildObject(pauseTemplate, "id", "name"), field)).JoinWithOption(
				b.LeftJoin(pauseTemplate,
					b.JoinExpression{
						Left:  pauseTemplate.Ident("pause_template_id"),
						Op:    "=",
						Right: workingCondition.Ident("id"),
					},
				),
			)

		case "shift_template":
			joins.Register(shiftTemplates)
			base.SelectMore(b.Alias(b.JSONBuildObject(shiftTemplates, "id", "name"), field)).JoinWithOption(
				b.LeftJoin(shiftTemplates,
					b.JoinExpression{
						Left:  workingCondition.Ident("shift_template_id"),
						Op:    "=",
						Right: shiftTemplates.Ident("id"),
					},
				),
			)
		}
	}

	base.Where(base.EQ(pauseTemplate.Ident("domain_id"), search.User().DomainId))
	if search.Query() != "" {
		base.Where(base.Like(pauseTemplate.Ident("name"), search.Query()))
	}

	if ids := search.IDs(); len(ids) > 0 {
		base.Where(base.In(pauseTemplate.Ident("id"), b.ConvertArgs(ids)...))
	}

	for field, direction := range search.OrderBy() {
		switch field {
		case "id", "name", "description", "created_at", "updated_at":
			base.OrderBy(b.OrderBy(pauseTemplate.Ident(field), direction))

		case "created_by":
			if !joins.Has(createdBy) {
				joins.Register(createdBy)
				base.JoinWithOption(b.LeftJoin(createdBy,
					b.JoinExpression{
						Left:  pauseTemplate.Ident("created_by"),
						Op:    "=",
						Right: createdBy.Ident("id"),
					},
				))
			}

			base.OrderBy(b.OrderBy(createdBy.Ident("name"), direction))

		case "updated_by":
			if !joins.Has(updatedBy) {
				joins.Register(updatedBy)
				base.JoinWithOption(b.LeftJoin(updatedBy,
					b.JoinExpression{
						Left:  pauseTemplate.Ident("updated_by"),
						Op:    "=",
						Right: updatedBy.Ident("id"),
					},
				))
			}

			base.OrderBy(b.OrderBy(updatedBy.Ident("name"), direction))

		case "pause_template":
			if !joins.Has(pauseTemplate) {
				joins.Register(pauseTemplate)
				base.JoinWithOption(b.LeftJoin(pauseTemplate,
					b.JoinExpression{
						Left:  pauseTemplate.Ident("pause_template_id"),
						Op:    "=",
						Right: workingCondition.Ident("id"),
					},
				))
			}

			base.OrderBy(b.OrderBy(pauseTemplate.Ident("name"), direction))

		case "shift_template":
			if !joins.Has(shiftTemplates) {
				joins.Register(shiftTemplates)
				base.JoinWithOption(b.LeftJoin(shiftTemplates,
					b.JoinExpression{
						Left:  workingCondition.Ident("shift_template_id"),
						Op:    "=",
						Right: shiftTemplates.Ident("id"),
					},
				))
			}

			base.OrderBy(b.OrderBy(shiftTemplates.Ident("name"), direction))
		}
	}

	var items []*model.WorkingCondition
	sql, args := base.Limit(search.Size()).Offset(search.Offset()).Build()
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

	ub := b.Update(b.WorkingConditionTable.Name(), columns)
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

func (w *WorkingCondition) DeleteWorkingCondition(ctx context.Context, read *options.Read) (int64, error) {
	db := b.Delete(b.WorkingConditionTable.Name())
	clauses := []string{
		db.Equal("domain_id", read.User().DomainId),
		db.Equal("id", read.ID()),
	}

	sql, args := db.Where(clauses...).Build()
	if err := w.db.Primary().Exec(ctx, sql, args...); err != nil {
		return 0, err
	}

	return read.ID(), nil
}
