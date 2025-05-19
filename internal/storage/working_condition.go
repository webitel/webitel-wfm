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
	const (
		linkCreatedBy = 1 << iota
		linkUpdatedBy
		linkPauseTemplate
		linkShiftTemplate
	)

	var (
		workingCondition = b.WorkingConditionTable
		createdBy        = b.UserTable.WithAlias("crt")
		updatedBy        = b.UserTable.WithAlias("upd")
		pauseTemplate    = b.PauseTemplateTable
		shiftTemplate    = b.ShiftTemplateTable
		base             = b.Select().From(workingCondition.String())

		join          = 0
		joinCreatedBy = func() {
			if join&linkCreatedBy != 0 {
				return
			}

			join |= linkCreatedBy
			base.JoinWithOption(
				b.LeftJoin(createdBy,
					b.Equal(workingCondition.Ident("created_by"), createdBy.Ident("id")),
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
					b.Equal(workingCondition.Ident("updated_by"), updatedBy.Ident("id")),
				),
			)
		}

		joinPauseTemplate = func() {
			if join&linkPauseTemplate != 0 {
				return
			}

			join |= linkPauseTemplate
			base.JoinWithOption(
				b.LeftJoin(pauseTemplate,
					b.Equal(workingCondition.Ident("pause_template_id"), pauseTemplate.Ident("id")),
				),
			)
		}

		joinShiftTemplate = func() {
			if join&linkShiftTemplate != 0 {
				return
			}

			join |= linkShiftTemplate
			base.JoinWithOption(
				b.LeftJoin(shiftTemplate,
					b.Equal(workingCondition.Ident("shift_template_id"), shiftTemplate.Ident("id")),
				),
			)
		}
	)

	{
		fields := []string{
			"id", "domain_id", "created_at", "created_by", "updated_at", "updated_by",
			"name", "description", "workday_hours", "workdays_per_month",
			"vacation", "sick_leaves", "days_off", "pause_duration",
			"pause_template", "shift_template",
		}

		for _, field := range fields {
			search.WithField(field)
		}

		for _, field := range search.Fields() {
			switch field {
			case "id", "domain_id", "created_at", "updated_at", "name", "description", "workday_hours", "workdays_per_month",
				"vacation", "sick_leaves", "days_off", "pause_duration":
				field = workingCondition.Ident(field)

			case "created_by":
				joinCreatedBy()
				field = b.Alias(b.JSONBuildObject(b.UserLookup(createdBy)), field)

			case "updated_by":
				joinUpdatedBy()
				field = b.Alias(b.JSONBuildObject(b.UserLookup(updatedBy)), field)

			case "pause_template":
				joinPauseTemplate()
				field = b.Alias(b.JSONBuildObject(b.Lookup(pauseTemplate, "id", "name")), field)

			case "shift_template":
				joinShiftTemplate()
				field = b.Alias(b.JSONBuildObject(b.Lookup(shiftTemplate, "id", "name")), field)
			}

			base.SelectMore(field)
		}
	}

	{
		base.Where(base.EQ(workingCondition.Ident("domain_id"), search.User().DomainId))
		if search.Query() != "" {
			base.Where(base.ILike(workingCondition.Ident("name"), search.Query()))
		}

		if ids := search.IDs(); len(ids) > 0 {
			base.Where(base.In(workingCondition.Ident("id"), b.ConvertArgs(ids)...))
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
				field = b.OrderBy(workingCondition.Ident(field), direction)

			case "created_by":
				joinCreatedBy()
				field = b.OrderBy(createdBy.Ident("name"), direction)

			case "updated_by":
				joinUpdatedBy()
				field = b.OrderBy(updatedBy.Ident("name"), direction)

			case "pause_template":
				joinPauseTemplate()
				field = b.OrderBy(pauseTemplate.Ident("name"), direction)

			case "shift_template":
				joinShiftTemplate()
				field = b.OrderBy(shiftTemplate.Ident("name"), direction)
			}

			base.OrderBy(field)
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
