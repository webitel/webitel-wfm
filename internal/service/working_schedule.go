package service

import (
	"context"

	"github.com/webitel/webitel-go-kit/pkg/errors"

	"github.com/webitel/webitel-wfm/infra/webitel/engine"
	"github.com/webitel/webitel-wfm/internal/model"
	"github.com/webitel/webitel-wfm/internal/model/options"
	"github.com/webitel/webitel-wfm/internal/storage"
	"github.com/webitel/webitel-wfm/pkg/compare"
	"github.com/webitel/webitel-wfm/pkg/timeutils"
)

var (
	ErrWorkingScheduleUpdateDraft = errors.InvalidArgument("working schedule can only be updated in a draft state", errors.WithID("service.working_schedule.state"))
	ErrAgentNotAllowed            = errors.Forbidden("you haven't read access to a desired set of agents")
	ErrEmptyForecastCalculation   = errors.InvalidArgument("attached team doesn't have configured forecast calculation procedure", errors.WithID("service.working_schedule.empty_forecast_calculation"))
)

type WorkingScheduleManager interface {
	CreateWorkingSchedule(ctx context.Context, user *model.SignedInUser, in *model.WorkingSchedule) (*model.WorkingSchedule, error)
	ReadWorkingSchedule(ctx context.Context, read *options.Read) (*model.WorkingSchedule, error)

	ReadWorkingScheduleForecast(ctx context.Context, read *options.Read, date *model.FilterBetween) ([]*model.ForecastCalculationResult, error)

	SearchWorkingSchedule(ctx context.Context, search *options.Search) ([]*model.WorkingSchedule, bool, error)
	UpdateWorkingSchedule(ctx context.Context, user *model.SignedInUser, in *model.WorkingSchedule) (*model.WorkingSchedule, error)
	DeleteWorkingSchedule(ctx context.Context, read *options.Read) (int64, error)

	UpdateWorkingScheduleAddAgents(ctx context.Context, read *options.Read, agentIDs []int64) ([]*model.LookupItem, error)
	UpdateWorkingScheduleRemoveAgent(ctx context.Context, read *options.Read, agentID int64) (int64, error)
}

type WorkingSchedule struct {
	storage storage.WorkingScheduleManager

	engine   *engine.Client
	forecast ForecastCalculationManager
}

func NewWorkingSchedule(storage storage.WorkingScheduleManager, engine *engine.Client) *WorkingSchedule {
	return &WorkingSchedule{
		storage: storage,
		engine:  engine,
	}
}

func (w *WorkingSchedule) CreateWorkingSchedule(ctx context.Context, user *model.SignedInUser, in *model.WorkingSchedule) (*model.WorkingSchedule, error) {
	agentIDs, err := w.engine.AgentService().Agents(ctx, &model.AgentSearch{TeamIds: []int64{in.Team.Id}})
	if err != nil {
		return nil, err
	}

	agents := make([]*model.LookupItem, 0, len(agentIDs))
	for _, a := range agentIDs {
		agents = append(agents, &model.LookupItem{Id: a})
	}

	in.Agents = agents

	out, err := w.storage.CreateWorkingSchedule(ctx, user, in)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (w *WorkingSchedule) ReadWorkingSchedule(ctx context.Context, read *options.Read) (*model.WorkingSchedule, error) {
	out, err := w.storage.ReadWorkingSchedule(ctx, read)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (w *WorkingSchedule) ReadWorkingScheduleForecast(ctx context.Context, read *options.Read, date *model.FilterBetween) ([]*model.ForecastCalculationResult, error) {
	ws, err := w.storage.ReadWorkingSchedule(ctx, read)
	if err != nil {
		return nil, err
	}

	if !date.From.Valid {
		date.From = model.NewTimestamp(ws.StartDateAt.Time.Unix())
	}

	if !date.To.Valid {
		date.To = model.NewTimestamp(ws.EndDateAt.Time.Unix())
	}

	period := timeutils.NewPeriod(date.From.Time, date.To.Time, timeutils.IncludeAll)
	if !timeutils.NewPeriod(ws.StartDateAt.Time, ws.EndDateAt.Time, timeutils.IncludeAll).Contains(period) {
		return nil, ErrAgentWorkingScheduleDateFilter
	}

	team, err := w.engine.TeamService().Team(ctx, ws.Team.Id)
	if err != nil {
		return nil, err
	}

	if team.ForecastCalculation == nil || team.ForecastCalculation.Id == 0 {
		return nil, errors.Wrap(ErrEmptyForecastCalculation, errors.WithValue("team", team.Name))
	}

	forecast, err := w.forecast.ExecuteForecastCalculation(ctx, read.User(), team.GetForecastCalculation().GetId(), team.GetId(), date)
	if err != nil {
		return nil, err
	}

	return forecast, nil
}

func (w *WorkingSchedule) SearchWorkingSchedule(ctx context.Context, search *options.Search) ([]*model.WorkingSchedule, bool, error) {
	out, err := w.storage.SearchWorkingSchedule(ctx, search)
	if err != nil {
		return nil, false, err
	}

	next, out := model.ListResult(int32(search.Size()), out)

	return out, next, nil
}

func (w *WorkingSchedule) UpdateWorkingSchedule(ctx context.Context, user *model.SignedInUser, in *model.WorkingSchedule) (*model.WorkingSchedule, error) {
	read, err := options.NewRead(ctx, options.WithID(in.Id))
	if err != nil {
		return nil, err
	}

	item, err := w.ReadWorkingSchedule(ctx, read)
	if err != nil {
		return nil, err
	}

	if item.State != model.WorkingScheduleStateDraft {
		return nil, errors.Wrap(ErrWorkingScheduleUpdateDraft, errors.WithValue("state", item.State.String()))
	}

	if item.Team.Id != in.Team.Id || item.Calendar.Id != in.Calendar.Id {
		if _, err := w.DeleteWorkingSchedule(ctx, read); err != nil {
			return nil, err
		}

		out, err := w.CreateWorkingSchedule(ctx, user, item)
		if err != nil {
			return nil, err
		}

		return out, nil
	}

	out, err := w.storage.UpdateWorkingSchedule(ctx, user, in)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (w *WorkingSchedule) DeleteWorkingSchedule(ctx context.Context, read *options.Read) (int64, error) {
	out, err := w.storage.DeleteWorkingSchedule(ctx, read)
	if err != nil {
		return 0, err
	}

	return out, nil
}

func (w *WorkingSchedule) UpdateWorkingScheduleAddAgents(ctx context.Context, read *options.Read, agentIDs []int64) ([]*model.LookupItem, error) {
	agents, err := w.engine.AgentService().Agents(ctx, &model.AgentSearch{Ids: agentIDs})
	if err != nil {
		return nil, err
	}

	// Checks if signed user has read access to a desired set of agents.
	if ok := compare.ElementsMatch(agents, agentIDs); !ok {
		return nil, errors.Wrap(ErrAgentNotAllowed, errors.WithID("service.working_schedule.check_agents"))
	}

	out, err := w.storage.UpdateWorkingScheduleAddAgents(ctx, read, agentIDs)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (w *WorkingSchedule) UpdateWorkingScheduleRemoveAgent(ctx context.Context, read *options.Read, agentID int64) (int64, error) {
	if _, err := w.engine.AgentService().Agent(ctx, agentID); err != nil {
		return 0, err
	}

	out, err := w.storage.UpdateWorkingScheduleRemoveAgent(ctx, read, agentID)
	if err != nil {
		return 0, err
	}

	return out, nil
}
