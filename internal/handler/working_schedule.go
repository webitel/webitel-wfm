package handler

import (
	"context"

	pb "github.com/webitel/webitel-wfm/gen/go/api/wfm"
	"github.com/webitel/webitel-wfm/infra/server/grpccontext"
	"github.com/webitel/webitel-wfm/internal/model"
	"github.com/webitel/webitel-wfm/internal/model/options"
	"github.com/webitel/webitel-wfm/internal/service"
	"github.com/webitel/webitel-wfm/pkg/timeutils"
)

type WorkingSchedule struct {
	pb.UnimplementedWorkingScheduleServiceServer

	service service.WorkingScheduleManager
}

func NewWorkingSchedule(service service.WorkingScheduleManager) *WorkingSchedule {
	return &WorkingSchedule{
		service: service,
	}
}

func (w *WorkingSchedule) CreateWorkingSchedule(ctx context.Context, req *pb.CreateWorkingScheduleRequest) (*pb.CreateWorkingScheduleResponse, error) {
	s := grpccontext.FromContext(ctx)

	out, err := w.service.CreateWorkingSchedule(ctx, s.SignedInUser, unmarshalWorkingScheduleProto(req.GetItem()))
	if err != nil {
		return nil, err
	}

	return &pb.CreateWorkingScheduleResponse{Item: out.MarshalProto()}, nil
}

func (w *WorkingSchedule) ReadWorkingSchedule(ctx context.Context, req *pb.ReadWorkingScheduleRequest) (*pb.ReadWorkingScheduleResponse, error) {
	read, err := options.NewRead(ctx, options.WithID(req.GetId()), options.WithFields(req.GetFields()))
	if err != nil {
		return nil, err
	}

	out, err := w.service.ReadWorkingSchedule(ctx, read)
	if err != nil {
		return nil, err
	}

	return &pb.ReadWorkingScheduleResponse{Item: out.MarshalProto()}, nil
}

func (w *WorkingSchedule) ReadWorkingScheduleForecast(ctx context.Context, req *pb.ReadWorkingScheduleForecastRequest) (*pb.ReadWorkingScheduleForecastResponse, error) {
	read, err := options.NewRead(ctx, options.WithID(req.GetId()))
	if err != nil {
		return nil, err
	}

	date := &model.FilterBetween{}
	if v := req.Date; v != nil {
		date = &model.FilterBetween{
			From: model.NewTimestamp(v.From),
			To:   model.NewTimestamp(v.To),
		}
	}

	items, err := w.service.ReadWorkingScheduleForecast(ctx, read, date)
	if err != nil {
		return nil, err
	}

	out := make(map[int64]*pb.WorkingScheduleForecast)
	for _, i := range items {
		day := timeutils.Date(i.Timestamp.Time).Unix()
		if _, ok := out[day]; !ok {
			out[day] = &pb.WorkingScheduleForecast{Forecast: make([]*pb.WorkingScheduleForecast_Forecast, 0)}
		}

		out[day].Forecast = append(out[day].Forecast, &pb.WorkingScheduleForecast_Forecast{
			Hour:   int64(i.Timestamp.Time.Hour()),
			Agents: *i.Agents,
		})
	}

	return &pb.ReadWorkingScheduleForecastResponse{Items: out}, nil
}

func (w *WorkingSchedule) SearchWorkingSchedule(ctx context.Context, req *pb.SearchWorkingScheduleRequest) (*pb.SearchWorkingScheduleResponse, error) {
	opts := []options.Option{
		options.WithPagination(req.GetPage(), req.GetSize()),
		options.WithSearch(req.GetQ()),
		options.WithFields(req.GetFields()),
		options.WithOrder(req.GetSort()),
	}

	search, err := options.NewSearch(ctx, opts...)
	if err != nil {
		return nil, err
	}

	items, next, err := w.service.SearchWorkingSchedule(ctx, search)
	if err != nil {
		return nil, err
	}

	return &pb.SearchWorkingScheduleResponse{Items: marshalWorkingScheduleBulkProto(items), Next: next}, nil
}

