package storage

import (
	"context"

	"github.com/webitel/webitel-wfm/infra/storage/dbsql"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql/cluster"
	"github.com/webitel/webitel-wfm/internal/model"
)

const (
	agentWorkingConditionsTable = "wfm.agent_working_conditions"
	agentWorkingConditionsView  = agentWorkingConditionsTable + "_v"
)

type AgentWorkingConditions struct {
	db cluster.Store
}

func NewAgentWorkingConditions(db cluster.Store) *AgentWorkingConditions {
	return &AgentWorkingConditions{
		db: db,
	}
}

func (a *AgentWorkingConditions) ReadAgentWorkingConditions(ctx context.Context, user *model.SignedInUser, agentId int64) (*model.AgentWorkingConditions, error) {
	var item model.AgentWorkingConditions

	columns := []string{dbsql.Wildcard(model.AgentWorkingConditions{})}
	sb := a.db.SQL().Select(columns...).From(agentWorkingConditionsView)
	sql, args := sb.Where(sb.Equal("domain_id", user.DomainId), sb.Equal("agent_id", agentId)).Build()
	if err := a.db.StandbyPreferred().Get(ctx, &item, sql, args...); err != nil {
		return nil, err
	}

	return &item, nil
}

func (a *AgentWorkingConditions) UpdateAgentWorkingConditions(ctx context.Context, user *model.SignedInUser, agentId int64, in *model.AgentWorkingConditions) error {
	columns := map[string]any{
		"domain_id":            user.DomainId,
		"updated_by":           user.Id,
		"agent_id":             agentId,
		"working_condition_id": in.WorkingCondition.Id,
		"pause_template_id":    in.PauseTemplate.Id,
	}

	sql, args := a.db.SQL().Insert(agentWorkingConditionsTable, columns).Build()
	if err := a.db.Primary().Exec(ctx, sql, args...); err != nil {
		return err
	}

	return nil
}

//
// func (a *Agent) ReadAgentWorkingConditions(ctx context.Context, search *model.SearchItem) (*model.AgentWorkingConditions, error) {
// 	var item model.AgentWorkingConditions
//
// 	columns := []string{
// 		"json_build_object('id', cw.id, 'name', cw.name) AS working_condition",
// 		"json_build_object('id', pt.id, 'name', pt.name) AS pause_template",
// 	}
//
// 	query, args := a.db.SQL().Select(columns...).
// 		From("wfm.agent_working_conditions a").
// 		LeftJoin("wfm.working_condition cw ON cw.id = a.working_condition_id").
// 		LeftJoin("wfm.pause_template pt ON pt.id = a.pause_template_id").
// 		Where(sq.Eq{"a.domain_id": search.SignedInUser.DomainId, "a.agent_id": search.Id}).Limit(1).
// 		MustSql()
//
// 	if err := a.db.StandbyPreferred().Conn().Get(ctx, &item, query, args...); err != nil {
// 		if err.GetDetail() != apperrors.ErrDBNoRows.Error() {
// 			return nil, err
// 		}
// 	}
//
// 	return &item, nil
// }
//
// func (a *Agent) UpdateAgentWorkingConditions(ctx context.Context, search *model.SearchItem, item *model.AgentWorkingConditions) error {
// 	columns := map[string]any{
// 		"domain_id":            search.SignedInUser.DomainId,
// 		"updated_by":           search.SignedInUser.Id,
// 		"updated_at":           time.Now().UTC(),
// 		"agent_id":             search.Id,
// 		"working_condition_id": item.WorkingCondition.Id,
// 		"pause_template_id":    item.PauseTemplate.Id,
// 	}
//
// 	query, args := a.db.SQL().Insert("wfm.agent_working_conditions").SetMap(columns).
// 		Suffix(`ON CONFLICT(domain_id, agent_id)
// 					DO UPDATE SET working_condition_id = EXCLUDED.working_condition_id
// 						, pause_template_id = EXCLUDED.pause_template_id
// 						, updated_by = EXCLUDED.updated_by, updated_at = EXCLUDED.updated_at`).MustSql()
//
// 	if err := a.db.Primary().Conn().Exec(ctx, query, args...); err != nil {
// 		return err
// 	}
//
// 	return nil
// }
