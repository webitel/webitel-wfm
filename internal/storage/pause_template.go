package storage

import (
	"context"
	"errors"
	"fmt"

	"github.com/webitel/webitel-wfm/infra/storage/cache"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql"
	b "github.com/webitel/webitel-wfm/infra/storage/dbsql/builder"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql/cluster"
	"github.com/webitel/webitel-wfm/internal/model"
	"github.com/webitel/webitel-wfm/internal/model/options"
	"github.com/webitel/webitel-wfm/pkg/werror"
)

const (
	// TODO: remove
	pauseTemplateTable = "wfm.pause_template"
	pauseTemplateView  = pauseTemplateTable + "_v"
)

// TODO: add cache invalidation
// TODO: dont delete all keys within prefix (add something as id range 1-10 stored in cache key)

type PauseTemplateManager interface {
	CreatePauseTemplate(ctx context.Context, user *model.SignedInUser, in *model.PauseTemplate) (int64, error)
	ReadPauseTemplate(ctx context.Context, read *options.Read) (*model.PauseTemplate, error)
	SearchPauseTemplate(ctx context.Context, search *options.Search) ([]*model.PauseTemplate, error)
	UpdatePauseTemplate(ctx context.Context, user *model.SignedInUser, in *model.PauseTemplate) error
	DeletePauseTemplate(ctx context.Context, read *options.Read) (int64, error)
}
type PauseTemplate struct {
	db cluster.Store

	// TODO: split db and cache in separate layers
	cache *cache.Scope[model.PauseTemplate]
}

func NewPauseTemplate(db cluster.Store, manager cache.Manager) *PauseTemplate {
	return &PauseTemplate{
		db:    db,
		cache: cache.NewScope[model.PauseTemplate](manager, b.PauseTemplateTable.Name()),
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
			"pause_template_id": b.Format("(SELECT id FROM pause_template)::bigint"), // get created pause template id from CTE
			"duration":          cause.Duration,
			"pause_cause_id":    cause.Cause.SafeId(),
		}

		causes = append(causes, columns)
	}

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
	cte := b.CTE(
		b.With("pause_template").As(b.Insert(b.PauseTemplateTable.Name(), template).SQL("RETURNING id")),
		b.With("causes").As(b.Insert(b.PauseTemplateCauseTable.Name(), causes).SQL("RETURNING id")),
	).Builder()

	var id int64
	sql, args := b.Select("distinct pause_template.id").With(cte).From("pause_template", "causes").Build()
	if err := p.db.Primary().Get(ctx, &id, sql, args...); err != nil {
		return 0, err
	}

	return id, nil
}

