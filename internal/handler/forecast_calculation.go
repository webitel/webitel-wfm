package handler

import (
	"context"

	"google.golang.org/grpc"

	pb "github.com/webitel/webitel-wfm/gen/go/api/wfm"
	"github.com/webitel/webitel-wfm/infra/server/grpccontext"
	"github.com/webitel/webitel-wfm/internal/model"
	"github.com/webitel/webitel-wfm/internal/service"
)

type ForecastCalculation struct {
	pb.UnimplementedForecastCalculationServiceServer

	service service.ForecastCalculationManager
}

func NewForecastCalculation(sr grpc.ServiceRegistrar, service service.ForecastCalculationManager) *ForecastCalculation {
	s := &ForecastCalculation{
		service: service,
	}

	pb.RegisterForecastCalculationServiceServer(sr, s)

	return s
}

func (f *ForecastCalculation) CreateForecastCalculation(ctx context.Context, req *pb.CreateForecastCalculationRequest) (*pb.CreateForecastCalculationResponse, error) {
	s := grpccontext.FromContext(ctx)
	out, err := f.service.CreateForecastCalculation(ctx, s.SignedInUser, unmarshalForecastCalculationProto(req.GetItem()))
	if err != nil {
		return nil, err
	}

	return &pb.CreateForecastCalculationResponse{Item: out.MarshalProto()}, nil
}

func (f *ForecastCalculation) ReadForecastCalculation(ctx context.Context, req *pb.ReadForecastCalculationRequest) (*pb.ReadForecastCalculationResponse, error) {
	s := grpccontext.FromContext(ctx)
	out, err := f.service.ReadForecastCalculation(ctx, s.SignedInUser, &model.SearchItem{Id: req.GetId(), Fields: req.GetFields()})
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

	items, next, err := f.service.SearchForecastCalculation(ctx, s.SignedInUser, search)
	if err != nil {
		return nil, err
	}

	return &pb.SearchForecastCalculationResponse{Items: marshalForecastCalculationBulkProto(items), Next: next}, nil
}

func (f *ForecastCalculation) UpdateForecastCalculation(ctx context.Context, req *pb.UpdateForecastCalculationRequest) (*pb.UpdateForecastCalculationResponse, error) {
	s := grpccontext.FromContext(ctx)
	out, err := f.service.UpdateForecastCalculation(ctx, s.SignedInUser, unmarshalForecastCalculationProto(req.GetItem()))
	if err != nil {
		return nil, err
	}

	return &pb.UpdateForecastCalculationResponse{Item: out.MarshalProto()}, nil
}

func (f *ForecastCalculation) DeleteForecastCalculation(ctx context.Context, req *pb.DeleteForecastCalculationRequest) (*pb.DeleteForecastCalculationResponse, error) {
	s := grpccontext.FromContext(ctx)
	id, err := f.service.DeleteForecastCalculation(ctx, s.SignedInUser, req.Id)
	if err != nil {
		return nil, err
	}

	return &pb.DeleteForecastCalculationResponse{Id: id}, nil
}

func (f *ForecastCalculation) ExecuteForecastCalculation(ctx context.Context, req *pb.ExecuteForecastCalculationRequest) (*pb.ExecuteForecastCalculationResponse, error) {
	s := grpccontext.FromContext(ctx)
	forecast := &model.FilterBetween{
		From: model.NewTimestamp(req.ForecastData.From),
		To:   model.NewTimestamp(req.ForecastData.To),
	}

	out, err := f.service.ExecuteForecastCalculation(ctx, s.SignedInUser, req.Id, req.TeamId, forecast)
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
		Args:         in.Args,
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
