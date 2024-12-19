package storage

import (
	"context"

	"github.com/webitel/webitel-wfm/infra/storage/dbsql"
	"github.com/webitel/webitel-wfm/internal/model"
	"github.com/webitel/webitel-wfm/pkg/werror"
)

const (
	shiftTemplateTable = "wfm.shift_template"
	shiftTemplateView  = shiftTemplateTable + "_v"
)

type ShiftTemplateManager interface {
	CreateShiftTemplate(ctx context.Context, user *model.SignedInUser, in *model.ShiftTemplate) (int64, error)
	ReadShiftTemplate(ctx context.Context, user *model.SignedInUser, id int64, fields []string) (*model.ShiftTemplate, error)
	SearchShiftTemplate(ctx context.Context, user *model.SignedInUser, search *model.SearchItem) ([]*model.ShiftTemplate, error)
	UpdateShiftTemplate(ctx context.Context, user *model.SignedInUser, in *model.ShiftTemplate) error
	DeleteShiftTemplate(ctx context.Context, user *model.SignedInUser, id int64) (int64, error)
}

type ShiftTemplate struct {
	db dbsql.Store
}

func NewShiftTemplate(db dbsql.Store) *ShiftTemplate {
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

	sql, args := s.db.SQL().Insert(shiftTemplateTable, columns).SQL("RETURNING id").Build()
	if err := s.db.Primary().Get(ctx, &id, sql, args...); err != nil {
		return 0, err
	}

	return id, nil
}

func (s *ShiftTemplate) ReadShiftTemplate(ctx context.Context, user *model.SignedInUser, id int64, fields []string) (*model.ShiftTemplate, error) {
	items, err := s.SearchShiftTemplate(ctx, user, &model.SearchItem{Id: id, Fields: fields})
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

func (s *ShiftTemplate) SearchShiftTemplate(ctx context.Context, user *model.SignedInUser, search *model.SearchItem) ([]*model.ShiftTemplate, error) {
	var (
		items   []*model.ShiftTemplate
		columns []string
	)

	columns = []string{dbsql.Wildcard(model.ShiftTemplate{})}
	if len(search.Fields) > 0 {
		columns = search.Fields
	}

	sb := s.db.SQL().Select(columns...).From(shiftTemplateView)
	sql, args := sb.Where(sb.Equal("domain_id", user.DomainId)).
		AddWhereClause(&search.Where("name").WhereClause).
		OrderBy(search.OrderBy(shiftTemplateView)).
		Limit(int(search.Limit())).
		Offset(int(search.Offset())).
		Build()

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

	ub := s.db.SQL().Update(shiftTemplateTable, columns)
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

func (s *ShiftTemplate) DeleteShiftTemplate(ctx context.Context, user *model.SignedInUser, id int64) (int64, error) {
	db := s.db.SQL().Delete(shiftTemplateTable)
	clauses := []string{
		db.Equal("domain_id", user.DomainId),
		db.Equal("id", id),
	}

	sql, args := db.Where(clauses...).Build()
	if err := s.db.Primary().Exec(ctx, sql, args...); err != nil {
		return 0, err
	}

	return id, nil
}
