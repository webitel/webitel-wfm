package service

import (
	"context"

	"golang.org/x/sync/errgroup"

	"github.com/webitel/webitel-wfm/infra/webitel/engine"
	"github.com/webitel/webitel-wfm/internal/model"
	"github.com/webitel/webitel-wfm/pkg/werror"
)

var ErrAgentWorkingScheduleDateFilter = werror.InvalidArgument("invalid input: agent working schedule date filter")

type AgentWorkingScheduleStorage interface {
	SearchAgentWorkingSchedule(ctx context.Context, user *model.SignedInUser, search *model.AgentWorkingScheduleSearch) ([]*model.AgentWorkingSchedule, error)
	Holidays(ctx context.Context, user *model.SignedInUser, search *model.AgentWorkingScheduleSearch) ([]*model.Holiday, error)
}

type AgentWorkingSchedule struct {
	storage         AgentWorkingScheduleStorage
	workingSchedule WorkingScheduleStorage
	engine          *engine.Client
}

func NewAgentWorkingSchedule(storage AgentWorkingScheduleStorage, workingSchedule WorkingScheduleStorage, engine *engine.Client) *AgentWorkingSchedule {
	return &AgentWorkingSchedule{
		storage:         storage,
		workingSchedule: workingSchedule,
		engine:          engine,
	}
}

func (a *AgentWorkingSchedule) SearchAgentWorkingSchedule(ctx context.Context, user *model.SignedInUser, search *model.AgentWorkingScheduleSearch) ([]*model.AgentWorkingSchedule, []*model.Holiday, error) {
	ws, err := a.workingSchedule.ReadWorkingSchedule(ctx, user, &model.SearchItem{Id: search.WorkingScheduleId})
	if err != nil {
		return nil, nil, err
	}

	if search.SearchItem.Date.From.Valid {
		if search.SearchItem.Date.From.Time.Before(ws.StartDateAt.Time) || search.SearchItem.Date.From.Time.After(ws.EndDateAt.Time) {
			return nil, nil, werror.Wrap(ErrAgentWorkingScheduleDateFilter, werror.WithID("service.agent_working_schedule.date.from"),
				werror.AppendMessage("from date should be after (or equal) working schedule start date or before (equal) end period"),
			)
		}
	}

	if search.SearchItem.Date.To.Valid {
		if search.SearchItem.Date.To.Time.After(ws.EndDateAt.Time) || search.SearchItem.Date.To.Time.Before(ws.StartDateAt.Time) {
			return nil, nil, werror.Wrap(ErrAgentWorkingScheduleDateFilter, werror.WithID("service.agent_working_schedule.date.to"),
				werror.AppendMessage("end date should be before (or equal) working schedule end date or after (equal) start period"),
			)
		}
	}

	if len(search.SupervisorIds) > 0 || len(search.TeamIds) > 0 || len(search.SkillIds) > 0 {
		search.AgentIds, err = a.engine.Agents(ctx, &model.AgentSearch{SupervisorIds: search.SupervisorIds, TeamIds: search.TeamIds, SkillIds: search.SkillIds})
		if err != nil {
			return nil, nil, err
		}
	}

	var items []*model.AgentWorkingSchedule
	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		if items, err = a.storage.SearchAgentWorkingSchedule(ctx, user, search); err != nil {
			return err
		}

		return nil
	})

	var holidays []*model.Holiday
	eg.Go(func() error {
		if holidays, err = a.storage.Holidays(ctx, user, search); err != nil {
			return err
		}

		return nil
	})

	if err := eg.Wait(); err != nil {
		return nil, nil, err
	}

	return items, holidays, nil
}
