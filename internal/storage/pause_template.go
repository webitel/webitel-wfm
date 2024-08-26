package storage

import (
	"context"

	"github.com/webitel/webitel-wfm/infra/storage/cache"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql/cluster"
	"github.com/webitel/webitel-wfm/internal/model"
)

const (
	pauseTemplateTable = "wfm.pause_template"
	pauseTemplateView  = pauseTemplateTable + "_v"
	pauseTemplateAcl   = pauseTemplateTable + "_acl"
)

// TODO: add cache invalidation
// TODO: dont delete all keys within prefix (add something as id range 1-10 stored in cache key)

type PauseTemplate struct {
	db       cluster.Store
	ptCache  *cache.Scope[model.PauseTemplate]
	ptcCache *cache.Scope[model.PauseTemplateCause]
}

func NewPauseTemplate(db cluster.Store, manager cache.Manager) *PauseTemplate {
	return &PauseTemplate{
		db:       db,
		ptCache:  cache.NewScope[model.PauseTemplate](manager, pauseTemplateTable),
		ptcCache: cache.NewScope[model.PauseTemplateCause](manager, "pause_template_cause"),
	}
}

func (p *PauseTemplate) CreatePauseTemplate(ctx context.Context, user *model.SignedInUser, in *model.PauseTemplate) (int64, error) {
	var id int64
	columns := map[string]interface{}{
		"domain_id":   user.DomainId,
		"created_by":  user.Id,
		"updated_by":  user.Id,
		"name":        in.Name,
		"description": in.Description,
	}

	sql, args := p.db.SQL().Insert(pauseTemplateTable, columns).SQL("RETURNING id").Build()
	if err := p.db.Primary().Get(ctx, &id, sql, args...); err != nil {
		return 0, err
	}

	return id, nil
}

func (p *PauseTemplate) SearchPauseTemplate(ctx context.Context, user *model.SignedInUser, search *model.SearchItem) ([]*model.PauseTemplate, error) {
	var (
		items   []*model.PauseTemplate
		columns []string
	)

	columns = []string{dbsql.Wildcard(model.PauseTemplate{})}
	if len(search.Fields) > 0 {
		columns = search.Fields
	}

	sb := p.db.SQL().Select(columns...).From(pauseTemplateView)
	sql, args := sb.Where(sb.Equal("domain_id", user.DomainId)).
		AddWhereClause(&search.Where("name").WhereClause).
		AddWhereClause(p.db.SQL().RBAC(user.UseRBAC, pauseTemplateAcl, 0, user.DomainId, user.Groups, user.Access)).
		OrderBy(search.OrderBy(pauseTemplateView)).
		Limit(int(search.Limit())).
		Offset(int(search.Offset())).
		Build()

	if err := p.db.StandbyPreferred().Select(ctx, &items, sql, args...); err != nil {
		return nil, err
	}

	return items, nil
}

func (p *PauseTemplate) UpdatePauseTemplate(ctx context.Context, user *model.SignedInUser, in *model.PauseTemplate) error {
	ub := p.db.SQL().Update(pauseTemplateTable)
	assignments := []string{
		ub.Assign("updated_by", user.Id),
		ub.Assign("name", in.Name),
		ub.Assign("description", in.Description),
	}

	clauses := []string{
		ub.Equal("domain_id", user.DomainId),
		ub.Equal("id", in.Id),
	}

	sql, args := ub.Set(assignments...).Where(clauses...).AddWhereClause(p.db.SQL().RBAC(user.UseRBAC, pauseTemplateAcl, in.Id, user.DomainId, user.Groups, user.Access)).Build()
	if err := p.db.Primary().Exec(ctx, sql, args...); err != nil {
		return err
	}

	return nil
}

func (p *PauseTemplate) DeletePauseTemplate(ctx context.Context, user *model.SignedInUser, id int64) (int64, error) {
	db := p.db.SQL().Delete(pauseTemplateTable)
	clauses := []string{
		db.Equal("domain_id", user.DomainId),
		db.Equal("id", id),
	}

	sql, args := db.Where(clauses...).AddWhereClause(p.db.SQL().RBAC(user.UseRBAC, pauseTemplateAcl, id, user.DomainId, user.Groups, user.Access)).Build()
	if err := p.db.Primary().Exec(ctx, sql, args...); err != nil {
		return 0, err
	}

	return id, nil
}

func (p *PauseTemplate) SearchPauseTemplateCause(ctx context.Context, user *model.SignedInUser, pauseTemplateId int64, search *model.SearchItem) ([]*model.PauseTemplateCause, error) {
	// TODO implement me
	panic("implement me")
}

func (p *PauseTemplate) UpdatePauseTemplateCauseBulk(ctx context.Context, user *model.SignedInUser, pauseTemplateId int64, in []*model.PauseTemplateCause) error {
	// TODO implement me
	panic("implement me")
}

