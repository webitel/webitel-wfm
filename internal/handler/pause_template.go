package handler

import (
	"context"

	"google.golang.org/grpc"

	pb "github.com/webitel/webitel-wfm/gen/go/api/wfm"
	"github.com/webitel/webitel-wfm/internal/model"
	"github.com/webitel/webitel-wfm/internal/model/options"
	"github.com/webitel/webitel-wfm/internal/service"
)

type PauseTemplate struct {
	pb.UnimplementedPauseTemplateServiceServer

	service service.PauseTemplateManager
}

func NewPauseTemplate(sr grpc.ServiceRegistrar, service service.PauseTemplateManager) *PauseTemplate {
	s := &PauseTemplate{
		service: service,
	}

	pb.RegisterPauseTemplateServiceServer(sr, s)

	return s
}

func (h *PauseTemplate) CreatePauseTemplate(ctx context.Context, req *pb.CreatePauseTemplateRequest) (*pb.CreatePauseTemplateResponse, error) {
	read, err := options.NewRead(ctx)
	if err != nil {
		return nil, err
	}

	out, err := h.service.CreatePauseTemplate(ctx, read, unmarshalPauseTemplateProto(req.GetItem()))
	if err != nil {
		return nil, err
	}

	return &pb.CreatePauseTemplateResponse{Item: out.MarshalProto()}, nil
}

func (h *PauseTemplate) ReadPauseTemplate(ctx context.Context, req *pb.ReadPauseTemplateRequest) (*pb.ReadPauseTemplateResponse, error) {
	read, err := options.NewRead(ctx, options.WithID(req.GetId()), options.WithFields(req.GetFields()))
	if err != nil {
		return nil, err
	}

	out, err := h.service.ReadPauseTemplate(ctx, read)
	if err != nil {
		return nil, err
	}

	return &pb.ReadPauseTemplateResponse{Item: out.MarshalProto()}, nil
}

func (h *PauseTemplate) SearchPauseTemplate(ctx context.Context, req *pb.SearchPauseTemplateRequest) (*pb.SearchPauseTemplateResponse, error) {
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

	items, next, err := h.service.SearchPauseTemplate(ctx, search)
	if err != nil {
		return nil, err
	}

	return &pb.SearchPauseTemplateResponse{Items: marshalPauseTemplateBulkProto(items), Next: next}, nil
}

func (h *PauseTemplate) UpdatePauseTemplate(ctx context.Context, req *pb.UpdatePauseTemplateRequest) (*pb.UpdatePauseTemplateResponse, error) {
	read, err := options.NewRead(ctx, options.WithID(req.Item.Id))
	if err != nil {
		return nil, err
	}

	out, err := h.service.UpdatePauseTemplate(ctx, read, unmarshalPauseTemplateProto(req.GetItem()))
	if err != nil {
		return nil, err
	}

	return &pb.UpdatePauseTemplateResponse{Item: out.MarshalProto()}, nil
}

func (h *PauseTemplate) DeletePauseTemplate(ctx context.Context, req *pb.DeletePauseTemplateRequest) (*pb.DeletePauseTemplateResponse, error) {
	read, err := options.NewRead(ctx, options.WithID(req.GetId()))
	if err != nil {
		return nil, err
	}

	id, err := h.service.DeletePauseTemplate(ctx, read)
	if err != nil {
		return nil, err
	}

	return &pb.DeletePauseTemplateResponse{Id: id}, nil
}

func marshalPauseTemplateBulkProto(in []*model.PauseTemplate) []*pb.PauseTemplate {
	out := make([]*pb.PauseTemplate, 0, len(in))
	for _, t := range in {
		out = append(out, t.MarshalProto())
	}

	return out
}

func unmarshalPauseTemplateProto(in *pb.PauseTemplate) *model.PauseTemplate {
	causes := make([]model.PauseTemplateCause, 0, len(in.Causes))
	for _, cause := range in.Causes {
		causes = append(causes, unmarshalPauseTemplateCauseProto(cause))
	}

	return &model.PauseTemplate{
		DomainRecord: model.DomainRecord{Id: in.Id},
		Name:         in.GetName(),
		Description:  in.Description,
		Causes:       causes,
	}
}

func unmarshalPauseTemplateCauseProto(cause *pb.PauseTemplateCause) model.PauseTemplateCause {
	out := model.PauseTemplateCause{
		Duration: cause.Duration,
	}

	if cause.Cause != nil {
		out.Cause = &model.LookupItem{Id: cause.Cause.Id}
	}

	return out
}
