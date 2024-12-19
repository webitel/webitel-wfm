package handler

import (
	"context"

	"google.golang.org/grpc"

	pb "github.com/webitel/webitel-wfm/gen/go/api/wfm"
	"github.com/webitel/webitel-wfm/infra/server/grpccontext"
	"github.com/webitel/webitel-wfm/internal/model"
	"github.com/webitel/webitel-wfm/internal/service"
)

type WorkingSchedule struct {
	pb.UnimplementedWorkingScheduleServiceServer

	service service.WorkingScheduleManager
}

func NewWorkingSchedule(sr grpc.ServiceRegistrar, service service.WorkingScheduleManager) *WorkingSchedule {
	s := &WorkingSchedule{
		service: service,
	}

	pb.RegisterWorkingScheduleServiceServer(sr, s)

	return s
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
	s := grpccontext.FromContext(ctx)
	out, err := w.service.ReadWorkingSchedule(ctx, s.SignedInUser, &model.SearchItem{Id: req.GetId(), Fields: req.GetFields()})
	if err != nil {
		return nil, err
	}

	return &pb.ReadWorkingScheduleResponse{Item: out.MarshalProto()}, nil
}

func (w *WorkingSchedule) SearchWorkingSchedule(ctx context.Context, req *pb.SearchWorkingScheduleRequest) (*pb.SearchWorkingScheduleResponse, error) {
	s := grpccontext.FromContext(ctx)
	search := &model.SearchItem{
		Page:   req.GetPage(),
		Size:   req.GetSize(),
		Search: req.Q,
		Sort:   req.Sort,
		Fields: req.Fields,
	}

	items, next, err := w.service.SearchWorkingSchedule(ctx, s.SignedInUser, search)
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
	s := grpccontext.FromContext(ctx)
	id, err := w.service.DeleteWorkingSchedule(ctx, s.SignedInUser, req.Id)
	if err != nil {
		return nil, err
	}

	return &pb.DeleteWorkingScheduleResponse{Id: id}, nil
}

func (w *WorkingSchedule) UpdateWorkingScheduleAddAgents(ctx context.Context, req *pb.UpdateWorkingScheduleAddAgentsRequest) (*pb.UpdateWorkingScheduleAddAgentsResponse, error) {
	s := grpccontext.FromContext(ctx)
	agents := make([]int64, 0, len(req.GetAgents()))
	for _, agent := range req.GetAgents() {
		agents = append(agents, agent.Id)
	}

	items, err := w.service.UpdateWorkingScheduleAddAgents(ctx, s.SignedInUser, req.WorkingScheduleId, agents)
	if err != nil {
		return nil, err
	}

	out := make([]*pb.LookupEntity, 0, len(items))
	for _, item := range items {
		out = append(out, item.MarshalProto())
	}

	return &pb.UpdateWorkingScheduleAddAgentsResponse{Agents: out}, nil
}

func (w *WorkingSchedule) UpdateWorkingScheduleRemoveAgents(ctx context.Context, req *pb.UpdateWorkingScheduleRemoveAgentsRequest) (*pb.UpdateWorkingScheduleRemoveAgentsResponse, error) {
	s := grpccontext.FromContext(ctx)
	agents := make([]int64, 0, len(req.GetAgents()))
	for _, agent := range req.GetAgents() {
		agents = append(agents, agent.Id)
	}

	items, err := w.service.UpdateWorkingScheduleRemoveAgents(ctx, s.SignedInUser, req.WorkingScheduleId, agents)
	if err != nil {
		return nil, err
	}

	out := make([]*pb.LookupEntity, 0, len(items))
	for _, item := range items {
		out = append(out, item.MarshalProto())
	}

	return &pb.UpdateWorkingScheduleRemoveAgentsResponse{Agents: out}, nil
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
