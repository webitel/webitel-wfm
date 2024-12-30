package service

import (
	"context"

	"golang.org/x/sync/errgroup"

	"github.com/webitel/webitel-wfm/infra/webitel/engine"
	"github.com/webitel/webitel-wfm/internal/model"
	"github.com/webitel/webitel-wfm/internal/storage"
	"github.com/webitel/webitel-wfm/pkg/timeutils"
	"github.com/webitel/webitel-wfm/pkg/werror"
)

var (
	ErrAgentWorkingScheduleDateFilter   = werror.InvalidArgument("invalid input: date should be within working schedule period", werror.WithID("service.agent_working_schedule.date"))
	ErrAgentWorkingScheduleDateShiftMap = werror.InvalidArgument("invalid input: required at least one shift day within date period", werror.WithID("service.agent_working_schedule.shift"))
)

type AgentWorkingScheduleManager interface {
	CreateAgentsWorkingScheduleShifts(ctx context.Context, user *model.SignedInUser, in *model.CreateAgentsWorkingScheduleShifts) ([]*model.AgentWorkingSchedule, error)
	SearchAgentsWorkingSchedule(ctx context.Context, user *model.SignedInUser, search *model.AgentWorkingScheduleSearch) ([]*model.AgentWorkingSchedule, []*model.Holiday, error)
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

func (a *AgentWorkingSchedule) CreateAgentsWorkingScheduleShifts(ctx context.Context, user *model.SignedInUser, in *model.CreateAgentsWorkingScheduleShifts) ([]*model.AgentWorkingSchedule, error) {
	ws, err := a.workingScheduleStorage.ReadWorkingSchedule(ctx, user, &model.SearchItem{Id: in.WorkingScheduleID})
	if err != nil {
		return nil, err
	}

	period := timeutils.NewPeriod(in.Date.From.Time, in.Date.To.Time, timeutils.IncludeAll)
	if !timeutils.NewPeriod(ws.StartDateAt.Time, ws.EndDateAt.Time, timeutils.IncludeAll).Contains(period) {
		return nil, ErrAgentWorkingScheduleDateFilter
	}

	series := period.GenerateSeries(0, 0, 1)
	schedules := make([]*model.AgentSchedule, 0, len(in.Shifts))
	for _, time := range series {
		if v, ok := in.Shifts[int64(time.Weekday())]; ok {
			schedules = append(schedules, &model.AgentSchedule{
				Date:  model.NewDate(time.Unix()),
				Shift: v,
			})
		}
	}

	if len(schedules) == 0 {
		return nil, ErrAgentWorkingScheduleDateShiftMap
	}

	agents := make([]*model.AgentWorkingSchedule, 0, len(in.Agents))
	for _, agent := range in.Agents {
		agents = append(agents, &model.AgentWorkingSchedule{
			Agent:    *agent,
			Schedule: schedules,
		})
	}

	out, err := a.storage.CreateAgentsWorkingScheduleShifts(ctx, user, ws.Id, agents)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (a *AgentWorkingSchedule) SearchAgentsWorkingSchedule(ctx context.Context, user *model.SignedInUser, search *model.AgentWorkingScheduleSearch) ([]*model.AgentWorkingSchedule, []*model.Holiday, error) {
	ws, err := a.workingScheduleStorage.ReadWorkingSchedule(ctx, user, &model.SearchItem{Id: search.WorkingScheduleId})
	if err != nil {
		return nil, nil, err
	}

	period := timeutils.NewPeriod(search.SearchItem.Date.From.Time, search.SearchItem.Date.To.Time, timeutils.IncludeAll)
	if !timeutils.NewPeriod(ws.StartDateAt.Time, ws.EndDateAt.Time, timeutils.IncludeAll).Contains(period) {
		return nil, nil, ErrAgentWorkingScheduleDateFilter
	}

	if len(search.SupervisorIds) > 0 || len(search.TeamIds) > 0 || len(search.SkillIds) > 0 {
		search.AgentIds, err = a.engine.AgentService().Agents(ctx, &model.AgentSearch{SupervisorIds: search.SupervisorIds, TeamIds: search.TeamIds, SkillIds: search.SkillIds})
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
