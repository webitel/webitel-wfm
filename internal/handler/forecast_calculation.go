package handler

import (
	"context"

	pb "github.com/webitel/webitel-wfm/gen/go/api/wfm"
	"github.com/webitel/webitel-wfm/infra/server/grpccontext"
	"github.com/webitel/webitel-wfm/internal/model"
)

type ForecastCalculationManager interface {
	CreateForecastCalculation(ctx context.Context, user *model.SignedInUser, in *model.ForecastCalculation) (*model.ForecastCalculation, error)
	ReadForecastCalculation(ctx context.Context, user *model.SignedInUser, search *model.SearchItem) (*model.ForecastCalculation, error)
	SearchForecastCalculation(ctx context.Context, user *model.SignedInUser, search *model.SearchItem) ([]*model.ForecastCalculation, bool, error)
	UpdateForecastCalculation(ctx context.Context, user *model.SignedInUser, in *model.ForecastCalculation) (*model.ForecastCalculation, error)
	DeleteForecastCalculation(ctx context.Context, user *model.SignedInUser, id int64) (int64, error)

	ExecuteForecastCalculation(ctx context.Context, user *model.SignedInUser, id int64) ([]*model.ForecastCalculationResult, error)
}

type ForecastCalculation struct {
	pb.UnimplementedForecastCalculationServiceServer

	svc ForecastCalculationManager
}

func NewForecastCalculation(svc ForecastCalculationManager) *ForecastCalculation {
	return &ForecastCalculation{
		svc: svc,
	}
}

func (f *ForecastCalculation) CreateForecastCalculation(ctx context.Context, req *pb.CreateForecastCalculationRequest) (*pb.CreateForecastCalculationResponse, error) {
	s := grpccontext.FromContext(ctx)
	out, err := f.svc.CreateForecastCalculation(ctx, s.SignedInUser, unmarshalForecastCalculationProto(req.GetItem()))
	if err != nil {
		return nil, err
	}

	return &pb.CreateForecastCalculationResponse{Item: out.MarshalProto()}, nil
}

func (f *ForecastCalculation) ReadForecastCalculation(ctx context.Context, req *pb.ReadForecastCalculationRequest) (*pb.ReadForecastCalculationResponse, error) {
	s := grpccontext.FromContext(ctx)
	out, err := f.svc.ReadForecastCalculation(ctx, s.SignedInUser, &model.SearchItem{Id: req.GetId(), Fields: req.GetFields()})
	if err != nil {
		return nil, err
	}

	return &pb.ReadForecastCalculationResponse{Item: out.MarshalProto()}, nil
}

func (f *ForecastCalculation) SearchForecastCalculation(ctx context.Context, req *pb.SearchForecastCalculationRequest) (*pb.SearchForecastCalculationResponse, error) {
	s := grpccontext.FromContext(ctx)
	search := &model.SearchItem{
		Page:   req.GetPage(),
		Size:   req.GetSize(),
		Search: req.Q,
		Sort:   req.Sort,
		Fields: req.Fields,
	}

	items, next, err := f.svc.SearchForecastCalculation(ctx, s.SignedInUser, search)
	if err != nil {
		return nil, err
	}

	return &pb.SearchForecastCalculationResponse{Items: marshalForecastCalculationBulkProto(items), Next: next}, nil
}

func (f *ForecastCalculation) UpdateForecastCalculation(ctx context.Context, req *pb.UpdateForecastCalculationRequest) (*pb.UpdateForecastCalculationResponse, error) {
	s := grpccontext.FromContext(ctx)
	out, err := f.svc.UpdateForecastCalculation(ctx, s.SignedInUser, unmarshalForecastCalculationProto(req.GetItem()))
	if err != nil {
		return nil, err
	}

	return &pb.UpdateForecastCalculationResponse{Item: out.MarshalProto()}, nil
}

func (f *ForecastCalculation) DeleteForecastCalculation(ctx context.Context, req *pb.DeleteForecastCalculationRequest) (*pb.DeleteForecastCalculationResponse, error) {
	s := grpccontext.FromContext(ctx)
	id, err := f.svc.DeleteForecastCalculation(ctx, s.SignedInUser, req.Id)
	if err != nil {
		return nil, err
	}

	return &pb.DeleteForecastCalculationResponse{Id: id}, nil
}

func (f *ForecastCalculation) ExecuteForecastCalculation(ctx context.Context, req *pb.ExecuteForecastCalculationRequest) (*pb.ExecuteForecastCalculationResponse, error) {
	s := grpccontext.FromContext(ctx)
	out, err := f.svc.ExecuteForecastCalculation(ctx, s.SignedInUser, req.Id)
	if err != nil {
		return nil, err
	}

	return &pb.ExecuteForecastCalculationResponse{Items: marshalForecastCalculationResultsProto(out)}, nil
}

func unmarshalForecastCalculationProto(in *pb.ForecastCalculation) *model.ForecastCalculation {
	return &model.ForecastCalculation{
		DomainRecord: model.DomainRecord{Id: in.Id},
		Name:         in.GetName(),
		Description:  in.Description,
		Procedure:    in.Procedure,
	}
}

func marshalForecastCalculationBulkProto(in []*model.ForecastCalculation) []*pb.ForecastCalculation {
	out := make([]*pb.ForecastCalculation, 0, len(in))
	for _, i := range in {
		out = append(out, i.MarshalProto())
	}

	return out
}

func marshalForecastCalculationResultsProto(in []*model.ForecastCalculationResult) []*pb.ExecuteForecastCalculationResponse_Forecast {
	out := make([]*pb.ExecuteForecastCalculationResponse_Forecast, 0, len(in))
	for _, i := range in {
		out = append(out, i.MarshalProto())
	}

	return out
}
