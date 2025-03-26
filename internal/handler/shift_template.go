package handler

import (
	"context"

	"google.golang.org/grpc"

	pb "github.com/webitel/webitel-wfm/gen/go/api/wfm"
	"github.com/webitel/webitel-wfm/internal/model"
	"github.com/webitel/webitel-wfm/internal/model/options"
	"github.com/webitel/webitel-wfm/internal/service"
)

type ShiftTemplate struct {
	pb.UnimplementedShiftTemplateServiceServer

	service service.ShiftTemplateManager
}

func NewShiftTemplate(sr grpc.ServiceRegistrar, service service.ShiftTemplateManager) *ShiftTemplate {
	s := &ShiftTemplate{
		service: service,
	}

	pb.RegisterShiftTemplateServiceServer(sr, s)

	return s
}

func (h *ShiftTemplate) CreateShiftTemplate(ctx context.Context, req *pb.CreateShiftTemplateRequest) (*pb.CreateShiftTemplateResponse, error) {
	read, err := options.NewRead(ctx)
	if err != nil {
		return nil, err
	}

	out, err := h.service.CreateShiftTemplate(ctx, read, unmarshalShiftTemplateProto(req.GetItem()))
	if err != nil {
		return nil, err
	}

	return &pb.CreateShiftTemplateResponse{Item: out.MarshalProto()}, nil
}

func (h *ShiftTemplate) ReadShiftTemplate(ctx context.Context, req *pb.ReadShiftTemplateRequest) (*pb.ReadShiftTemplateResponse, error) {
	read, err := options.NewRead(ctx, options.WithID(req.GetId()), options.WithFields(req.GetFields()))
	if err != nil {
		return nil, err
	}

	out, err := h.service.ReadShiftTemplate(ctx, read)
	if err != nil {
		return nil, err
	}

	return &pb.ReadShiftTemplateResponse{Item: out.MarshalProto()}, nil
}

func (h *ShiftTemplate) SearchShiftTemplate(ctx context.Context, req *pb.SearchShiftTemplateRequest) (*pb.SearchShiftTemplateResponse, error) {
	opts := []options.Option{
		options.WithPagination(req.GetPage(), req.GetSize()),
		options.WithSearch(req.GetQ()),
		options.WithFields(req.GetFields()),
		options.WithOrder(req.GetSort()),
	}

	search, err := options.NewSearch(ctx, opts...)
	if err != nil {
		return nil, err
	}

	items, next, err := h.service.SearchShiftTemplate(ctx, search)
	if err != nil {
		return nil, err
	}

	return &pb.SearchShiftTemplateResponse{Items: marshalShiftTemplateBulkProto(items), Next: next}, nil
}

func (h *ShiftTemplate) UpdateShiftTemplate(ctx context.Context, req *pb.UpdateShiftTemplateRequest) (*pb.UpdateShiftTemplateResponse, error) {
	read, err := options.NewRead(ctx, options.WithID(req.Item.Id))
	if err != nil {
		return nil, err
	}

	out, err := h.service.UpdateShiftTemplate(ctx, read, unmarshalShiftTemplateProto(req.GetItem()))
	if err != nil {
		return nil, err
	}

	return &pb.UpdateShiftTemplateResponse{Item: out.MarshalProto()}, nil
}

func (h *ShiftTemplate) DeleteShiftTemplate(ctx context.Context, req *pb.DeleteShiftTemplateRequest) (*pb.DeleteShiftTemplateResponse, error) {
	read, err := options.NewRead(ctx, options.WithID(req.GetId()))
	if err != nil {
		return nil, err
	}

	id, err := h.service.DeleteShiftTemplate(ctx, read)
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
		DomainRecord: model.DomainRecord{Id: in.Id},
		Name:         in.GetName(),
		Description:  in.Description,
		Times:        times,
	}
}

func unmarshalShiftTemplateTimeProto(item *pb.ShiftTemplateTime) model.ShiftTemplateTime {
	return model.ShiftTemplateTime{
		Start: item.Start,
		End:   item.End,
	}
}
