package handler

import (
	"context"

	pb "github.com/webitel/webitel-wfm/gen/go/api/wfm"
	"github.com/webitel/webitel-wfm/infra/server/grpccontext"
	"github.com/webitel/webitel-wfm/internal/model"
	"github.com/webitel/webitel-wfm/internal/service"
)

type AgentWorkingSchedule struct {
	pb.UnimplementedAgentWorkingScheduleServiceServer

	service service.AgentWorkingScheduleManager
}

func NewAgentWorkingSchedule(service service.AgentWorkingScheduleManager) *AgentWorkingSchedule {
	return &AgentWorkingSchedule{
		service: service,
	}
}

func (a *AgentWorkingSchedule) SearchAgentsWorkingSchedule(ctx context.Context, req *pb.SearchAgentsWorkingScheduleRequest) (*pb.SearchAgentsWorkingScheduleResponse, error) {
	s := grpccontext.FromContext(ctx)
	search := &model.AgentWorkingScheduleSearch{
		WorkingScheduleID: req.GetWorkingScheduleId(),
		SupervisorIDs:     req.GetSupervisorId(),
		TeamIDs:           req.GetTeamId(),
		SkillIDs:          req.GetSkillId(),
		SearchItem: model.SearchItem{
			Page:   req.GetPage(),
			Size:   req.GetSize(),
			Search: req.Q,
			Date: &model.FilterBetween{
				From: model.DayStart(model.DateFromMillis(req.GetDate().GetFrom())),
				To:   model.DayStart(model.DateFromMillis(req.GetDate().GetTo())),
			},
		},
	}

	out, err := a.service.SearchAgentsWorkingSchedule(ctx, s.SignedInUser, search)
	if err != nil {
		return nil, err
	}

	return &pb.SearchAgentsWorkingScheduleResponse{
		Holidays: marshalAgentWorkingScheduleHolidayBulkProto(out.Holidays),
		Items:    marshalAgentWorkingScheduleBulkProto(out.Items),
		Total:    out.Total,
		Next:     out.Next,
		Calendar: out.Calendar.MarshalProto(),
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