func (w *WorkingSchedule) UpdateWorkingSchedule(ctx context.Context, req *pb.UpdateWorkingScheduleRequest) (*pb.UpdateWorkingScheduleResponse, error) {
	s := grpccontext.FromContext(ctx)

	out, err := w.service.UpdateWorkingSchedule(ctx, s.SignedInUser, unmarshalWorkingScheduleProto(req.GetItem()))
	if err != nil {
		return nil, err
	}

	return &pb.UpdateWorkingScheduleResponse{Item: out.MarshalProto()}, nil
}

func (w *WorkingSchedule) DeleteWorkingSchedule(ctx context.Context, req *pb.DeleteWorkingScheduleRequest) (*pb.DeleteWorkingScheduleResponse, error) {
	read, err := options.NewRead(ctx, options.WithID(req.GetId()))
	if err != nil {
		return nil, err
	}

	id, err := w.service.DeleteWorkingSchedule(ctx, read)
	if err != nil {
		return nil, err
	}

	return &pb.DeleteWorkingScheduleResponse{Id: id}, nil
}

func (w *WorkingSchedule) UpdateWorkingScheduleAddAgents(ctx context.Context, req *pb.UpdateWorkingScheduleAddAgentsRequest) (*pb.UpdateWorkingScheduleAddAgentsResponse, error) {
	read, err := options.NewRead(ctx, options.WithID(req.GetId()))
	if err != nil {
		return nil, err
	}

	agents := make([]int64, 0, len(req.GetAgents()))
	for _, agent := range req.GetAgents() {
		agents = append(agents, agent.Id)
	}

	items, err := w.service.UpdateWorkingScheduleAddAgents(ctx, read, agents)
	if err != nil {
		return nil, err
	}

	out := make([]*pb.LookupEntity, 0, len(items))
	for _, item := range items {
		out = append(out, item.MarshalProto())
	}

	return &pb.UpdateWorkingScheduleAddAgentsResponse{Agents: out}, nil
}

func (w *WorkingSchedule) UpdateWorkingScheduleRemoveAgents(ctx context.Context, req *pb.UpdateWorkingScheduleRemoveAgentRequest) (*pb.UpdateWorkingScheduleRemoveAgentResponse, error) {
	read, err := options.NewRead(ctx, options.WithID(req.GetId()))
	if err != nil {
		return nil, err
	}

	out, err := w.service.UpdateWorkingScheduleRemoveAgent(ctx, read, req.GetAgentId())
	if err != nil {
		return nil, err
	}

	return &pb.UpdateWorkingScheduleRemoveAgentResponse{Id: out}, nil
}

func unmarshalWorkingScheduleProto(in *pb.WorkingSchedule) *model.WorkingSchedule {
	skills := make([]*model.LookupItem, 0, len(in.ExtraSkills))
	for _, skill := range in.ExtraSkills {
		skills = append(skills, &model.LookupItem{Id: skill.Id})
	}

	agents := make([]*model.LookupItem, 0, len(in.Agents))
	for _, agent := range in.Agents {
		agents = append(agents, &model.LookupItem{Id: agent.Id})
	}

	return &model.WorkingSchedule{
		DomainRecord:         model.DomainRecord{Id: in.Id},
		Name:                 in.Name,
		State:                model.WorkingScheduleState(in.State.Number()),
		Team:                 model.LookupItem{Id: in.Team.Id},
		Calendar:             model.LookupItem{Id: in.Calendar.Id},
		StartDateAt:          model.NewDate(in.StartDateAt),
		EndDateAt:            model.NewDate(in.EndDateAt),
		StartTimeAt:          in.StartTimeAt,
		EndTimeAt:            in.EndTimeAt,
		ExtraSkills:          skills,
		BlockOutsideActivity: in.BlockOutsideActivity,
		Agents:               agents,
	}
}

func marshalWorkingScheduleBulkProto(in []*model.WorkingSchedule) []*pb.WorkingSchedule {
	out := make([]*pb.WorkingSchedule, 0, len(in))
	for _, i := range in {
		out = append(out, i.MarshalProto())
	}

	return out
}
