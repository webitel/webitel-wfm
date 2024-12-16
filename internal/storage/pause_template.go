package storage

import (
	"context"
	"errors"

	"github.com/webitel/webitel-wfm/infra/storage/cache"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql"
	"github.com/webitel/webitel-wfm/internal/model"
	"github.com/webitel/webitel-wfm/pkg/werror"
)

const (
	pauseTemplateTable = "wfm.pause_template"
	pauseTemplateView  = pauseTemplateTable + "_v"

	pauseTemplateCauseTable = pauseTemplateTable + "_cause"
)

// TODO: add cache invalidation
// TODO: dont delete all keys within prefix (add something as id range 1-10 stored in cache key)

type PauseTemplate struct {
	db    dbsql.Store
	cache *cache.Scope[model.PauseTemplate]
}

func NewPauseTemplate(db dbsql.Store, manager cache.Manager) *PauseTemplate {
	return &PauseTemplate{
		db:    db,
		cache: cache.NewScope[model.PauseTemplate](manager, pauseTemplateTable),
	}
}

func (p *PauseTemplate) CreatePauseTemplate(ctx context.Context, user *model.SignedInUser, in *model.PauseTemplate) (int64, error) {
	template := []map[string]any{
		{
			"domain_id":   user.DomainId,
			"created_by":  user.Id,
			"updated_by":  user.Id,
			"name":        in.Name,
			"description": in.Description,
		},
	}

	causes := make([]map[string]any, 0, len(in.Causes))
	for _, cause := range in.Causes {
		columns := map[string]any{
			"domain_id":         user.DomainId,
			"pause_template_id": p.db.SQL().Format("(SELECT id FROM pause_template)::bigint"), // get created pause template id from CTE
			"duration":          cause.Duration,
			"pause_cause_id":    cause.Cause.SafeId(),
		}

		causes = append(causes, columns)
	}

	cte := p.db.SQL().CTE(
		p.db.SQL().With("pause_template").As(p.db.SQL().Insert(pauseTemplateTable, template).SQL("RETURNING id")),
		p.db.SQL().With("causes").As(p.db.SQL().Insert(pauseTemplateCauseTable, causes).SQL("RETURNING id")),
	).Builder()

	var id int64

	// WITH
	// 	pause_template AS (
	// 		INSERT INTO wfm.pause_template () VALUES () RETURNING id
	// 	)
	//
	// 	, causes AS (
	// 		INSERT INTO wfm.pause_template_cause () VALUES (), (), () RETURNING id
	// 	)
	//
	// SELECT distinct pause_template.id FROM pause_template, causes;
	sql, args := p.db.SQL().Select("distinct pause_template.id").With(cte).From("pause_template", "causes").Build()
	if err := p.db.Primary().Get(ctx, &id, sql, args...); err != nil {
		return 0, err
	}

	return id, nil
}

func (p *PauseTemplate) ReadPauseTemplate(ctx context.Context, user *model.SignedInUser, id int64, fields []string) (*model.PauseTemplate, error) {
	items, err := p.SearchPauseTemplate(ctx, user, &model.SearchItem{Id: id, Fields: fields})
	if err != nil {
		return nil, err
	}

	if len(items) > 1 {
		return nil, werror.Wrap(dbsql.ErrEntityConflict, werror.WithID("storage.pause_template.read.conflict"))
	}

	if len(items) == 0 {
		return nil, werror.Wrap(dbsql.ErrNoRows, werror.WithID("storage.pause_template.read"))
	}

	return items[0], nil
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
	template := map[string]any{
		"updated_by":  user.Id,
		"name":        in.Name,
		"description": in.Description,
	}

	updateTemplate := p.db.SQL().Update(pauseTemplateTable, template)
	clauses := []string{
		updateTemplate.Equal("domain_id", user.DomainId),
		updateTemplate.Equal("id", in.Id),
	}

	updateTemplate.Where(clauses...).SQL("RETURNING id")

	templateId := p.db.SQL().Format("(SELECT id FROM pause_template)::bigint") // get created pause template id from CTE
	deleteCauses := p.db.SQL().Delete(pauseTemplateCauseTable)
	deleteCauses.Where(
		deleteCauses.Equal("domain_id", user.DomainId),
		deleteCauses.Equal("pause_template_id", templateId),
	).SQL("RETURNING id")

	causes := make([]map[string]any, 0, len(in.Causes))
	for _, cause := range in.Causes {
		columns := map[string]any{
			"domain_id":         user.DomainId,
			"pause_template_id": templateId,
			"duration":          cause.Duration,
			"pause_cause_id":    cause.Cause.SafeId(),
		}

		causes = append(causes, columns)
	}

	insertCauses := p.db.SQL().Insert(pauseTemplateCauseTable, causes).SQL("RETURNING id")
	cte := p.db.SQL().CTE(
		p.db.SQL().With("pause_template").As(updateTemplate),
		p.db.SQL().With("del_causes").As(deleteCauses),
		p.db.SQL().With("ins_causes").As(insertCauses),
	).Builder()

	var id int64

	// WITH
	// 	pause_template AS (
	// 		UPDATE wfm.pause_template SET name = ? ... WHERE domain_id = ? ... RETURNING id
	// 	)
	//
	// 	, del_causes AS (
	// 		DELETE FROM wfm.pause_template_cause WHERE domain_id = ? AND pause_template_id = (SELECT id FROM pause_template) RETURNING id
	// 	)
	//
	// 	, ins_causes AS (
	// 		INSERT INTO wfm.pause_template_cause () VALUES (), (), () RETURNING id
	// 	)
	//
	// SELECT distinct pause_template.id FROM pause_template, del_causes, ins_causes;
	sql, args := p.db.SQL().Select("distinct pause_template.id").From("pause_template", "del_causes", "ins_causes").With(cte).Build()
	if err := p.db.Primary().Get(ctx, &id, sql, args...); err != nil {
		if errors.Is(err, dbsql.ErrNoRows) {
			return werror.Wrap(dbsql.ErrNoRows, werror.WithID("storage.pause_template.update"), werror.WithCause(err))
		}

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

	sql, args := db.Where(clauses...).Build()
	if err := p.db.Primary().Exec(ctx, sql, args...); err != nil {
		return 0, err
	}

	return id, nil
}
