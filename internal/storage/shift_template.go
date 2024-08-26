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

//
// func (s *ShiftTemplate) CreateShiftTemplate(ctx context.Context, search *model.SearchItem, in *model.ShiftTemplate) (int64, error) {
// 	var id int64
// 	columns := map[string]interface{}{
// 		"domain_id":   search.SignedInUser.DomainId,
// 		"created_by":  search.SignedInUser.Id,
// 		"updated_by":  search.SignedInUser.Id,
// 		"name":        in.Name,
// 		"description": in.Description,
// 	}
//
// 	query, args := s.store.SQL().Insert("wfm.shift_template").SetMap(columns).Suffix("RETURNING id").MustSql()
// 	if err := s.store.Primary().Conn().Get(ctx, &id, query, args...); err != nil {
// 		return 0, err.SetId("Store.shift_template.create")
// 	}
//
// 	return id, nil
// }
//
// func (s *ShiftTemplate) SearchShiftTemplate(ctx context.Context, search *model.SearchItem) ([]*model.ShiftTemplate, error) {
// 	var (
// 		item    []*model.ShiftTemplate
// 		columns []string
// 	)
//
// 	columns = []string{dbsql.Wildcard(model.ShiftTemplate{})}
// 	if len(search.Fields) > 0 {
// 		columns = search.Fields
// 	}
//
// 	filters := []map[string]any{
// 		dbsql.Eq{"domain_id": search.SignedInUser.DomainId},
// 		dbsql.Eq{"id": search.Id},
// 		dbsql.ILike{"name": search.Search},
// 	}
//
// 	query, args := s.store.SQL().Select(columns...).From("wfm.shift_template_v").
// 		Where(search.Where(filters...)).
// 		Limit(uint64(search.Limit())).
// 		Offset(uint64(search.Offset())).
// 		OrderBy(search.OrderBy("wfm.shift_template")).MustSql()
//
// 	if err := s.store.StandbyPreferred().Conn().Select(ctx, &item, query, args...); err != nil {
// 		return nil, err.SetId("Store.shift_template.search")
// 	}
//
// 	return item, nil
// }
//
// func (s *ShiftTemplate) UpdateShiftTemplate(ctx context.Context, search *model.SearchItem, in *model.ShiftTemplate) error {
// 	columns := map[string]interface{}{
// 		"name":        in.Name,
// 		"description": in.Description,
// 		"updated_by":  search.SignedInUser.Id,
// 		"updated_at":  time.Now(),
// 	}
//
// 	query, args := s.store.SQL().Update("wfm.shift_template").SetMap(columns).Where(sq.Eq{"domain_id": search.SignedInUser.DomainId, "id": search.Id}).MustSql()
// 	if err := s.store.Primary().Conn().Exec(ctx, query, args...); err != nil {
// 		return err.SetId("Store.shift_template.update")
// 	}
//
// 	return nil
// }
//
// func (s *ShiftTemplate) DeleteShiftTemplate(ctx context.Context, search *model.SearchItem) (int64, error) {
// 	var out int64
// 	query, args := s.store.SQL().Delete("wfm.shift_template").Where(dbsql.Eq{"domain_id": search.SignedInUser.DomainId, "id": search.Id}).Suffix("RETURNING id").MustSql()
// 	if err := s.store.Primary().Conn().Get(ctx, &out, query, args...); err != nil {
// 		return 0, err.SetId("Store.shift_template.delete")
// 	}
//
// 	return out, nil
// }
//
// func (s *ShiftTemplate) SearchShiftTemplateTime(ctx context.Context, search *model.SearchItem) ([]*model.ShiftTemplateTime, error) {
// 	var (
// 		items   []*model.ShiftTemplateTime
// 		columns []string
// 	)
//
// 	columns = []string{dbsql.Wildcard(model.ShiftTemplateTime{})}
// 	if len(search.Fields) > 0 {
// 		columns = search.Fields
// 	}
//
// 	filters := []map[string]any{
// 		dbsql.Eq{"domain_id": search.SignedInUser.DomainId},
// 		dbsql.Eq{"shift_template_id": search.Id},
// 	}
//
// 	query, args := s.store.SQL().Select(columns...).From("wfm.shift_template_time_v").
// 		Where(search.Where(filters...)).
// 		Limit(uint64(search.Limit())).
// 		Offset(uint64(search.Offset())).
// 		OrderBy(search.OrderBy("wfm.shift_template_time")).MustSql()
//
// 	if err := s.store.StandbyPreferred().Conn().Select(ctx, &items, query, args...); err != nil {
// 		return nil, err.SetId("Store.shift_template_time.list")
// 	}
//
// 	return items, nil
// }
//
// func (s *ShiftTemplate) UpdateShiftTemplateTimeBulk(ctx context.Context, search *model.SearchItem, in []*model.ShiftTemplateTime) error {
// 	// TODO implement me
// 	panic("implement me")
// }
