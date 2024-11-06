package handler

import (
	"context"

	pb "github.com/webitel/webitel-wfm/gen/go/api/wfm"
	"github.com/webitel/webitel-wfm/infra/server/grpccontext"
	"github.com/webitel/webitel-wfm/internal/model"
)

type WorkingScheduleService interface {
	CreateWorkingSchedule(ctx context.Context, user *model.SignedInUser, in *model.WorkingSchedule) (*model.WorkingSchedule, error)
	ReadWorkingSchedule(ctx context.Context, user *model.SignedInUser, search *model.SearchItem) (*model.WorkingSchedule, error)
	SearchWorkingSchedule(ctx context.Context, user *model.SignedInUser, search *model.SearchItem) ([]*model.WorkingSchedule, bool, error)
	UpdateWorkingSchedule(ctx context.Context, user *model.SignedInUser, in *model.WorkingSchedule) (*model.WorkingSchedule, error)
	DeleteWorkingSchedule(ctx context.Context, user *model.SignedInUser, id int64) (int64, error)
}

type WorkingSchedule struct {
	pb.UnimplementedWorkingScheduleServiceServer

	svc WorkingScheduleService
}

func NewWorkingSchedule(svc WorkingScheduleService) *WorkingSchedule {
	return &WorkingSchedule{
		svc: svc,
	}
}

func (w *WorkingSchedule) CreateWorkingSchedule(ctx context.Context, req *pb.CreateWorkingScheduleRequest) (*pb.CreateWorkingScheduleResponse, error) {
	s := grpccontext.FromContext(ctx)
	out, err := w.svc.CreateWorkingSchedule(ctx, s.SignedInUser, unmarshalWorkingScheduleProto(req.GetItem()))
	if err != nil {
		return nil, err
	}

	return &pb.CreateWorkingScheduleResponse{Item: out.MarshalProto()}, nil
}

func (w *WorkingSchedule) ReadWorkingSchedule(ctx context.Context, req *pb.ReadWorkingScheduleRequest) (*pb.ReadWorkingScheduleResponse, error) {
	s := grpccontext.FromContext(ctx)
	out, err := w.svc.ReadWorkingSchedule(ctx, s.SignedInUser, &model.SearchItem{Id: req.GetId(), Fields: req.GetFields()})
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

	items, next, err := w.svc.SearchWorkingSchedule(ctx, s.SignedInUser, search)
	if err != nil {
		return nil, err
	}

	return &pb.SearchWorkingScheduleResponse{Items: marshalWorkingScheduleBulkProto(items), Next: next}, nil
}

func (w *WorkingSchedule) UpdateWorkingSchedule(ctx context.Context, req *pb.UpdateWorkingScheduleRequest) (*pb.UpdateWorkingScheduleResponse, error) {
	panic("implement me")
}

func (w *WorkingSchedule) DeleteWorkingSchedule(ctx context.Context, req *pb.DeleteWorkingScheduleRequest) (*pb.DeleteWorkingScheduleResponse, error) {
	panic("implement me")
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
		State:                int32(in.State.Number()),
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
