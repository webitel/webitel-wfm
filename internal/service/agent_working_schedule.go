package service

import (
	"context"

	"golang.org/x/sync/errgroup"

	"github.com/webitel/webitel-wfm/internal/model"
	"github.com/webitel/webitel-wfm/pkg/werror"
)

var ErrAgentWorkingScheduleDateFilter = werror.InvalidArgument("invalid input: agent working schedule date filter")

type AgentWorkingScheduleStorage interface {
	SearchAgentWorkingSchedule(ctx context.Context, user *model.SignedInUser, search *model.SearchItem) ([]*model.AgentWorkingSchedule, error)
	Holidays(ctx context.Context, user *model.SignedInUser, search *model.SearchItem) ([]*model.Holiday, error)
}

type AgentWorkingSchedule struct {
	storage         AgentWorkingScheduleStorage
	workingSchedule WorkingScheduleStorage
}

func NewAgentWorkingSchedule(storage AgentWorkingScheduleStorage, workingSchedule WorkingScheduleStorage) *AgentWorkingSchedule {
	return &AgentWorkingSchedule{
		storage:         storage,
		workingSchedule: workingSchedule,
	}
}

func (a *AgentWorkingSchedule) SearchAgentWorkingSchedule(ctx context.Context, user *model.SignedInUser, search *model.SearchItem) ([]*model.AgentWorkingSchedule, []*model.Holiday, error) {
	ws, err := a.workingSchedule.ReadWorkingSchedule(ctx, user, &model.SearchItem{Id: search.Id})
	if err != nil {
		return nil, nil, err
	}

	if search.Date.From.Valid {
		if search.Date.From.Time.Before(ws.StartDateAt.Time) || search.Date.From.Time.After(ws.EndDateAt.Time) {
			return nil, nil, werror.Wrap(ErrAgentWorkingScheduleDateFilter, werror.WithID("service.agent_working_schedule.date.from"),
				werror.AppendMessage("from date should be after (or equal) working schedule start date or before (equal) end period"),
			)
		}
	}

	if search.Date.To.Valid {
		if search.Date.To.Time.After(ws.EndDateAt.Time) || search.Date.To.Time.Before(ws.StartDateAt.Time) {
			return nil, nil, werror.Wrap(ErrAgentWorkingScheduleDateFilter, werror.WithID("service.agent_working_schedule.date.to"),
				werror.AppendMessage("end date should be before (or equal) working schedule end date or after (equal) start period"),
			)
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
