package storage

import (
	"context"
	"strconv"

	"github.com/webitel/webitel-wfm/infra/storage/cache"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql/cluster"
	"github.com/webitel/webitel-wfm/internal/model"
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

func (a *AgentAbsence) CreateAgentAbsence(ctx context.Context, user *model.SignedInUser, in *model.AgentAbsence) (int64, error) {
	var id int64
	columns := map[string]any{
		"domain_id":       user.DomainId,
		"created_by":      user.Id,
		"updated_by":      user.Id,
		"absent_at":       in.Absence.AbsentAt,
		"agent_id":        in.Agent.Id,
		"absence_type_id": in.Absence.AbsenceTypeId,
	}

	sql, args := a.db.SQL().Insert(agentAbsenceTable, columns).SQL("RETURNING id").Build()
	if err := a.db.Primary().Get(ctx, &id, sql, args...); err != nil {
		return 0, err
	}
	go a.cache.Key(user.DomainId, 0, user).Delete(ctx)

	return id, nil
}

func (a *AgentAbsence) UpdateAgentAbsence(ctx context.Context, user *model.SignedInUser, in *model.AgentAbsence) error {
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
		return err
	}

	return nil
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

func (a *AgentAbsence) CreateAgentsAbsencesBulk(ctx context.Context, user *model.SignedInUser, agentIds []int64, in []*model.AgentAbsenceBulk) ([]int64, error) {
	var ids []int64

	absences := a.db.SQL().Values()
	for _, v := range in {
		// TODO: int to string conversion used for emitting error "unable to encode 2 (as int arg) into text format for text"
		absences.Values(strconv.FormatInt(v.AbsenceTypeId, 10), strconv.FormatInt(v.AbsentAtFrom, 10), strconv.FormatInt(v.AbsentAtTo, 10))
	}

	cte := a.db.SQL().CTE(
		a.db.SQL().With("agents").As(
			a.db.SQL().Format("select unnest($?::int[]) id", agentIds),
		),
		a.db.SQL().With("absences").As(
			a.db.SQL().Format("select absence_type_id::bigint, to_timestamp(absent_at_from::bigint) absent_at_from, to_timestamp(absent_at_to::bigint) absent_at_to from ($?) x (absence_type_id, absent_at_from, absent_at_to)", absences),
		),
		a.db.SQL().With("record").As(
			a.db.SQL().Format("select $?::int as domain_id, $?::bigint as created_by, $?::bigint as updated_by", user.DomainId, user.Id, user.Id),
		),
	)

	sb := a.db.SQL().Select("r.domain_id", "r.created_by", "r.updated_by", "s", "a.id", "ab.absence_type_id").With(cte)
	sb.From(
		sb.As("record", "r"),
		sb.As("agents", "a"),
		sb.As("absences", "ab"),
		sb.As("generate_series(ab.absent_at_from::date, ab.absent_at_to::date, '1d'::interval)", "s"),
	)

	ib := a.db.SQL().Insert("wfm.agent_absence", nil).Cols("domain_id", "created_by", "updated_by", "absent_at", "agent_id", "absence_type_id")
	sib := ib.Select("*")
	sib.From(sib.BuilderAs(sb, "x")).SQL("RETURNING id")

	sql, args := ib.Build()
	if err := a.db.Primary().Select(ctx, &ids, sql, args...); err != nil {
		return nil, err
	}

	return ids, nil
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
