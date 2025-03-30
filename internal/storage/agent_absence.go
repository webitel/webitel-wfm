package storage

import (
	"context"
	"fmt"

	"github.com/webitel/webitel-wfm/infra/storage/cache"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql"
	b "github.com/webitel/webitel-wfm/infra/storage/dbsql/builder"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql/cluster"
	"github.com/webitel/webitel-wfm/internal/model"
	"github.com/webitel/webitel-wfm/internal/model/options"
	"github.com/webitel/webitel-wfm/pkg/werror"
)

type AgentAbsenceManager interface {
	CreateAgentAbsence(ctx context.Context, read *options.Read, in *model.Absence) (int64, error)
	ReadAgentAbsence(ctx context.Context, read *options.Read) (*model.Absence, error)
	SearchAgentAbsence(ctx context.Context, search *options.Search) ([]*model.Absence, error)
	UpdateAgentAbsence(ctx context.Context, read *options.Read, in *model.Absence) error
	DeleteAgentAbsence(ctx context.Context, read *options.Read) error

	CreateAgentsAbsences(ctx context.Context, search *options.Search, in []*model.AgentAbsences) ([]int64, error)
	SearchAgentsAbsences(ctx context.Context, search *options.Search) ([]*model.AgentAbsences, error)
}

type AgentAbsence struct {
	db    cluster.Store
	cache *cache.Scope[model.Absence]
}

func NewAgentAbsence(db cluster.Store, manager cache.Manager) *AgentAbsence {
	return &AgentAbsence{
		db:    db,
		cache: cache.NewScope[model.Absence](manager, b.AgentAbsenceTable.Name()),
	}
}

func (a *AgentAbsence) CreateAgentAbsence(ctx context.Context, read *options.Read, in *model.Absence) (int64, error) {
	var id int64

	sql, args := a.createAgentAbsenceQuery(a.createAgentAbsencePrepareColumns(read.User(), read.DerivedByName("agent").ID(), in))
	if err := a.db.Primary().Get(ctx, &id, sql, args...); err != nil {
		return 0, err
	}

	return id, nil
}

func (a *AgentAbsence) ReadAgentAbsence(ctx context.Context, read *options.Read) (*model.Absence, error) {
	search, err := options.NewSearch(ctx, options.WithID(read.ID()))
	if err != nil {
		return nil, err
	}

	items, err := a.SearchAgentAbsence(ctx, search.PopulateFromRead(read))
	if err != nil {
		return nil, err
	}

	if len(items) > 1 {
		return nil, werror.Wrap(dbsql.ErrEntityConflict, werror.WithID("storage.agent_absence.read.conflict"))
	}

	if len(items) == 0 {
		return nil, werror.Wrap(dbsql.ErrNoRows, werror.WithID("storage.agent_absence.read"))
	}

	return items[0], nil
}

func (a *AgentAbsence) SearchAgentAbsence(ctx context.Context, search *options.Search) ([]*model.Absence, error) {

	panic("implement me")
}

func (a *AgentAbsence) UpdateAgentAbsence(ctx context.Context, read *options.Read, in *model.Absence) error {
	columns := map[string]any{
		"updated_by":      read.User().Id,
		"absent_at":       in.AbsentAt,
		"absence_type_id": in.AbsenceType,
	}

	ub := b.Update(b.AgentAbsenceTable.Name(), columns)
	clauses := []string{
		ub.Equal("domain_id", read.User().DomainId),
		ub.Equal("id", in.Id),
		ub.Equal("agent_id", read.DerivedByName("agent").ID()),
	}

	sql, args := ub.Where(clauses...).Build()
	if err := a.db.Primary().Exec(ctx, sql, args...); err != nil {
		return err
	}

	return nil
}

func (a *AgentAbsence) DeleteAgentAbsence(ctx context.Context, read *options.Read) error {
	db := b.Delete(b.AgentAbsenceTable.Name())
	clauses := []string{
		db.Equal("domain_id", read.User().DomainId),
		db.Equal("id", read.ID()),
		db.Equal("agent_id", read.DerivedByName("agent").ID()),
	}

	sql, args := db.Where(clauses...).Build()
	if err := a.db.Primary().Exec(ctx, sql, args...); err != nil {
		return err
	}

	return nil
}

func (a *AgentAbsence) CreateAgentsAbsences(ctx context.Context, search *options.Search, in []*model.AgentAbsences) ([]int64, error) {
	columns := make([]map[string]any, 0, len(in))
	for _, agentAbsence := range in {
		for _, absence := range agentAbsence.Absence {
			columns = append(columns, a.createAgentAbsencePrepareColumns(search.User(), agentAbsence.Agent.Id, absence))
		}
	}

	var ids []int64
	sql, args := a.createAgentAbsenceQuery(columns...)
	if err := a.db.Primary().Select(ctx, &ids, sql, args...); err != nil {
		return nil, err
	}

	return ids, nil
}

