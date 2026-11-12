package handler

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"

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
	from, err := parseDate("date.from", req.GetDate().GetFrom())
	if err != nil {
		return nil, err
	}

	to, err := parseDate("date.to", req.GetDate().GetTo())
	if err != nil {
		return nil, err
	}

	s := grpccontext.FromContext(ctx)
	search := &model.AgentWorkingScheduleSearch{
		WorkingScheduleID: req.GetWorkingScheduleId(),
		SupervisorIDs:     req.GetSupervisorId(),
		TeamIDs:           req.GetTeamId(),
		SkillIDs:          req.GetSkillId(),
		SearchItem: model.SearchItem{
			Search: req.Q,
			Date: &model.FilterBetween{
				From: pgtype.Timestamp{Time: from.Time, Valid: from.Valid},
				To:   pgtype.Timestamp{Time: to.Time, Valid: to.Valid},
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
		Total:    int64(len(out.Items)),
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
