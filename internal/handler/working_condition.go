package handler

import (
	"context"

	"google.golang.org/grpc"

	pb "github.com/webitel/webitel-wfm/gen/go/api/wfm"
	"github.com/webitel/webitel-wfm/internal/model"
	"github.com/webitel/webitel-wfm/internal/model/options"
	"github.com/webitel/webitel-wfm/internal/service"
)

type WorkingCondition struct {
	pb.UnimplementedWorkingConditionServiceServer

	service service.WorkingConditionManager
}

func NewWorkingCondition(sr grpc.ServiceRegistrar, service service.WorkingConditionManager) *WorkingCondition {
	s := &WorkingCondition{
		service: service,
	}

	pb.RegisterWorkingConditionServiceServer(sr, s)

	return s
}

func (w *WorkingCondition) CreateWorkingCondition(ctx context.Context, req *pb.CreateWorkingConditionRequest) (*pb.CreateWorkingConditionResponse, error) {
	read, err := options.NewRead(ctx)
	if err != nil {
		return nil, err
	}

	out, err := w.service.CreateWorkingCondition(ctx, read, unmarshalWorkingConditionProto(req.GetItem()))
	if err != nil {
		return nil, err
	}

	return &pb.CreateWorkingConditionResponse{Item: out.MarshalProto()}, nil
}

func (w *WorkingCondition) ReadWorkingCondition(ctx context.Context, req *pb.ReadWorkingConditionRequest) (*pb.ReadWorkingConditionResponse, error) {
	read, err := options.NewRead(ctx, options.WithID(req.GetId()), options.WithFields(req.GetFields()))
	if err != nil {
		return nil, err
	}

	out, err := w.service.ReadWorkingCondition(ctx, read)
	if err != nil {
		return nil, err
	}

	return &pb.ReadWorkingConditionResponse{Item: out.MarshalProto()}, nil
}

func (w *WorkingCondition) SearchWorkingCondition(ctx context.Context, req *pb.SearchWorkingConditionRequest) (*pb.SearchWorkingConditionResponse, error) {
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

	items, next, err := w.service.SearchWorkingCondition(ctx, search)
	if err != nil {
		return nil, err
	}

	return &pb.SearchWorkingConditionResponse{Items: marshalWorkingConditionBulkProto(items), Next: next}, nil
}

func (w *WorkingCondition) UpdateWorkingCondition(ctx context.Context, req *pb.UpdateWorkingConditionRequest) (*pb.UpdateWorkingConditionResponse, error) {
	read, err := options.NewRead(ctx, options.WithID(req.Item.Id))
	if err != nil {
		return nil, err
	}

	out, err := w.service.UpdateWorkingCondition(ctx, read, unmarshalWorkingConditionProto(req.GetItem()))
	if err != nil {
		return nil, err
	}

	return &pb.UpdateWorkingConditionResponse{Item: out.MarshalProto()}, nil
}

func (w *WorkingCondition) DeleteWorkingCondition(ctx context.Context, req *pb.DeleteWorkingConditionRequest) (*pb.DeleteWorkingConditionResponse, error) {
	read, err := options.NewRead(ctx, options.WithID(req.GetId()))
	if err != nil {
		return nil, err
	}

	id, err := w.service.DeleteWorkingCondition(ctx, read)
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
		DomainRecord:     model.DomainRecord{Id: item.Id},
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
