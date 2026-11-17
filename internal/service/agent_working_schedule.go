package service

import (
	"context"

	"golang.org/x/sync/errgroup"

	"github.com/webitel/webitel-go-kit/pkg/errors"

	"github.com/webitel/webitel-wfm/infra/webitel/engine"
	"github.com/webitel/webitel-wfm/internal/model"
	"github.com/webitel/webitel-wfm/internal/model/options"
	"github.com/webitel/webitel-wfm/internal/storage"
	"github.com/webitel/webitel-wfm/pkg/timeutils"
)

var ErrAgentWorkingScheduleDateFilter = errors.InvalidArgument("invalid input: date should be within working schedule period", errors.WithID("service.agent_working_schedule.date"))

type AgentWorkingScheduleManager interface {
	SearchAgentsWorkingSchedule(ctx context.Context, user *model.SignedInUser, search *model.AgentWorkingScheduleSearch) (*model.AgentWorkingScheduleCalendar, error)
}
type AgentWorkingSchedule struct {
	storage                storage.AgentWorkingScheduleManager
	workingScheduleStorage storage.WorkingScheduleManager
	engine                 *engine.Client
}

func NewAgentWorkingSchedule(storage storage.AgentWorkingScheduleManager, workingScheduleStorage storage.WorkingScheduleManager, engine *engine.Client) *AgentWorkingSchedule {
	return &AgentWorkingSchedule{
		storage:                storage,
		workingScheduleStorage: workingScheduleStorage,
		engine:                 engine,
	}
}

func (a *AgentWorkingSchedule) SearchAgentsWorkingSchedule(ctx context.Context, user *model.SignedInUser, search *model.AgentWorkingScheduleSearch) (*model.AgentWorkingScheduleCalendar, error) {
	read, err := options.NewRead(ctx, options.WithID(search.WorkingScheduleID))
	if err != nil {
		return nil, err
	}

	ws, err := a.workingScheduleStorage.ReadWorkingSchedule(ctx, read)
	if err != nil {
		return nil, err
	}

	if search.SearchItem.Date == nil {
		search.SearchItem.Date = &model.FilterBetween{}
	}

	date := search.SearchItem.Date
	if !date.From.Valid {
		date.From = model.DayStart(ws.StartDateAt)
	}

	if !date.To.Valid {
		date.To = model.DayStart(ws.EndDateAt)
	}

	period := timeutils.NewPeriod(date.From.Time, date.To.Time, timeutils.IncludeAll)
	if !timeutils.NewPeriod(ws.StartDateAt.Time, ws.EndDateAt.Time, timeutils.IncludeAll).Contains(period) {
		return nil, ErrAgentWorkingScheduleDateFilter
	}

	filtered := len(search.SupervisorIDs) > 0 || len(search.TeamIDs) > 0 || len(search.SkillIDs) > 0
	if filtered {
		search.AgentIDs, err = a.engine.AgentService().Agents(ctx, &model.AgentSearch{SupervisorIDs: search.SupervisorIDs, TeamIDs: search.TeamIDs, SkillIDs: search.SkillIDs})
		if err != nil {
			return nil, err
		}
	}

	var out model.AgentWorkingScheduleCalendar

	eg, ctx := errgroup.WithContext(ctx)

	// A filter that matched no agent leaves AgentIds empty, which storage reads
	// as no filter at all: list nobody rather than everyone.
	if !filtered || len(search.AgentIDs) > 0 {
		eg.Go(func() error {
			var err error
			if out.Items, err = a.storage.SearchAgentWorkingSchedule(ctx, user, search); err != nil {
				return err
			}

			return nil
		})
	}

	eg.Go(func() error {
		var err error
		if out.Holidays, err = a.storage.Holidays(ctx, user, search); err != nil {
			return err
		}

		return nil
	})

	eg.Go(func() error {
		var err error
		if out.Calendar, err = a.storage.ScheduleCalendar(ctx, user, search.WorkingScheduleID); err != nil {
			return err
		}

		return nil
	})

	if err := eg.Wait(); err != nil {
		return nil, err
	}

	if len(out.Items) > 0 {
		out.Total = out.Items[0].Total
	}

	out.Next, out.Items = model.ListResult(search.SearchItem.Limit(), out.Items)

	return &out, nil
}