func (a *AgentAbsence) SearchAgentsAbsences(ctx context.Context, search *options.Search) ([]*model.AgentAbsences, error) {
	const (
		linkCreatedBy = 1 << iota
		linkUpdatedBy
		linkAgent
	)

	var (
		agentAbsence = b.AgentAbsenceTable
		createdBy    = b.UserTable.WithAlias("crt")
		updatedBy    = b.UserTable.WithAlias("upd")
		agent        = b.AgentTable
		agentUser    = b.UserTable.WithAlias("au")
		base         = b.Select().From(agentAbsence.String())

		join          = 0
		joinCreatedBy = func() {
			if join&linkCreatedBy != 0 {
				return
			}

			join |= linkCreatedBy
			base.JoinWithOption(
				b.LeftJoin(createdBy,
					b.JoinExpression{
						Left:  agentAbsence.Ident("created_by"),
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
						Left:  agentAbsence.Ident("updated_by"),
						Op:    "=",
						Right: updatedBy.Ident("id"),
					},
				),
			)
		}

		joinAgent = func() {
			if join&linkAgent != 0 {
				return
			}

			join |= linkAgent
			base.JoinWithOption(
				b.LeftJoin(agent,
					b.JoinExpression{
						Left:  agentAbsence.Ident("agent_id"),
						Op:    "=",
						Right: agent.Ident("id"),
					},
				),
			)

			base.JoinWithOption(
				b.LeftJoin(agentUser,
					b.JoinExpression{
						Left:  agent.Ident("user_id"),
						Op:    "=",
						Right: agentUser.Ident("id"),
					},
				),
			)
		}
	)

	{
		for _, field := range []string{"agent", "absences"} {
			search.WithField(field)
		}

		for _, field := range search.Fields() {
			switch field {
			case "agent":
				joinAgent()
				field = b.Alias(b.JSONBuildObject(b.UserLookup(agentUser)), field)

				// Apply GROUP BY here, because later we will use jsonb_agg()
				base.GroupBy(field)

			case "absences": // Nested (1 -> many), apply filters or orders here
				absencesDerived := search.DerivedByName(field)
				absencesDerivedFields := absencesDerived.Fields()
				if len(absencesDerivedFields) == 0 {
					for _, v := range []string{"id", "created_at", "updated_at", "absent_at", "absence_type_id"} {
						absencesDerivedFields.WithField(v)
					}
				}

				jsonObj := make(b.JSONBuildObjectFields, len(absencesDerivedFields))
				for _, absencesDerivedField := range absencesDerivedFields {
					switch absencesDerivedField {
					case "id", "created_at", "updated_at", "absent_at", "absence_type_id":
						jsonObj.More(b.JSONBuildObjectFields{absencesDerivedField: agentAbsence.Ident(absencesDerivedField)})

					case "created_by":
						joinCreatedBy()
						jsonObj.More(b.JSONBuildObjectFields{absencesDerivedField: b.JSONBuildObject(b.UserLookup(createdBy))})

					case "updated_by":
						joinUpdatedBy()
						jsonObj.More(b.JSONBuildObjectFields{absencesDerivedField: b.JSONBuildObject(b.UserLookup(updatedBy))})
					}
				}

				field = b.Alias(fmt.Sprintf("jsonb_agg(%s)", b.JSONBuildObject(jsonObj)), field)
			}

			base.SelectMore(field)
		}
	}

	{
		// TODO: add search by agent_id or absence id itself
		base.Where(base.EQ(agentAbsence.Ident("domain_id"), search.User().DomainId))
		if search.Query() != "" {
			base.Where(base.ILike(agentUser.Ident("name"), search.Query()))
		}

		if ids := search.IDs(); len(ids) > 0 {
			base.Where(base.In(agentAbsence.Ident("id"), b.ConvertArgs(ids)...))
		}
	}

	var items []*model.AgentAbsences
	sql, args := base.Build() // TODO: add ORDER BY clauses
	if err := a.db.StandbyPreferred().Select(ctx, &items, sql, args...); err != nil {
		return nil, err
	}

	return items, nil
}

func (a *AgentAbsence) createAgentAbsencePrepareColumns(user *model.SignedInUser, agent int64, in *model.Absence) map[string]any {
	return map[string]any{
		"domain_id":       user.DomainId,
		"created_by":      user.Id,
		"updated_by":      user.Id,
		"absent_at":       in.AbsentAt,
		"agent_id":        agent,
		"absence_type_id": int32(in.AbsenceType),
	}
}

func (a *AgentAbsence) createAgentAbsenceQuery(args ...map[string]any) (string, []any) {
	return b.Insert(b.AgentAbsenceTable.Name(), args).SQL("RETURNING id").Build()
}