// func (p *PauseTemplate) CreatePauseTemplate(ctx context.Context, search *model.SearchItem, in *model.PauseTemplate) (int64, error) {
// 	var id int64
// 	columns := map[string]interface{}{
// 		"domain_id":   search.SignedInUser.DomainId,
// 		"created_by":  search.SignedInUser.Id,
// 		"updated_by":  search.SignedInUser.Id,
// 		"name":        in.Name,
// 		"description": in.Description,
// 	}
//
// 	query, args := p.db.SQL().Insert("wfm.shift_template").SetMap(columns).Suffix("RETURNING id").MustSql()
// 	if err := p.db.Primary().Conn().Get(ctx, &id, query, args...); err != nil {
// 		return 0, err
// 	}
//
// 	go p.cache.Key(search.DomainId, 0).Delete(ctx)
//
// 	return id, nil
// }
//
// func (p *PauseTemplate) SearchPauseTemplate(ctx context.Context, search *model.SearchItem) ([]*model.PauseTemplate, error) {
// 	var items []*model.PauseTemplate
//
// 	items, ok := p.cache.Key(search.DomainId, search.Id, search).GetMany(ctx)
// 	if ok {
// 		return items, nil
// 	}
//
// 	columns := []string{dbsql.Wildcard(model.PauseTemplate{})}
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
// 	query, args := p.db.SQL().Select(columns...).From("wfm.pause_template_v").
// 		Where(search.Where(filters...)).
// 		Limit(uint64(search.Limit())).
// 		Offset(uint64(search.Offset())).
// 		OrderBy(search.OrderBy("wfm.pause_template")).MustSql()
//
// 	if err := p.db.StandbyPreferred().Conn().Select(ctx, &items, query, args...); err != nil {
// 		return nil, err
// 	}
//
// 	go p.cache.Key(search.DomainId, search.Id, search).SetMany(ctx, items)
//
// 	return items, nil
// }
//
// func (p *PauseTemplate) UpdatePauseTemplate(ctx context.Context, search *model.SearchItem, in *model.PauseTemplate) error {
// 	columns := map[string]interface{}{
// 		"name":        in.Name,
// 		"description": in.Description,
// 		"updated_by":  search.SignedInUser.Id,
// 		"updated_at":  time.Now().UTC(),
// 	}
//
// 	query, args := p.db.SQL().Update("wfm.pause_template").SetMap(columns).Where(sq.Eq{"domain_id": search.SignedInUser.DomainId, "id": search.Id}).MustSql()
// 	if err := p.db.Primary().Conn().Exec(ctx, query, args...); err != nil {
// 		return err
// 	}
//
// 	go p.cache.Key(search.DomainId, search.Id).Delete(ctx)
//
// 	return nil
// }
//
// func (p *PauseTemplate) DeletePauseTemplate(ctx context.Context, search *model.SearchItem) (int64, error) {
// 	var id int64
// 	query, args := p.db.SQL().Delete("wfm.pause_template").Where(sq.Eq{"domain_id": search.SignedInUser.DomainId, "id": search.Id}).Suffix("RETURNING id").MustSql()
// 	if err := p.db.Primary().Conn().Get(ctx, &id, query, args...); err != nil {
// 		return 0, err
// 	}
//
// 	go p.cache.Key(search.DomainId, search.Id).Delete(ctx)
//
// 	return id, nil
// }
// func (p *PauseTemplateCause) SearchPauseTemplateCause(ctx context.Context, search *model.SearchItem) ([]*model.PauseTemplateCause, error) {
// 	var items []*model.PauseTemplateCause
//
// 	items, ok := p.cache.Key(search.DomainId, 0, search).GetMany(ctx)
// 	if ok {
// 		return items, nil
// 	}
//
// 	columns := []string{dbsql.Wildcard(model.PauseTemplateCause{})}
// 	if len(search.Fields) > 0 {
// 		columns = search.Fields
// 	}
//
// 	filters := []map[string]any{
// 		dbsql.Eq{"domain_id": search.SignedInUser.DomainId},
// 		dbsql.Eq{"pause_template_id": search.Id},
// 		dbsql.ILike{"cause ->> 'name'": search.Search},
// 	}
//
// 	query, args := p.db.SQL().Select(columns...).From("wfm.pause_template_cause_v").
// 		Where(search.Where(filters...)).
// 		Limit(uint64(search.Limit())).
// 		Offset(uint64(search.Offset())).
// 		OrderBy(search.OrderBy("wfm.pause_template_cause")).MustSql()
//
// 	if err := p.db.StandbyPreferred().Conn().Select(ctx, &items, query, args...); err != nil {
// 		return nil, err
// 	}
//
// 	go p.cache.Key(search.DomainId, 0, search).SetMany(ctx, items)
//
// 	return items, nil
// }
//
// func (p *PauseTemplateCause) UpdatePauseTemplateCauseBulk(ctx context.Context, search *model.SearchItem, in []*model.PauseTemplateCause) error {
// 	// FIXME: squirrel
// 	query := `WITH causes AS (INSERT INTO wfm.pause_template_cause (id, domain_id, pause_template_id, pause_cause_id, duration, created_by, updated_by)
// VALUES (coalesce(nullif(:id, 0), nextval('wfm.pause_template_cause_id_seq')), :domain_id, :pause_template_id, :cause, :duration, :created_by, :updated_by)
// ON CONFLICT (id, domain_id, pause_template_id)
//     DO UPDATE SET duration       = EXCLUDED.duration,
//                   pause_cause_id = EXCLUDED.pause_cause_id,
//                   updated_at     = now(),
//                   updated_by     = EXCLUDED.updated_by
// RETURNING id, domain_id, pause_template_id)
//
// 		DELETE FROM wfm.pause_template_cause
// 		WHERE id NOT IN (SELECT id FROM causes) AND (domain_id, pause_template_id) = (SELECT DISTINCT domain_id, pause_template_id FROM causes)`
//
// 	args := make([]map[string]interface{}, 0, len(in))
// 	for _, v := range in {
// 		args = append(args, map[string]interface{}{
// 			"domain_id":         search.SignedInUser.DomainId,
// 			"pause_template_id": search.Id,
// 			"id":                v.Id,
// 			"cause":             v.Cause.Id,
// 			"duration":          v.Duration,
// 			"created_by":        search.SignedInUser.Id,
// 			"updated_by":        search.SignedInUser.Id,
// 		})
// 	}
//
// 	if err := p.db.Primary().Conn().Exec(ctx, query, args); err != nil {
// 		return err
// 	}
//
// 	go p.cache.Key(search.DomainId, search.Id).Delete(ctx)
//
// 	return nil
// }
