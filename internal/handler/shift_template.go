package handler

import (
	"context"

	pb "github.com/webitel/webitel-wfm/gen/go/api"
	"github.com/webitel/webitel-wfm/infra/server/grpccontext"
	"github.com/webitel/webitel-wfm/internal/model"
)

type ShiftTemplateManager interface {
	CreateShiftTemplate(ctx context.Context, user *model.SignedInUser, in *model.ShiftTemplate) (int64, error)
	ReadShiftTemplate(ctx context.Context, user *model.SignedInUser, id int64, fields []string) (*model.ShiftTemplate, error)
	SearchShiftTemplate(ctx context.Context, user *model.SignedInUser, search *model.SearchItem) ([]*model.ShiftTemplate, bool, error)
	UpdateShiftTemplate(ctx context.Context, user *model.SignedInUser, in *model.ShiftTemplate) error
	DeleteShiftTemplate(ctx context.Context, user *model.SignedInUser, id int64) (int64, error)
}

type ShiftTemplate struct {
	pb.UnimplementedShiftTemplateServiceServer

	svc ShiftTemplateManager
}

func NewShiftTemplate(svc ShiftTemplateManager) *ShiftTemplate {
	return &ShiftTemplate{
		svc: svc,
	}
}

func (h *ShiftTemplate) CreateShiftTemplate(ctx context.Context, req *pb.CreateShiftTemplateRequest) (*pb.CreateShiftTemplateResponse, error) {
	s := grpccontext.FromContext(ctx)
	id, err := h.svc.CreateShiftTemplate(ctx, s.SignedInUser, unmarshalShiftTemplateProto(req.GetItem()))
	if err != nil {
		return nil, err
	}

	out, err := h.svc.ReadShiftTemplate(ctx, s.SignedInUser, id, nil)
	if err != nil {
		return nil, err
	}

	return &pb.CreateShiftTemplateResponse{Item: out.MarshalProto()}, nil
}

func (h *ShiftTemplate) ReadShiftTemplate(ctx context.Context, req *pb.ReadShiftTemplateRequest) (*pb.ReadShiftTemplateResponse, error) {
	s := grpccontext.FromContext(ctx)
	out, err := h.svc.ReadShiftTemplate(ctx, s.SignedInUser, req.Id, req.Fields)
	if err != nil {
		return nil, err
	}

	return &pb.ReadShiftTemplateResponse{Item: out.MarshalProto()}, nil
}

func (h *ShiftTemplate) SearchShiftTemplate(ctx context.Context, req *pb.SearchShiftTemplateRequest) (*pb.SearchShiftTemplateResponse, error) {
	s := grpccontext.FromContext(ctx)
	search := &model.SearchItem{
		Page:   req.GetPage(),
		Size:   req.GetSize(),
		Search: req.Q,
		Sort:   req.Sort,
		Fields: req.Fields,
	}

	items, next, err := h.svc.SearchShiftTemplate(ctx, s.SignedInUser, search)
	if err != nil {
		return nil, err
	}

	return &pb.SearchShiftTemplateResponse{Items: marshalShiftTemplateBulkProto(items), Next: next}, nil
}

func (h *ShiftTemplate) UpdateShiftTemplate(ctx context.Context, req *pb.UpdateShiftTemplateRequest) (*pb.UpdateShiftTemplateResponse, error) {
	s := grpccontext.FromContext(ctx)
	if err := h.svc.UpdateShiftTemplate(ctx, s.SignedInUser, unmarshalShiftTemplateProto(req.GetItem())); err != nil {
		return nil, err
	}

	out, err := h.svc.ReadShiftTemplate(ctx, s.SignedInUser, req.Item.Id, nil)
	if err != nil {
		return nil, err
	}

	return &pb.UpdateShiftTemplateResponse{Item: out.MarshalProto()}, nil
}

func (h *ShiftTemplate) DeleteShiftTemplate(ctx context.Context, req *pb.DeleteShiftTemplateRequest) (*pb.DeleteShiftTemplateResponse, error) {
	s := grpccontext.FromContext(ctx)
	id, err := h.svc.DeleteShiftTemplate(ctx, s.SignedInUser, req.Id)
	if err != nil {
		return nil, err
	}

	return &pb.DeleteShiftTemplateResponse{Id: id}, nil
}

func marshalShiftTemplateBulkProto(templates []*model.ShiftTemplate) []*pb.ShiftTemplate {
	out := make([]*pb.ShiftTemplate, 0, len(templates))
	for _, t := range templates {
		out = append(out, t.MarshalProto())
	}

	return out
}

func unmarshalShiftTemplateProto(in *pb.ShiftTemplate) *model.ShiftTemplate {
	times := make([]model.ShiftTemplateTime, 0, len(in.Times))
	for _, t := range in.Times {
		times = append(times, unmarshalShiftTemplateTimeProto(t))
	}

	return &model.ShiftTemplate{
		Name:        in.GetName(),
		Description: in.Description,
		Times:       times,
	}
}

func unmarshalShiftTemplateTimeProto(item *pb.ShiftTemplateTime) model.ShiftTemplateTime {
	return model.ShiftTemplateTime{
		Start: item.Start,
		End:   item.End,
	}
}
