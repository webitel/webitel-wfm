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

type ShiftTemplateManager interface {
	CreateShiftTemplate(ctx context.Context, user *model.SignedInUser, in *model.ShiftTemplate) (int64, error)
	ReadShiftTemplate(ctx context.Context, read *options.Read) (*model.ShiftTemplate, error)
	SearchShiftTemplate(ctx context.Context, search *options.Search) ([]*model.ShiftTemplate, error)
	UpdateShiftTemplate(ctx context.Context, user *model.SignedInUser, in *model.ShiftTemplate) error
	DeleteShiftTemplate(ctx context.Context, read *options.Read) (int64, error)
}

type ShiftTemplate struct {
	db cluster.Store
}

func NewShiftTemplate(db cluster.Store) *ShiftTemplate {
	return &ShiftTemplate{
		db: db,
	}
}

func (s *ShiftTemplate) CreateShiftTemplate(ctx context.Context, user *model.SignedInUser, in *model.ShiftTemplate) (int64, error) {
	var id int64
	columns := []map[string]any{
		{
			"domain_id":   user.DomainId,
			"created_by":  user.Id,
			"updated_by":  user.Id,
			"name":        in.Name,
			"description": in.Description,
			"times":       in.Times,
		},
	}

	sql, args := b.Insert(b.ShiftTemplateTable.Name(), columns).SQL("RETURNING id").Build()
	if err := s.db.Primary().Get(ctx, &id, sql, args...); err != nil {
		return 0, err
	}

	return id, nil
}

func (s *ShiftTemplate) ReadShiftTemplate(ctx context.Context, read *options.Read) (*model.ShiftTemplate, error) {
	search, err := options.NewSearch(ctx, options.WithID(read.ID()))
	if err != nil {
		return nil, err
	}

	items, err := s.SearchShiftTemplate(ctx, search.PopulateFromRead(read))
	if err != nil {
		return nil, err
	}

	if len(items) > 1 {
		return nil, werror.Wrap(dbsql.ErrEntityConflict, werror.WithID("storage.shift_template.read.conflict"))
	}

	if len(items) == 0 {
		return nil, werror.Wrap(dbsql.ErrNoRows, werror.WithID("storage.shift_template.read"))
	}

	return items[0], nil
}

func (s *ShiftTemplate) SearchShiftTemplate(ctx context.Context, search *options.Search) ([]*model.ShiftTemplate, error) {
	const (
		linkCreatedBy = 1 << iota
		linkUpdatedBy
	)

	var (
		shiftTemplate = b.ShiftTemplateTable
		createdBy     = b.UserTable.WithAlias("crt")
		updatedBy     = b.UserTable.WithAlias("upd")
		base          = b.Select().From(shiftTemplate.String())

		join          = 0
		joinCreatedBy = func() {
			if join&linkCreatedBy != 0 {
				return
			}

			join |= linkCreatedBy
			base.JoinWithOption(
				b.LeftJoin(createdBy,
					b.Equal(shiftTemplate.Ident("created_by"), createdBy.Ident("id")),
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
					b.Equal(shiftTemplate.Ident("updated_by"), updatedBy.Ident("id")),
				),
			)
		}
	)

	// Fields to retrieve.
	{
		fields := []string{
			"id", "created_at", "created_by", "updated_at", "updated_by",
			"name", "description", "times",
		}

		for _, field := range fields {
			search.WithField(field)
		}

		for _, field := range search.Fields() {
			switch field {
			case "id", "domain_id", "created_at", "updated_at", "name", "description", "times":
				field = shiftTemplate.Ident(field)

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

	// Add WHERE clauses.
	{
		base.Where(base.EQ(shiftTemplate.Ident("domain_id"), search.User().DomainId))
		if search.Query() != "" {
			base.Where(base.ILike(shiftTemplate.Ident("name"), search.Query()))
		}

		if ids := search.IDs(); len(ids) > 0 {
			base.Where(base.In(shiftTemplate.Ident("id"), b.ConvertArgs(ids)...))
		}
	}

	// Construct ORDER BY fields.
	{
		orderBy := search.OrderBy()
		if len(orderBy) == 0 {
			orderBy.WithOrderBy("created_at", b.OrderDirectionASC)
		}

		for field, direction := range orderBy {
			switch field {
			case "id", "name", "description", "created_at", "updated_at":
				field = b.OrderBy(shiftTemplate.Ident(field), direction)

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

	var items []*model.ShiftTemplate
	sql, args := base.Limit(search.Size()).Offset(search.Offset()).Build()
	if err := s.db.StandbyPreferred().Select(ctx, &items, sql, args...); err != nil {
		return nil, err
	}

	return items, nil
}

func (s *ShiftTemplate) UpdateShiftTemplate(ctx context.Context, user *model.SignedInUser, in *model.ShiftTemplate) error {
	columns := map[string]any{
		"updated_by":  user.Id,
		"name":        in.Name,
		"description": in.Description,
		"times":       in.Times,
	}

	ub := b.Update(b.ShiftTemplateTable.Name(), columns)
	clauses := []string{
		ub.Equal("domain_id", user.DomainId),
		ub.Equal("id", in.Id),
	}

	sql, args := ub.Where(clauses...).Build()
	if err := s.db.Primary().Exec(ctx, sql, args...); err != nil {
		return err
	}

	return nil
}

func (s *ShiftTemplate) DeleteShiftTemplate(ctx context.Context, read *options.Read) (int64, error) {
	db := b.Delete(b.ShiftTemplateTable.Name())
	clauses := []string{
		db.Equal("domain_id", read.User().DomainId),
		db.Equal("id", read.ID()),
	}

	sql, args := db.Where(clauses...).Build()
	if err := s.db.Primary().Exec(ctx, sql, args...); err != nil {
		return 0, err
	}

	return read.ID(), nil
}
