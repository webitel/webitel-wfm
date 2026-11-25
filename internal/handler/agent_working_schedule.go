package handler

import (
	"context"
	"time"

	"github.com/webitel/webitel-go-kit/pkg/errors"
	"github.com/webitel/webitel-go-kit/pkg/etag"

	pb "github.com/webitel/webitel-wfm/gen/go/api/wfm"
	"github.com/webitel/webitel-wfm/infra/server/grpccontext"
	"github.com/webitel/webitel-wfm/internal/model"
	"github.com/webitel/webitel-wfm/internal/service"
)

// ErrShiftEtag: a missing, malformed or foreign etag carries no usable version, so it is a bad request, not the version conflict.
var ErrShiftEtag = errors.InvalidArgument("invalid input: the etag of this shift is required", errors.WithID("handler.shift.etag"))

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

func (a *AgentWorkingSchedule) CreateAgentsWorkingScheduleShifts(ctx context.Context, req *pb.CreateAgentsWorkingScheduleShiftsRequest) (*pb.CreateAgentsWorkingScheduleShiftsResponse, error) {
	s := grpccontext.FromContext(ctx)
	tpl := &model.WeekTemplate{
		From: time.UnixMilli(req.GetDate().GetFrom()).UTC(),
		To:   time.UnixMilli(req.GetDate().GetTo()).UTC(),
		Days: make(map[time.Weekday][]model.ShiftTemplateTime, len(req.GetDays())),
	}

	for _, agent := range req.GetAgents() {
		tpl.Agents = append(tpl.Agents, agent.GetId())
	}

	for weekday, day := range req.GetDays() {
		for _, r := range day.GetRanges() {
			tpl.Days[time.Weekday(weekday)] = append(tpl.Days[time.Weekday(weekday)], model.ShiftTemplateTime{Start: r.GetStart(), End: r.GetEnd()})
		}
	}

	for _, p := range req.GetPauses() {
		pause := model.ShiftTemplatePause{Start: p.GetStart(), Duration: p.GetDuration()}
		if p.GetCause().GetId() > 0 {
			pause.Cause = &model.LookupItem{ID: p.GetCause().GetId()}
		}

		tpl.Pauses = append(tpl.Pauses, pause)
	}

	tpl.Skills = unmarshalShiftSkills(req.GetSkills())

	items, skipped, err := a.service.CreateAgentsWorkingScheduleShifts(ctx, s.SignedInUser, req.GetWorkingScheduleId(), tpl)
	if err != nil {
		return nil, err
	}

	out := &pb.CreateAgentsWorkingScheduleShiftsResponse{Items: marshalAgentWorkingScheduleBulkProto(items)}
	for _, v := range skipped {
		out.Skipped = append(out.Skipped, v.MarshalProto())
	}

	return out, nil
}

func (a *AgentWorkingSchedule) UpdateAgentWorkingScheduleShift(ctx context.Context, req *pb.UpdateAgentWorkingScheduleShiftRequest) (*pb.UpdateAgentWorkingScheduleShiftResponse, error) {
	s := grpccontext.FromContext(ctx)
	shift := req.GetShift()

	ver, err := shiftVersion(shift.GetId(), shift.GetEtag())
	if err != nil {
		return nil, err
	}

	out, err := a.service.UpdateShift(ctx, s.SignedInUser, req.GetWorkingScheduleId(), shift.GetId(), ver, unmarshalShiftProto(shift))
	if err != nil {
		return nil, err
	}

	return &pb.UpdateAgentWorkingScheduleShiftResponse{Shift: out.MarshalProto()}, nil
}

func (a *AgentWorkingSchedule) DeleteAgentWorkingScheduleShift(ctx context.Context, req *pb.DeleteAgentWorkingScheduleShiftRequest) (*pb.DeleteAgentWorkingScheduleShiftResponse, error) {
	s := grpccontext.FromContext(ctx)

	ver, err := shiftVersion(req.GetId(), req.GetEtag())
	if err != nil {
		return nil, err
	}

	id, err := a.service.DeleteShift(ctx, s.SignedInUser, req.GetWorkingScheduleId(), req.GetId(), ver)
	if err != nil {
		return nil, err
	}

	return &pb.DeleteAgentWorkingScheduleShiftResponse{Id: id}, nil
}

func unmarshalShiftSkills(in []*pb.AgentScheduleShiftSkill) []*model.ShiftSkill {
	out := make([]*model.ShiftSkill, 0, len(in))
	for _, k := range in {
		out = append(out, &model.ShiftSkill{Skill: model.LookupItem{ID: k.GetSkill().GetId()}, Enabled: k.GetEnabled()})
	}

	return out
}

// shiftVersion is the version in the etag of shift id; any other etag is a bad request, not a conflict.
func shiftVersion(id int64, tag string) (int32, error) {
	t, err := etag.ExpectEtag(etag.EtagWorkingScheduleShift, tag)
	if err != nil || t.GetOid() != id {
		return 0, errors.Wrap(ErrShiftEtag, errors.WithCause(err))
	}

	return t.GetVer(), nil
}

func unmarshalShiftProto(in *pb.AgentScheduleShift) *model.Shift {
	pauses := make([]*model.ShiftPause, 0, len(in.GetPauses()))
	for _, p := range in.GetPauses() {
		pause := &model.ShiftPause{Start: time.UnixMilli(p.GetStartAt()), End: time.UnixMilli(p.GetEndAt())}
		if p.GetCause().GetId() > 0 {
			pause.Cause = &model.LookupItem{ID: p.GetCause().GetId()}
		}

		pauses = append(pauses, pause)
	}

	return &model.Shift{Start: time.UnixMilli(in.GetStartAt()), End: time.UnixMilli(in.GetEndAt()), Pauses: pauses, Skills: unmarshalShiftSkills(in.GetSkills())}
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
