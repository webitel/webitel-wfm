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
		linkWorkingCondition = 1 << iota
		linkPauseTemplate
	)

	var (
		agentWorkingCondition = b.AgentWorkingConditionTable
		workingCondition      = b.WorkingConditionTable
		pauseTemplate         = b.PauseTemplateTable
		base                  = b.Select().From(agentWorkingCondition.String())

		join                 = 0
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
		for _, field := range []string{"working_condition", "pause_template"} {
			read.WithField(field)
		}

		for _, field := range read.Fields() {
			switch field {
			case "working_condition":
				joinWorkingCondition()
				field = b.Alias(b.JSONBuildObject(b.Lookup(workingCondition, "id", "name")), field)

			case "pause_template":
				joinPauseTemplate()
				field = b.Alias(b.JSONBuildObject(b.Lookup(pauseTemplate, "id", "name")), field)
			}

			base.SelectMore(field)
		}
	}

	{
		base.Where(
			base.EQ(agentWorkingCondition.Ident("domain_id"), read.User().DomainId),
			base.EQ(agentWorkingCondition.Ident("agent_id"), read.ID()),
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
