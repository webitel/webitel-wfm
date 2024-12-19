package handler

import (
	"context"

	"google.golang.org/grpc"

	pb "github.com/webitel/webitel-wfm/gen/go/api/wfm"
	"github.com/webitel/webitel-wfm/infra/server/grpccontext"
	"github.com/webitel/webitel-wfm/internal/model"
	"github.com/webitel/webitel-wfm/internal/service"
)

type AgentWorkingSchedule struct {
	pb.UnimplementedAgentWorkingScheduleServiceServer

	service service.AgentWorkingScheduleManager
}

func NewAgentWorkingSchedule(sr grpc.ServiceRegistrar, service service.AgentWorkingScheduleManager) *AgentWorkingSchedule {
	s := &AgentWorkingSchedule{
		service: service,
	}

	pb.RegisterAgentWorkingScheduleServiceServer(sr, s)

	return s
}

func (a *AgentWorkingSchedule) SearchAgentsWorkingSchedule(ctx context.Context, req *pb.SearchAgentsWorkingScheduleRequest) (*pb.SearchAgentsWorkingScheduleResponse, error) {
	s := grpccontext.FromContext(ctx)
	search := &model.AgentWorkingScheduleSearch{
		WorkingScheduleId: req.WorkingScheduleId,
		SupervisorIds:     req.SupervisorId,
		TeamIds:           req.TeamId,
		SkillIds:          req.SkillId,
		SearchItem: model.SearchItem{
			Search: req.Q,
			Date: &model.FilterBetween{
				From: model.NewTimestamp(req.Date.From),
				To:   model.NewTimestamp(req.Date.To),
			},
		},
	}

	items, holidays, err := a.service.SearchAgentWorkingSchedule(ctx, s.SignedInUser, search)
	if err != nil {
		return nil, err
	}

	return &pb.SearchAgentsWorkingScheduleResponse{
		Holidays: marshalAgentWorkingScheduleHolidayBulkProto(holidays),
		Items:    marshalAgentWorkingScheduleBulkProto(items),
		Total:    int64(len(items)),
	}, nil
}

func marshalAgentWorkingScheduleBulkProto(in []*model.AgentWorkingSchedule) []*pb.AgentWorkingSchedule {
	out := make([]*pb.AgentWorkingSchedule, 0, len(in))
	for _, i := range in {
		out = append(out, i.MarshalProto())
	}

	return out
}

func marshalAgentWorkingScheduleHolidayBulkProto(in []*model.Holiday) []*pb.Holiday {
	out := make([]*pb.Holiday, 0, len(in))
	for _, i := range in {
		out = append(out, i.MarshalProto())
	}

	return out
}
