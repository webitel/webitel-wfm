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

func (a *AgentWorkingSchedule) CreateAgentsWorkingScheduleShifts(ctx context.Context, req *pb.CreateAgentsWorkingScheduleShiftsRequest) (*pb.CreateAgentsWorkingScheduleShiftsResponse, error) {
	s := grpccontext.FromContext(ctx)

	agents := make([]*model.LookupItem, 0, len(req.GetAgents()))
	for _, agent := range req.GetAgents() {
		agents = append(agents, &model.LookupItem{ID: agent.GetId()})
	}

	shifts := make(map[int64]*model.AgentScheduleShift, len(req.GetItems()))
	for k, item := range req.GetItems() {
		shifts[k] = unmarshalAgentScheduleShift(item)
	}

	opts := &model.CreateAgentsWorkingScheduleShifts{
		WorkingScheduleID: req.GetWorkingScheduleId(),
		Date: model.FilterBetween{
			From: model.NewTimestamp(req.GetDate().GetFrom()),
			To:   model.NewTimestamp(req.GetDate().GetTo()),
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
		WorkingScheduleID: req.GetWorkingScheduleId(),
		SupervisorIDs:     req.GetSupervisorId(),
		TeamIDs:           req.GetTeamId(),
		SkillIDs:          req.GetSkillId(),
		SearchItem: model.SearchItem{
			Search: req.Q,
			Date: &model.FilterBetween{
				From: model.NewTimestamp(req.GetDate().GetFrom()),
				To:   model.NewTimestamp(req.GetDate().GetTo()),
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
	pauses := make([]*model.AgentScheduleShiftPause, 0, len(in.GetPauses()))
	for _, pause := range in.GetPauses() {
		p := &model.AgentScheduleShiftPause{
			DomainRecord: model.DomainRecord{ID: pause.GetId()},
			Start:        pause.GetStart(),
			End:          pause.GetEnd(),
		}

		if pause.GetCause() != nil {
			p.Cause = &model.LookupItem{ID: pause.GetCause().GetId()}
		}

		pauses = append(pauses, p)
	}

	skills := make([]*model.AgentScheduleShiftSkill, 0, len(in.GetSkills()))
	for _, skill := range in.GetSkills() {
		skills = append(skills, &model.AgentScheduleShiftSkill{
			Skill:    model.LookupItem{ID: skill.GetSkill().GetId()},
			Capacity: skill.GetCapacity(),
			Enabled:  skill.GetEnabled(),
		})
	}

	return &model.AgentScheduleShift{
		DomainRecord: model.DomainRecord{ID: in.GetId()},
		Start:        in.GetStart(),
		End:          in.GetEnd(),
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
