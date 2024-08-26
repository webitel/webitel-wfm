package handler

import (
	"context"

	pb "github.com/webitel/webitel-wfm/gen/go/api"
	"github.com/webitel/webitel-wfm/infra/server/grpccontext"
	"github.com/webitel/webitel-wfm/internal/model"
)

type WorkingConditionManager interface {
	CreateWorkingCondition(ctx context.Context, user *model.SignedInUser, in *model.WorkingCondition) (int64, error)
	ReadWorkingCondition(ctx context.Context, user *model.SignedInUser, search *model.SearchItem) (*model.WorkingCondition, error)
	SearchWorkingCondition(ctx context.Context, user *model.SignedInUser, search *model.SearchItem) ([]*model.WorkingCondition, bool, error)
	UpdateWorkingCondition(ctx context.Context, user *model.SignedInUser, in *model.WorkingCondition) error
	DeleteWorkingCondition(ctx context.Context, user *model.SignedInUser, id int64) (int64, error)
}

type WorkingCondition struct {
	pb.UnimplementedWorkingConditionServiceServer

	svc WorkingConditionManager
}

func NewWorkingCondition(svc WorkingConditionManager) *WorkingCondition {
	return &WorkingCondition{
		svc: svc,
	}
}

func (w *WorkingCondition) CreateWorkingCondition(ctx context.Context, req *pb.CreateWorkingConditionRequest) (*pb.CreateWorkingConditionResponse, error) {
	s := grpccontext.FromContext(ctx)
	id, err := w.svc.CreateWorkingCondition(ctx, s.SignedInUser, unmarshalWorkingConditionProto(req.GetItem()))
	if err != nil {
		return nil, err
	}

	out, err := w.svc.ReadWorkingCondition(ctx, s.SignedInUser, &model.SearchItem{Id: id})
	if err != nil {
		return nil, err
	}

	return &pb.CreateWorkingConditionResponse{Item: out.MarshalProto()}, nil
}

func (w *WorkingCondition) ReadWorkingCondition(ctx context.Context, req *pb.ReadWorkingConditionRequest) (*pb.ReadWorkingConditionResponse, error) {
	s := grpccontext.FromContext(ctx)
	search := &model.SearchItem{
		Id:     req.Id,
		Fields: req.Fields,
	}

	out, err := w.svc.ReadWorkingCondition(ctx, s.SignedInUser, search)
	if err != nil {
		return nil, err
	}

	return &pb.ReadWorkingConditionResponse{Item: out.MarshalProto()}, nil
}

func (w *WorkingCondition) SearchWorkingCondition(ctx context.Context, req *pb.SearchWorkingConditionRequest) (*pb.SearchWorkingConditionResponse, error) {
	s := grpccontext.FromContext(ctx)
	search := &model.SearchItem{
		Page:   req.GetPage(),
		Size:   req.GetSize(),
		Search: req.Q,
		Sort:   req.Sort,
		Fields: req.Fields,
	}

	items, next, err := w.svc.SearchWorkingCondition(ctx, s.SignedInUser, search)
	if err != nil {
		return nil, err
	}

	return &pb.SearchWorkingConditionResponse{Items: marshalWorkingConditionBulkProto(items), Next: next}, nil
}

func (w *WorkingCondition) UpdateWorkingCondition(ctx context.Context, req *pb.UpdateWorkingConditionRequest) (*pb.UpdateWorkingConditionResponse, error) {
	s := grpccontext.FromContext(ctx)
	if err := w.svc.UpdateWorkingCondition(ctx, s.SignedInUser, unmarshalWorkingConditionProto(req.GetItem())); err != nil {
		return nil, err
	}

	out, err := w.svc.ReadWorkingCondition(ctx, s.SignedInUser, &model.SearchItem{Id: req.Item.Id})
	if err != nil {
		return nil, err
	}

	return &pb.UpdateWorkingConditionResponse{Item: out.MarshalProto()}, nil
}

func (w *WorkingCondition) DeleteWorkingCondition(ctx context.Context, req *pb.DeleteWorkingConditionRequest) (*pb.DeleteWorkingConditionResponse, error) {
	s := grpccontext.FromContext(ctx)
	id, err := w.svc.DeleteWorkingCondition(ctx, s.SignedInUser, req.Id)
	if err != nil {
		return nil, err
	}

	return &pb.DeleteWorkingConditionResponse{Id: id}, nil
}

func marshalWorkingConditionBulkProto(items []*model.WorkingCondition) []*pb.WorkingCondition {
	out := make([]*pb.WorkingCondition, 0, len(items))
	for _, t := range items {
		out = append(out, t.MarshalProto())
	}

	return out
}

func unmarshalWorkingConditionProto(item *pb.WorkingCondition) *model.WorkingCondition {
	out := &model.WorkingCondition{
		Name:             item.GetName(),
		Description:      item.Description,
		WorkdayHours:     item.WorkdayHours,
		WorkdaysPerMonth: item.WorkdaysPerMonth,
		Vacation:         item.Vacation,
		SickLeaves:       item.SickLeaves,
		DaysOff:          item.DaysOff,
		PauseDuration:    item.PauseDuration,
		PauseTemplate: model.LookupItem{
			Id: item.PauseTemplate.Id,
		},
	}

	if item.ShiftTemplate != nil {
		out.ShiftTemplate = &model.LookupItem{
			Id: item.ShiftTemplate.Id,
		}
	}

	return out
}
