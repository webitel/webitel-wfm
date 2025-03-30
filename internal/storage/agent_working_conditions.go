package storage

import (
	"context"

	b "github.com/webitel/webitel-wfm/infra/storage/dbsql/builder"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql/cluster"
	"github.com/webitel/webitel-wfm/internal/model"
	"github.com/webitel/webitel-wfm/internal/model/options"
)

type AgentWorkingConditionsManager interface {
	ReadAgentWorkingConditions(ctx context.Context, read *options.Read) (*model.AgentWorkingConditions, error)
	UpdateAgentWorkingConditions(ctx context.Context, read *options.Read, in *model.AgentWorkingConditions) error
}

type AgentWorkingConditions struct {
	db cluster.Store
}

func NewAgentWorkingConditions(db cluster.Store) *AgentWorkingConditions {
	return &AgentWorkingConditions{
		db: db,
	}
}

func (a *AgentWorkingConditions) ReadAgentWorkingConditions(ctx context.Context, read *options.Read) (*model.AgentWorkingConditions, error) {
	const (
		linkUpdatedBy = 1 << iota
		linkAgent
		linkWorkingCondition
		linkPauseTemplate
	)

	var (
		agentWorkingCondition = b.AgentWorkingConditionTable
		updatedBy             = b.UserTable.WithAlias("upd")
		agent                 = b.AgentTable
		agentUser             = b.UserTable.WithAlias("au")
		workingCondition      = b.WorkingConditionTable
		pauseTemplate         = b.PauseTemplateTable
		base                  = b.Select().From(agentWorkingCondition.String())

		join          = 0
		joinUpdatedBy = func() {
			if join&linkUpdatedBy != 0 {
				return
			}

			join |= linkUpdatedBy
			base.JoinWithOption(
				b.LeftJoin(updatedBy,
					b.JoinExpression{
						Left:  agentWorkingCondition.Ident("updated_by"),
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
						Left:  agentWorkingCondition.Ident("agent_id"),
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

		joinWorkingCondition = func() {
			if join&linkWorkingCondition != 0 {
				return
			}

			join |= linkWorkingCondition
			base.JoinWithOption(
				b.LeftJoin(workingCondition,
					b.JoinExpression{
						Left:  agentWorkingCondition.Ident("working_condition_id"),
						Op:    "=",
						Right: workingCondition.Ident("id"),
					},
				),
			)
		}

		joinPauseTemplate = func() {
			if join&linkPauseTemplate != 0 {
				return
			}

			join |= linkPauseTemplate
			base.JoinWithOption(
				b.LeftJoin(pauseTemplate,
					b.JoinExpression{
						Left:  agentWorkingCondition.Ident("pause_template_id"),
						Op:    "=",
						Right: pauseTemplate.Ident("id"),
					},
				),
			)
		}
	)

	{
		for _, field := range []string{"id", "updated_at", "updated_by", "agent"} {
			read.WithField(field)
		}

		for _, field := range read.Fields() {
			switch field {
			case "id", "domain_id", "updated_at":
				field = agentWorkingCondition.Ident(field)

			case "updated_by":
				joinUpdatedBy()
				field = b.Alias(b.JSONBuildObject(updatedBy, "id", "name"), field)

			case "agent":
				joinAgent()
				field = b.Alias(b.JSONBuildObject(agentUser, "id", "name"), field)

			case "working_condition":
				joinWorkingCondition()
				field = b.Alias(b.JSONBuildObject(workingCondition, "id", "name"), field)

			case "pause_template":
				joinPauseTemplate()
				field = b.Alias(b.JSONBuildObject(pauseTemplate, "id", "name"), field)
			}

			base.SelectMore(field)
		}
	}

	{
		base.Where(
			base.EQ(agentWorkingCondition.Ident("domain_id"), read.User().DomainId),
			base.EQ(agent.Ident("id"), read.ID()),
		)
	}

	var item model.AgentWorkingConditions
	sql, args := base.Build()
	if err := a.db.StandbyPreferred().Get(ctx, &item, sql, args...); err != nil {
		return nil, err
	}

	return &item, nil
}

func (a *AgentWorkingConditions) UpdateAgentWorkingConditions(ctx context.Context, read *options.Read, in *model.AgentWorkingConditions) error {
	columns := []map[string]any{
		{
			"domain_id":            read.User().DomainId,
			"updated_by":           read.User().Id,
			"agent_id":             read.ID(),
			"working_condition_id": in.WorkingCondition.Id,
			"pause_template_id":    in.PauseTemplate.SafeId(),
		},
	}

	sql, args := b.Insert(b.AgentWorkingConditionTable.Name(), columns).Build()
	if err := a.db.Primary().Exec(ctx, sql, args...); err != nil {
		return err
	}

	return nil
}
