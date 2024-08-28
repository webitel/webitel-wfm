package storage

import (
	"context"

	"github.com/webitel/webitel-wfm/infra/storage/dbsql"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql/cluster"
	"github.com/webitel/webitel-wfm/internal/model"
)

const (
	shiftTemplateTable = "wfm.shift_template"
	shiftTemplateView  = shiftTemplateTable + "_v"
	shiftTemplateAcl   = shiftTemplateTable + "_acl"
)

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
	columns := map[string]interface{}{
		"domain_id":   user.DomainId,
		"created_by":  user.Id,
		"updated_by":  user.Id,
		"name":        in.Name,
		"description": in.Description,
	}

	sql, args := s.db.SQL().Insert(shiftTemplateTable, columns).SQL("RETURNING id").Build()
	if err := s.db.Primary().Get(ctx, &id, sql, args...); err != nil {
		return 0, err
	}

	return id, nil
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
		AddWhereClause(s.db.SQL().RBAC(user.UseRBAC, shiftTemplateAcl, 0, user.DomainId, user.Groups, user.Access)).
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
	ub := s.db.SQL().Update(shiftTemplateTable)
	assignments := []string{
		ub.Assign("updated_by", user.Id),
		ub.Assign("name", in.Name),
		ub.Assign("description", in.Description),
	}

	clauses := []string{
		ub.Equal("domain_id", user.DomainId),
		ub.Equal("id", in.Id),
	}

	sql, args := ub.Set(assignments...).Where(clauses...).AddWhereClause(s.db.SQL().RBAC(user.UseRBAC, shiftTemplateAcl, in.Id, user.DomainId, user.Groups, user.Access)).Build()
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

	sql, args := db.Where(clauses...).AddWhereClause(s.db.SQL().RBAC(user.UseRBAC, shiftTemplateAcl, id, user.DomainId, user.Groups, user.Access)).Build()
	if err := s.db.Primary().Exec(ctx, sql, args...); err != nil {
		return 0, err
	}

	return id, nil
}

func (s *ShiftTemplate) SearchShiftTemplateTime(ctx context.Context, user *model.SignedInUser, shiftTemplateId int64, search *model.SearchItem) ([]*model.ShiftTemplateTime, error) {
	// TODO implement me
	panic("implement me")
}

func (s *ShiftTemplate) UpdateShiftTemplateTimeBulk(ctx context.Context, user *model.SignedInUser, shiftTemplateId int64, in []*model.ShiftTemplateTime) error {
	// TODO implement me
	panic("implement me")
}
