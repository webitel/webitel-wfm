package handler

import (
	"context"
	"fmt"

	"google.golang.org/grpc"

	pb "github.com/webitel/webitel-wfm/gen/go/api/wfm"
	"github.com/webitel/webitel-wfm/infra/server/grpccontext"
	"github.com/webitel/webitel-wfm/internal/model"
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
	s := grpccontext.FromContext(ctx)
	id, err := h.service.CreatePauseTemplate(ctx, s.SignedInUser, unmarshalPauseTemplateProto(req.GetItem()))
	if err != nil {
		return nil, err
	}

	out, err := h.service.ReadPauseTemplate(ctx, s.SignedInUser, id, nil)
	if err != nil {
		return nil, err
	}

	return &pb.CreatePauseTemplateResponse{Item: out.MarshalProto()}, nil
}

func (h *PauseTemplate) ReadPauseTemplate(ctx context.Context, req *pb.ReadPauseTemplateRequest) (*pb.ReadPauseTemplateResponse, error) {
	s := grpccontext.FromContext(ctx)
	out, err := h.service.ReadPauseTemplate(ctx, s.SignedInUser, req.GetId(), req.GetFields())
	if err != nil {
		return nil, err
	}

	return &pb.ReadPauseTemplateResponse{Item: out.MarshalProto()}, nil
}

func (h *PauseTemplate) SearchPauseTemplate(ctx context.Context, req *pb.SearchPauseTemplateRequest) (*pb.SearchPauseTemplateResponse, error) {
	s := grpccontext.FromContext(ctx)
	search := &model.SearchItem{
		Page:   req.GetPage(),
		Size:   req.GetSize(),
		Search: req.Q,
		Sort:   req.Sort,
		Fields: req.Fields,
	}

	items, next, err := h.service.SearchPauseTemplate(ctx, s.SignedInUser, search)
	if err != nil {
		return nil, err
	}

	return &pb.SearchPauseTemplateResponse{Items: marshalPauseTemplateBulkProto(items), Next: next}, nil
}

func (h *PauseTemplate) UpdatePauseTemplate(ctx context.Context, req *pb.UpdatePauseTemplateRequest) (*pb.UpdatePauseTemplateResponse, error) {
	s := grpccontext.FromContext(ctx)
	if err := h.service.UpdatePauseTemplate(ctx, s.SignedInUser, unmarshalPauseTemplateProto(req.GetItem())); err != nil {
		return nil, err
	}

	out, err := h.service.ReadPauseTemplate(ctx, s.SignedInUser, req.Item.Id, nil)
	if err != nil {
		return nil, err
	}

	return &pb.UpdatePauseTemplateResponse{Item: out.MarshalProto()}, nil
}

func (h *PauseTemplate) DeletePauseTemplate(ctx context.Context, req *pb.DeletePauseTemplateRequest) (*pb.DeletePauseTemplateResponse, error) {
	s := grpccontext.FromContext(ctx)
	id, err := h.service.DeletePauseTemplate(ctx, s.SignedInUser, req.Id)
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

	fmt.Println(out, cause)

	return out
}
