package storage

import (
	"context"

	"github.com/webitel/webitel-wfm/infra/storage/cache"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql/cluster"
	"github.com/webitel/webitel-wfm/internal/model"
	"github.com/webitel/webitel-wfm/pkg/werror"
)

const (
	agentAbsenceTable = "wfm.agent_absence"
	agentAbsenceView  = agentAbsenceTable + "_v"
	agentAbsenceAcl   = agentAbsenceTable + "_acl"
)

type AgentAbsence struct {
	db    cluster.Store
	cache *cache.Scope[model.AgentAbsence]
}

func NewAgentAbsence(db cluster.Store, manager cache.Manager) *AgentAbsence {
	return &AgentAbsence{
		db:    db,
		cache: cache.NewScope[model.AgentAbsence](manager, agentAbsenceTable),
	}
}

func (a *AgentAbsence) CreateAgentAbsence(ctx context.Context, user *model.SignedInUser, in *model.AgentAbsence) (*model.AgentAbsence, error) {
	var id int64

	sql, args := a.createAgentAbsenceQuery(user, in)
	if err := a.db.Primary().Get(ctx, &id, sql, args...); err != nil {
		return nil, err
	}

	out, err := a.ReadAgentAbsence(ctx, user, &model.SearchItem{Id: id})
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (a *AgentAbsence) ReadAgentAbsence(ctx context.Context, user *model.SignedInUser, search *model.SearchItem) (*model.AgentAbsence, error) {
	item, err := a.ReadAgentAbsences(ctx, user, &model.AgentAbsenceSearch{SearchItem: *search})
	if err != nil {
		return nil, err
	}

	out := &model.AgentAbsence{
		Agent:   item.Agent,
		Absence: item.Absence[0],
	}

	return out, nil
}

func (a *AgentAbsence) UpdateAgentAbsence(ctx context.Context, user *model.SignedInUser, in *model.AgentAbsence) (*model.AgentAbsence, error) {
	ub := a.db.SQL().Update(agentAbsenceTable)
	assignments := []string{
		ub.Assign("updated_by", user.Id),
		ub.Assign("absent_at", in.Absence.AbsentAt),
		ub.Assign("absence_type_id", in.Absence.AbsenceTypeId),
	}

	clauses := []string{
		ub.Equal("domain_id", user.DomainId),
		ub.Equal("id", in.Absence.Id),
		ub.Equal("agent_id", in.Agent.Id),
	}

	sql, args := ub.Set(assignments...).Where(clauses...).AddWhereClause(a.db.SQL().RBAC(user.UseRBAC, agentAbsenceAcl, in.Absence.Id, user.DomainId, user.Groups, user.Access)).Build()
	if err := a.db.Primary().Exec(ctx, sql, args...); err != nil {
		return nil, err
	}

	out, err := a.ReadAgentAbsence(ctx, user, &model.SearchItem{Id: in.Agent.Id})
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (a *AgentAbsence) DeleteAgentAbsence(ctx context.Context, user *model.SignedInUser, agentId, id int64) error {
	db := a.db.SQL().Delete(agentAbsenceTable)
	clauses := []string{
		db.Equal("domain_id", user.DomainId),
		db.Equal("id", id),
		db.Equal("agent_id", agentId),
	}

	sql, args := db.Where(clauses...).AddWhereClause(a.db.SQL().RBAC(user.UseRBAC, agentAbsenceAcl, id, user.DomainId, user.Groups, user.Access)).Build()
	if err := a.db.Primary().Exec(ctx, sql, args...); err != nil {
		return err
	}

	return nil
}

func (a *AgentAbsence) CreateAgentsAbsencesBulk(ctx context.Context, user *model.SignedInUser, agentIds []int64, in []*model.AgentAbsenceBulk) ([]*model.AgentAbsences, error) {
	batch := a.db.Primary().WithBatch()
	for _, agentId := range agentIds {
		for _, absence := range in {
			start := model.NewDate(absence.AbsentAtFrom)
			end := model.NewDate(absence.AbsentAtTo)

			for d := start; d.Time.After(end.Time) == false; d.Time = d.Time.AddDate(0, 0, 1) {
				req := &model.AgentAbsence{
					Agent: model.LookupItem{
						Id: agentId,
					},
					Absence: model.Absence{
						AbsenceTypeId: absence.AbsenceTypeId,
						AbsentAt:      d,
					},
				}

				batch.Queue(a.createAgentAbsenceQuery(user, req))
			}
		}
	}

	var ids []int64
	if err := batch.Select(ctx, &ids); err != nil {
		return nil, err
	}

	out, err := a.SearchAgentsAbsences(ctx, user, &model.AgentAbsenceSearch{Ids: ids})
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (a *AgentAbsence) ReadAgentAbsences(ctx context.Context, user *model.SignedInUser, search *model.AgentAbsenceSearch) (*model.AgentAbsences, error) {
	items, err := a.SearchAgentsAbsences(ctx, user, search)
	if err != nil {
		return nil, err
	}

	if len(items) > 1 {
		return nil, werror.NewDBEntityConflictError("storage.agent_absence.read.conflict")
	}

	if len(items) == 0 {
		return nil, werror.NewDBNoRowsErr("storage.agent_absence.read")
	}

	return items[0], nil
}

func (a *AgentAbsence) SearchAgentsAbsences(ctx context.Context, user *model.SignedInUser, search *model.AgentAbsenceSearch) ([]*model.AgentAbsences, error) {
	var items []*model.AgentAbsences
	var defaultSort = "agent"

	if search.SearchItem.Sort == nil {
		sort := "agent"
		search.SearchItem.Sort = &sort
	}

	columns := []string{dbsql.Wildcard(model.Absence{})}
	if len(search.SearchItem.Fields) > 0 {
		columns = search.SearchItem.Fields
	}

	columns = append(columns, "agent")
	ssb := a.db.SQL().Select(columns...)
	ssb.From(ssb.As(agentAbsenceView, "v")).
		Where(ssb.Equal("domain_id", user.DomainId)).
		AddWhereClause(&search.Where("agent ->> 'name'").WhereClause).
		AddWhereClause(a.db.SQL().RBAC(user.UseRBAC, agentAbsenceAcl, 0, user.DomainId, user.Groups, user.Access))

	if search.SearchItem.Sort != &defaultSort {
		ssb.OrderBy(search.SearchItem.OrderBy(pauseTemplateView))
	}

	sb := a.db.SQL().Select("agent", "json_agg(row_to_json(x)) absence")
	if search.SearchItem.Sort == &defaultSort {
		sb.OrderBy(search.SearchItem.OrderBy(pauseTemplateView))
	}

	sql, args := sb.From(sb.BuilderAs(ssb, "x")).
		GroupBy("agent").
		Limit(int(search.SearchItem.Limit())).
		Offset(int(search.SearchItem.Offset())).
		Build()

	if err := a.db.StandbyPreferred().Select(ctx, &items, sql, args...); err != nil {
		return nil, err
	}

	return items, nil
}

func (a *AgentAbsence) createAgentAbsenceQuery(user *model.SignedInUser, in *model.AgentAbsence) (string, []any) {
	columns := map[string]any{
		"domain_id":       user.DomainId,
		"created_by":      user.Id,
		"updated_by":      user.Id,
		"absent_at":       in.Absence.AbsentAt,
		"agent_id":        in.Agent.Id,
		"absence_type_id": in.Absence.AbsenceTypeId,
	}

	return a.db.SQL().Insert(agentAbsenceTable, columns).SQL("RETURNING id").Build()
}
