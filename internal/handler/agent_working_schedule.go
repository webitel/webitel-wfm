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

func (a *AgentWorkingSchedule) CreateAgentsWorkingScheduleShifts(ctx context.Context, req *pb.CreateAgentsWorkingScheduleShiftsRequest) (*pb.CreateAgentsWorkingScheduleShiftsResponse, error) {
	s := grpccontext.FromContext(ctx)
	agents := make([]*model.LookupItem, 0, len(req.Agents))
	for _, agent := range req.Agents {
		agents = append(agents, &model.LookupItem{Id: agent.Id})
	}

	shifts := make(map[int64]*model.AgentScheduleShift, len(req.Items))
	for k, item := range req.Items {
		shifts[k] = unmarshalAgentScheduleShift(item)
	}

	opts := &model.CreateAgentsWorkingScheduleShifts{
		WorkingScheduleID: req.WorkingScheduleId,
		Date: model.FilterBetween{
			From: model.NewTimestamp(req.Date.From),
			To:   model.NewTimestamp(req.Date.To),
		},
		Agents: agents,
		Shifts: shifts,
	}

	out, err := a.service.CreateAgentsWorkingScheduleShifts(ctx, s.SignedInUser, opts)
	if err != nil {
		return nil, err
	}

	return &pb.CreateAgentsWorkingScheduleShiftsResponse{Items: marshalAgentWorkingScheduleBulkProto(out)}, nil
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

	items, holidays, err := a.service.SearchAgentsWorkingSchedule(ctx, s.SignedInUser, search)
	if err != nil {
		return nil, err
	}

	return &pb.SearchAgentsWorkingScheduleResponse{
		Holidays: marshalAgentWorkingScheduleHolidayBulkProto(holidays),
		Items:    marshalAgentWorkingScheduleBulkProto(items),
		Total:    int64(len(items)),
	}, nil
}

func unmarshalAgentScheduleShift(in *pb.AgentScheduleShift) *model.AgentScheduleShift {
	pauses := make([]*model.AgentScheduleShiftPause, 0, len(in.Pauses))
	for _, pause := range in.Pauses {
		p := &model.AgentScheduleShiftPause{
			DomainRecord: model.DomainRecord{Id: in.Id},
			Start:        pause.Start,
			End:          pause.End,
		}

		if pause.Cause != nil {
			p.Cause = &model.LookupItem{Id: pause.Cause.Id}
		}

		pauses = append(pauses, p)
	}

	skills := make([]*model.AgentScheduleShiftSkill, 0, len(in.Skills))
	for _, skill := range in.Skills {
		skills = append(skills, &model.AgentScheduleShiftSkill{
			Skill:    model.LookupItem{Id: skill.Skill.Id},
			Capacity: skill.Capacity,
			Enabled:  skill.Enabled,
		})
	}

	return &model.AgentScheduleShift{
		DomainRecord: model.DomainRecord{Id: in.Id},
		Start:        in.Start,
		End:          in.End,
		Pauses:       pauses,
		Skills:       skills,
	}
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
