package handler

import (
	"context"

	"google.golang.org/grpc"

	pb "github.com/webitel/webitel-wfm/gen/go/api/wfm"
	"github.com/webitel/webitel-wfm/internal/model"
	"github.com/webitel/webitel-wfm/internal/model/options"
	"github.com/webitel/webitel-wfm/internal/service"
)

type AgentAbsence struct {
	pb.UnimplementedAgentAbsenceServiceServer

	service service.AgentAbsenceManager
}

func NewAgentAbsence(sr grpc.ServiceRegistrar, service service.AgentAbsenceManager) *AgentAbsence {
	s := &AgentAbsence{
		service: service,
	}

	pb.RegisterAgentAbsenceServiceServer(sr, s)

	return s
}

func (a *AgentAbsence) CreateAgentAbsence(ctx context.Context, req *pb.CreateAgentAbsenceRequest) (*pb.CreateAgentAbsenceResponse, error) {
	read, err := options.NewRead(ctx, options.WithDerivedID("agent", req.GetAgentId()))
	if err != nil {
		return nil, err
	}

	out, err := a.service.CreateAgentAbsence(ctx, read, unmarshalAbsenceProto(req.GetItem()))
	if err != nil {
		return nil, err
	}

	return &pb.CreateAgentAbsenceResponse{Item: out.MarshalProto()}, nil
}

func (a *AgentAbsence) ReadAgentAbsence(ctx context.Context, req *pb.ReadAgentAbsenceRequest) (*pb.ReadAgentAbsenceResponse, error) {
	read, err := options.NewRead(ctx, options.WithID(req.GetId()), options.WithDerivedID("agent", req.GetAgentId()))
	if err != nil {
		return nil, err
	}

	out, err := a.service.ReadAgentAbsence(ctx, read)
	if err != nil {
		return nil, err
	}

	return &pb.ReadAgentAbsenceResponse{Item: out.MarshalProto()}, nil
}

func (a *AgentAbsence) SearchAgentAbsence(ctx context.Context, req *pb.SearchAgentAbsenceRequest) (*pb.SearchAgentAbsenceResponse, error) {
	opts := []options.Option{
		options.WithDerivedID("agent", req.GetAgentId()),
		options.WithPagination(req.GetPage(), req.GetSize()),
		options.WithFields(req.GetFields()),
		options.WithOrder(req.GetSort()),
		// TODO: Add filters.
	}

	search, err := options.NewSearch(ctx, opts...)
	if err != nil {
		return nil, err
	}

	out, err := a.service.SearchAgentAbsence(ctx, search)
	if err != nil {
		return nil, err
	}

	return &pb.SearchAgentAbsenceResponse{Items: marshalAbsenceBulkProto(out)}, nil
}

func (a *AgentAbsence) UpdateAgentAbsence(ctx context.Context, req *pb.UpdateAgentAbsenceRequest) (*pb.UpdateAgentAbsenceResponse, error) {
	read, err := options.NewRead(ctx, options.WithID(req.GetItem().GetId()), options.WithDerivedID("agent", req.GetAgentId()))
	if err != nil {
		return nil, err
	}

	out, err := a.service.UpdateAgentAbsence(ctx, read, unmarshalAbsenceProto(req.GetItem()))
	if err != nil {
		return nil, err
	}

	return &pb.UpdateAgentAbsenceResponse{Item: out.MarshalProto()}, nil
}

func (a *AgentAbsence) DeleteAgentAbsence(ctx context.Context, req *pb.DeleteAgentAbsenceRequest) (*pb.DeleteAgentAbsenceResponse, error) {
	read, err := options.NewRead(ctx, options.WithID(req.GetId()), options.WithDerivedID("agent", req.GetAgentId()))
	if err != nil {
		return nil, err
	}

	if err := a.service.DeleteAgentAbsence(ctx, read); err != nil {
		return nil, err
	}

	return &pb.DeleteAgentAbsenceResponse{Id: req.Id}, nil
}

func (a *AgentAbsence) CreateAgentsAbsences(ctx context.Context, req *pb.CreateAgentsAbsencesRequest) (*pb.CreateAgentsAbsencesResponse, error) {
	search, err := options.NewSearch(ctx)
	if err != nil {
		return nil, err
	}

	out, err := a.service.CreateAgentsAbsences(ctx, search, unmarshalAgentsAbsencesBulk(req.GetAgentIds(), req.GetItems()))
	if err != nil {
		return nil, err
	}

	return &pb.CreateAgentsAbsencesResponse{Items: marshalAgentsAbsences(out)}, nil
}

func (a *AgentAbsence) SearchAgentsAbsences(ctx context.Context, req *pb.SearchAgentsAbsencesRequest) (*pb.SearchAgentsAbsencesResponse, error) {
	opts := []options.Option{
		options.WithPagination(req.GetPage(), req.GetSize()),
		options.WithSearch(req.GetQ()),
		options.WithFields(req.GetFields()),
		options.WithOrder(req.GetSort()),
		// TODO: Add filters.
	}

	search, err := options.NewSearch(ctx, opts...)
	if err != nil {
		return nil, err
	}

	out, next, err := a.service.SearchAgentsAbsences(ctx, search)
	if err != nil {
		return nil, err
	}

	return &pb.SearchAgentsAbsencesResponse{Items: marshalAgentsAbsences(out), Next: next}, nil
}

func unmarshalAbsenceProto(in *pb.Absence) *model.Absence {
	return &model.Absence{
		DomainRecord: model.DomainRecord{
			Id: in.Id,
		},
		AbsentAt:    model.NewDate(in.AbsentAt),
		AbsenceType: model.AgentAbsenceType(in.TypeId),
	}
}

func unmarshalAgentsAbsencesBulk(agentIDs []int64, in []*pb.CreateAgentsAbsencesRequestAbsentType) []*model.AgentAbsences {
	absences := make([]*model.Absence, 0, len(in))
	for _, absence := range in {
		start := model.NewDate(absence.DateFrom)
		end := model.NewDate(absence.DateTo)
		for d := start; !d.Time.After(end.Time); d.Time = d.Time.AddDate(0, 0, 1) {
			item := &model.Absence{
				AbsentAt:    d,
				AbsenceType: model.AgentAbsenceType(absence.TypeId),
			}

			absences = append(absences, item)
		}
	}

	out := make([]*model.AgentAbsences, 0, len(agentIDs))
	for _, id := range agentIDs {
		out = append(out, &model.AgentAbsences{
			Agent: model.LookupItem{
				Id: id,
			},
			Absence: absences,
		})
	}

	return out
}

func marshalAgentsAbsences(in []*model.AgentAbsences) []*pb.AgentAbsences {
	out := make([]*pb.AgentAbsences, 0, len(in))
	for _, it := range in {
		out = append(out, it.MarshalProto())
	}

	return out
}

func marshalAbsenceBulkProto(in []*model.Absence) []*pb.Absence {
	out := make([]*pb.Absence, 0, len(in))
	for _, t := range in {
		out = append(out, t.MarshalProto())
	}

	return out
}
