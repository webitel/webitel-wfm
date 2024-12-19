package handler

import (
	"context"

	"google.golang.org/grpc"

	pb "github.com/webitel/webitel-wfm/gen/go/api/wfm"
	"github.com/webitel/webitel-wfm/infra/server/grpccontext"
	"github.com/webitel/webitel-wfm/internal/model"
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

func (a *AgentAbsence) CreateAgentAbsence(ctx context.Context, in *pb.CreateAgentAbsenceRequest) (*pb.CreateAgentAbsenceResponse, error) {
	s := grpccontext.FromContext(ctx)
	out, err := a.service.CreateAgentAbsence(ctx, s.SignedInUser, unmarshalAgentAbsenceProto(in.Item))
	if err != nil {
		return nil, err
	}

	return &pb.CreateAgentAbsenceResponse{Item: out.MarshalProto()}, nil
}

func (a *AgentAbsence) UpdateAgentAbsence(ctx context.Context, in *pb.UpdateAgentAbsenceRequest) (*pb.UpdateAgentAbsenceResponse, error) {
	s := grpccontext.FromContext(ctx)
	out, err := a.service.UpdateAgentAbsence(ctx, s.SignedInUser, unmarshalAgentAbsenceProto(in.Item))
	if err != nil {
		return nil, err
	}

	return &pb.UpdateAgentAbsenceResponse{Item: out.MarshalProto()}, nil
}

func (a *AgentAbsence) DeleteAgentAbsence(ctx context.Context, in *pb.DeleteAgentAbsenceRequest) (*pb.DeleteAgentAbsenceResponse, error) {
	s := grpccontext.FromContext(ctx)
	if err := a.service.DeleteAgentAbsence(ctx, s.SignedInUser, in.AgentId, in.Id); err != nil {
		return nil, err
	}

	return &pb.DeleteAgentAbsenceResponse{Id: in.Id}, nil
}

func (a *AgentAbsence) ReadAgentAbsences(ctx context.Context, in *pb.ReadAgentAbsencesRequest) (*pb.ReadAgentAbsencesResponse, error) {
	s := grpccontext.FromContext(ctx)
	search := &model.AgentAbsenceSearch{
		AgentIds:     []int64{in.AgentId},
		AbsentAtFrom: model.NewTimestamp(in.AbsentAtFrom),
		AbsentAtTo:   model.NewTimestamp(in.AbsentAtTo),
	}

	out, err := a.service.ReadAgentAbsences(ctx, s.SignedInUser, search)
	if err != nil {
		return nil, err
	}

	return &pb.ReadAgentAbsencesResponse{Item: out.MarshalProto()}, nil
}

func (a *AgentAbsence) CreateAgentsAbsencesBulk(ctx context.Context, in *pb.CreateAgentsAbsencesBulkRequest) (*pb.CreateAgentsAbsencesBulkResponse, error) {
	s := grpccontext.FromContext(ctx)
	out, err := a.service.CreateAgentsAbsencesBulk(ctx, s.SignedInUser, in.AgentIds, unmarshalAgentsAbsencesBulk(in.Items))
	if err != nil {
		return nil, err
	}

	return &pb.CreateAgentsAbsencesBulkResponse{Items: marshalAgentsAbsences(out)}, nil
}

func (a *AgentAbsence) SearchAgentsAbsences(ctx context.Context, in *pb.SearchAgentsAbsencesRequest) (*pb.SearchAgentsAbsencesResponse, error) {
	s := grpccontext.FromContext(ctx)
	search := &model.AgentAbsenceSearch{
		SearchItem: model.SearchItem{
			Search: in.Q,
			Page:   in.GetPage(),
			Size:   in.GetSize(),
			Sort:   in.Sort,
			Fields: in.Fields,
		},
		AbsentAtFrom:  model.NewTimestamp(in.AbsentAtFrom),
		AbsentAtTo:    model.NewTimestamp(in.AbsentAtTo),
		SupervisorIds: in.SupervisorId,
		TeamIds:       in.TeamId,
		SkillIds:      in.SkillId,
	}

	out, next, err := a.service.SearchAgentsAbsences(ctx, s.SignedInUser, search)
	if err != nil {
		return nil, err
	}

	return &pb.SearchAgentsAbsencesResponse{Items: marshalAgentsAbsences(out), Next: next}, nil
}

func unmarshalAgentAbsenceProto(in *pb.AgentAbsence) *model.AgentAbsence {
	return &model.AgentAbsence{
		Agent: model.LookupItem{
			Id: in.Agent.Id,
		},
		Absence: model.Absence{
			DomainRecord: model.DomainRecord{
				Id: in.Absence.Id,
			},
			AbsentAt:    model.NewDate(in.Absence.AbsentAt),
			AbsenceType: model.AgentAbsenceType(in.Absence.TypeId),
		},
	}
}

func unmarshalAgentsAbsencesBulk(in []*pb.CreateAgentsAbsencesBulkRequestAbsentType) []*model.AgentAbsenceBulk {
	out := make([]*model.AgentAbsenceBulk, 0, len(in))
	for _, it := range in {
		item := &model.AgentAbsenceBulk{
			AbsenceType:  model.AgentAbsenceType(it.TypeId),
			AbsentAtFrom: it.AbsentAtFrom,
			AbsentAtTo:   it.AbsentAtTo,
		}

		out = append(out, item)
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
