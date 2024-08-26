package handler

import (
	"context"

	pb "github.com/webitel/webitel-wfm/gen/go/api"
	"github.com/webitel/webitel-wfm/infra/server/grpccontext"
	"github.com/webitel/webitel-wfm/internal/model"
)

type PauseTemplateManager interface {
	CreatePauseTemplate(ctx context.Context, user *model.SignedInUser, in *model.PauseTemplate) (int64, error)
	ReadPauseTemplate(ctx context.Context, user *model.SignedInUser, search *model.SearchItem) (*model.PauseTemplate, error)
	SearchPauseTemplate(ctx context.Context, user *model.SignedInUser, search *model.SearchItem) ([]*model.PauseTemplate, bool, error)
	UpdatePauseTemplate(ctx context.Context, user *model.SignedInUser, in *model.PauseTemplate) error
	DeletePauseTemplate(ctx context.Context, user *model.SignedInUser, id int64) (int64, error)

	SearchPauseTemplateCause(ctx context.Context, user *model.SignedInUser, pauseTemplateId int64, search *model.SearchItem) ([]*model.PauseTemplateCause, bool, error)
	UpdatePauseTemplateCauseBulk(ctx context.Context, user *model.SignedInUser, pauseTemplateId int64, in []*model.PauseTemplateCause) error
}

type PauseTemplate struct {
	pb.UnimplementedPauseTemplateServiceServer

	svc PauseTemplateManager
}

func NewPauseTemplate(svc PauseTemplateManager) *PauseTemplate {
	return &PauseTemplate{
		svc: svc,
	}
}

func (h *PauseTemplate) CreatePauseTemplate(ctx context.Context, req *pb.CreatePauseTemplateRequest) (*pb.CreatePauseTemplateResponse, error) {
	s := grpccontext.FromContext(ctx)
	id, err := h.svc.CreatePauseTemplate(ctx, s.SignedInUser, unmarshalPauseTemplateProto(req.GetItem()))
	if err != nil {
		return nil, err
	}

	out, err := h.svc.ReadPauseTemplate(ctx, s.SignedInUser, &model.SearchItem{Id: id})
	if err != nil {
		return nil, err
	}

	return &pb.CreatePauseTemplateResponse{Item: out.MarshalProto()}, nil
}

func (h *PauseTemplate) ReadPauseTemplate(ctx context.Context, req *pb.ReadPauseTemplateRequest) (*pb.ReadPauseTemplateResponse, error) {
	s := grpccontext.FromContext(ctx)
	out, err := h.svc.ReadPauseTemplate(ctx, s.SignedInUser, &model.SearchItem{Id: req.GetId(), Fields: req.GetFields()})
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

	items, next, err := h.svc.SearchPauseTemplate(ctx, s.SignedInUser, search)
	if err != nil {
		return nil, err
	}

	return &pb.SearchPauseTemplateResponse{Items: marshalPauseTemplateBulkProto(items), Next: next}, nil
}

func (h *PauseTemplate) UpdatePauseTemplate(ctx context.Context, req *pb.UpdatePauseTemplateRequest) (*pb.UpdatePauseTemplateResponse, error) {
	s := grpccontext.FromContext(ctx)
	if err := h.svc.UpdatePauseTemplate(ctx, s.SignedInUser, unmarshalPauseTemplateProto(req.GetItem())); err != nil {
		return nil, err
	}

	out, err := h.svc.ReadPauseTemplate(ctx, s.SignedInUser, &model.SearchItem{Id: req.Item.Id})
	if err != nil {
		return nil, err
	}

	return &pb.UpdatePauseTemplateResponse{Item: out.MarshalProto()}, nil
}

func (h *PauseTemplate) DeletePauseTemplate(ctx context.Context, req *pb.DeletePauseTemplateRequest) (*pb.DeletePauseTemplateResponse, error) {
	s := grpccontext.FromContext(ctx)
	id, err := h.svc.DeletePauseTemplate(ctx, s.SignedInUser, req.Id)
	if err != nil {
		return nil, err
	}

	return &pb.DeletePauseTemplateResponse{Id: id}, nil
}

func (h *PauseTemplate) SearchPauseTemplateCause(ctx context.Context, req *pb.SearchPauseTemplateCauseRequest) (*pb.SearchPauseTemplateCauseResponse, error) {
	s := grpccontext.FromContext(ctx)
	search := &model.SearchItem{
		Page:   req.GetPage(),
		Size:   req.GetSize(),
		Search: req.Q,
		Sort:   req.Sort,
		Fields: req.Fields,
	}

	items, next, err := h.svc.SearchPauseTemplateCause(ctx, s.SignedInUser, req.PauseTemplateId, search)
	if err != nil {
		return nil, err
	}

	return &pb.SearchPauseTemplateCauseResponse{Items: marshalPauseTemplateCauseBulkProto(items), Next: next}, nil
}

func (h *PauseTemplate) UpdatePauseTemplateCauseBulk(ctx context.Context, req *pb.UpdatePauseTemplateCauseBulkRequest) (*pb.UpdatePauseTemplateCauseBulkResponse, error) {
	s := grpccontext.FromContext(ctx)
	if err := h.svc.UpdatePauseTemplateCauseBulk(ctx, s.SignedInUser, req.PauseTemplateId, unmarshalPauseTemplateCauseBulkProto(req.Items)); err != nil {
		return nil, err
	}

	items, next, err := h.svc.SearchPauseTemplateCause(ctx, s.SignedInUser, req.PauseTemplateId, &model.SearchItem{})
	if err != nil {
		return nil, err
	}

	return &pb.UpdatePauseTemplateCauseBulkResponse{Items: marshalPauseTemplateCauseBulkProto(items), Next: next}, nil
}

func marshalPauseTemplateBulkProto(templates []*model.PauseTemplate) []*pb.PauseTemplate {
	out := make([]*pb.PauseTemplate, 0, len(templates))
	for _, t := range templates {
		out = append(out, t.MarshalProto())
	}

	return out
}

func unmarshalPauseTemplateProto(template *pb.PauseTemplate) *model.PauseTemplate {
	return &model.PauseTemplate{
		Name:        template.GetName(),
		Description: template.Description,
	}
}

func marshalPauseTemplateCauseBulkProto(causes []*model.PauseTemplateCause) []*pb.PauseTemplateCause {
	out := make([]*pb.PauseTemplateCause, 0, len(causes))
	for _, c := range causes {
		out = append(out, c.MarshalProto())
	}

	return out
}

func unmarshalPauseTemplateCauseBulkProto(causes []*pb.PauseTemplateCause) []*model.PauseTemplateCause {
	out := make([]*model.PauseTemplateCause, 0, len(causes))
	for _, c := range causes {
		out = append(out, unmarshalPauseTemplateCauseProto(c))
	}

	return out
}

func unmarshalPauseTemplateCauseProto(cause *pb.PauseTemplateCause) *model.PauseTemplateCause {
	return &model.PauseTemplateCause{
		DomainRecord: model.DomainRecord{
			Id: cause.Id,
		},
		Duration: cause.Duration,
		Cause: model.LookupItem{
			Id: cause.Cause.Id,
		},
	}
}
