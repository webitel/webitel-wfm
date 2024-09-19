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
	columns := map[string]any{
		"updated_by":  user.Id,
		"name":        in.Name,
		"description": in.Description,
	}

	ub := p.db.SQL().Update(pauseTemplateTable, columns)
	clauses := []string{
		ub.Equal("domain_id", user.DomainId),
		ub.Equal("id", in.Id),
	}

	sql, args := ub.Where(clauses...).AddWhereClause(p.db.SQL().RBAC(user.UseRBAC, pauseTemplateAcl, in.Id, user.DomainId, user.Groups, user.Access)).Build()
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
