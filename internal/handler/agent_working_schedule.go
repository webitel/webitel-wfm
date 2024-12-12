package handler

import (
	"context"

	pb "github.com/webitel/webitel-wfm/gen/go/api/wfm"
	"github.com/webitel/webitel-wfm/infra/server/grpccontext"
	"github.com/webitel/webitel-wfm/internal/model"
)

type AgentWorkingScheduleService interface {
	SearchAgentWorkingSchedule(ctx context.Context, user *model.SignedInUser, search *model.SearchItem) ([]*model.AgentWorkingSchedule, []*model.Holiday, error)
}

type AgentWorkingSchedule struct {
	pb.UnimplementedAgentWorkingScheduleServiceServer

	svc AgentWorkingScheduleService
}

func NewAgentWorkingSchedule(svc AgentWorkingScheduleService) *AgentWorkingSchedule {
	return &AgentWorkingSchedule{
		svc: svc,
	}
}

func (a *AgentWorkingSchedule) SearchAgentsWorkingSchedule(ctx context.Context, req *pb.SearchAgentsWorkingScheduleRequest) (*pb.SearchAgentsWorkingScheduleResponse, error) {
	s := grpccontext.FromContext(ctx)
	search := &model.SearchItem{
		Id: req.WorkingScheduleId,
		Date: &model.FilterBetween{
			From: model.NewTimestamp(req.Date.From),
			To:   model.NewTimestamp(req.Date.To),
		},
	}

	items, holidays, err := a.svc.SearchAgentWorkingSchedule(ctx, s.SignedInUser, search)
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