func (p *PauseTemplate) ReadPauseTemplate(ctx context.Context, read *options.Read) (*model.PauseTemplate, error) {
	search, err := options.NewSearch(ctx, options.WithID(read.ID()))
	if err != nil {
		return nil, err
	}

	items, err := p.SearchPauseTemplate(ctx, search.PopulateFromRead(read))
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

func (p *PauseTemplate) SearchPauseTemplate(ctx context.Context, search *options.Search) ([]*model.PauseTemplate, error) {
	const (
		linkCreatedBy = 1 << iota
		linkUpdatedBy
	)

	var (
		pauseTemplate = b.PauseTemplateTable
		createdBy     = b.UserTable.WithAlias("crt")
		updatedBy     = b.UserTable.WithAlias("upd")
		base          = b.Select().From(pauseTemplate.String())

		join          = 0
		joinCreatedBy = func() {
			if join&linkCreatedBy != 0 {
				return
			}

			join |= linkCreatedBy
			base.JoinWithOption(
				b.LeftJoin(createdBy,
					b.JoinExpression{
						Left:  pauseTemplate.Ident("created_by"),
						Op:    "=",
						Right: createdBy.Ident("id"),
					},
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
					b.JoinExpression{
						Left:  pauseTemplate.Ident("updated_by"),
						Op:    "=",
						Right: updatedBy.Ident("id"),
					},
				),
			)
		}
	)

	for _, field := range search.Fields() {
		switch field {
		case "id", "domain_id", "created_at", "updated_at", "name", "description":
			field = pauseTemplate.Ident(field)

		case "created_by":
			joinCreatedBy()
			field = b.Alias(b.JSONBuildObject(createdBy, "id", "name"), field)

		case "updated_by":
			joinUpdatedBy()
			field = b.Alias(b.JSONBuildObject(updatedBy, "id", "name"), field)

		case "causes":
			{
				causesDerived := search.DerivedByName(field)
				causesDerivedFields := causesDerived.Fields()
				if len(causesDerivedFields) == 0 {
					for _, v := range []string{"id", "duration", "cause"} {
						causesDerivedFields.WithField(v)
					}

					causesDerived.WithDerived("cause", causesDerived.DerivedByName("cause"))
				}

				var (
					pauseTemplateCause = b.PauseTemplateCauseTable
					causes             = b.Select().From(pauseTemplateCause.String())
				)

				causes.Where(fmt.Sprintf("%s = %s", pauseTemplate.Ident("id"), pauseTemplateCause.Ident("pause_template_id")))
				for _, causesDerivedField := range causesDerivedFields {
					switch causesDerivedField {
					case "id", "duration":
						causes.SelectMore(pauseTemplateCause.Ident(causesDerivedField))
						if _, d, ok := causesDerived.OrderByField(causesDerivedField); ok {
							causes.OrderBy(b.OrderBy(pauseTemplateCause.Ident(causesDerivedField), d))
						}

					case "cause":
						var pauseCause = b.PauseCauseTable
						causeDerived := causesDerived.DerivedByName(causesDerivedField)
						causeDerivedFields := causeDerived.Fields()
						if len(causeDerivedFields) == 0 {
							for _, v := range []string{"id", "name"} {
								causeDerivedFields.WithField(v)
							}
						}

						causes.SelectMore(b.Alias(b.JSONBuildObject(pauseCause, causeDerivedFields...), causesDerivedField)).JoinWithOption(
							b.LeftJoin(pauseCause,
								b.JoinExpression{
									Left:  pauseCause.Ident("id"),
									Op:    "=",
									Right: pauseTemplateCause.Ident("pause_cause_id"),
								},
							),
						)

						if _, d, ok := causesDerived.OrderByField(causesDerivedField); ok {
							causes.OrderBy(b.OrderBy(pauseCause.Ident("name"), d))
						}
					}
				}

				causesJSON := b.Select("json_agg(row_to_json(causes))")
				causesJSON.From(causesJSON.BuilderAs(causes, "causes"))
				field = base.BuilderAs(causesJSON, "causes")
			}
		}

		base.SelectMore(field)
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
			field = b.OrderBy(pauseTemplate.Ident(field), direction)

		case "created_by":
			joinCreatedBy()
			field = b.OrderBy(createdBy.Ident("name"), direction)

		case "updated_by":
			joinUpdatedBy()
			field = b.OrderBy(updatedBy.Ident("name"), direction)
		}
	}

	var items []*model.PauseTemplate
	sql, args := base.Limit(search.Size()).Offset(search.Offset()).Build()
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

	updateTemplate := b.Update(b.PauseTemplateTable.Name(), template)
	clauses := []string{
		updateTemplate.Equal("domain_id", user.DomainId),
		updateTemplate.Equal("id", in.Id),
	}

	updateTemplate.Where(clauses...).SQL("RETURNING id")

	templateId := b.Format("(SELECT id FROM pause_template)::bigint") // get created pause template id from CTE
	deleteCauses := b.Delete(b.PauseTemplateCauseTable.Name())
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

	insertCauses := b.Insert(b.PauseTemplateCauseTable.Name(), causes).SQL("RETURNING id")

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
	cte := b.CTE(
		b.With("pause_template").As(updateTemplate),
		b.With("del_causes").As(deleteCauses),
		b.With("ins_causes").As(insertCauses),
	).Builder()

	var id int64
	sql, args := b.Select("distinct pause_template.id").From("pause_template", "del_causes", "ins_causes").With(cte).Build()
	if err := p.db.Primary().Get(ctx, &id, sql, args...); err != nil {
		if errors.Is(err, dbsql.ErrNoRows) {
			return werror.Wrap(dbsql.ErrNoRows, werror.WithID("storage.pause_template.update"), werror.WithCause(err))
		}

		return err
	}

	return nil
}

func (p *PauseTemplate) DeletePauseTemplate(ctx context.Context, read *options.Read) (int64, error) {
	db := b.Delete(b.PauseTemplateTable.Name())
	clauses := []string{
		db.Equal("domain_id", read.User().DomainId),
		db.Equal("id", read.ID()),
	}

	sql, args := db.Where(clauses...).Build()
	if err := p.db.Primary().Exec(ctx, sql, args...); err != nil {
		return 0, err
	}

	return read.ID(), nil
}
