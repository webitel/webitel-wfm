package handler

import (
	"context"

	pb "github.com/webitel/webitel-wfm/gen/go/api/wfm"
	"github.com/webitel/webitel-wfm/infra/server/grpccontext"
	"github.com/webitel/webitel-wfm/internal/model"
)

type AgentWorkingConditionsManager interface {
	ReadAgentWorkingConditions(ctx context.Context, user *model.SignedInUser, agentId int64) (*model.AgentWorkingConditions, error)
	UpdateAgentWorkingConditions(ctx context.Context, user *model.SignedInUser, agentId int64, in *model.AgentWorkingConditions) error
}

type AgentWorkingConditions struct {
	pb.UnimplementedAgentWorkingConditionsServiceServer

	svc AgentWorkingConditionsManager
}

func NewAgentWorkingConditions(svc AgentWorkingConditionsManager) *AgentWorkingConditions {
	return &AgentWorkingConditions{
		svc: svc,
	}
}

func (a *AgentWorkingConditions) ReadAgentWorkingConditions(ctx context.Context, req *pb.ReadAgentWorkingConditionsRequest) (*pb.ReadAgentWorkingConditionsResponse, error) {
	s := grpccontext.FromContext(ctx)
	out, err := a.svc.ReadAgentWorkingConditions(ctx, s.SignedInUser, req.GetAgentId())
	if err != nil {
		return nil, err
	}

	return &pb.ReadAgentWorkingConditionsResponse{Item: out.MarshalProto()}, nil
}

func (a *AgentWorkingConditions) UpdateAgentWorkingConditions(ctx context.Context, req *pb.UpdateAgentWorkingConditionsRequest) (*pb.UpdateAgentWorkingConditionsResponse, error) {
	s := grpccontext.FromContext(ctx)
	if err := a.svc.UpdateAgentWorkingConditions(ctx, s.SignedInUser, req.GetAgentId(), unmarshalAgentWorkingConditionsProto(req.GetItem())); err != nil {
		return nil, err
	}

	out, err := a.svc.ReadAgentWorkingConditions(ctx, s.SignedInUser, req.GetAgentId())
	if err != nil {
		return nil, err
	}

	return &pb.UpdateAgentWorkingConditionsResponse{Item: out.MarshalProto()}, nil
}

func unmarshalAgentWorkingConditionsProto(item *pb.AgentWorkingConditions) *model.AgentWorkingConditions {
	out := &model.AgentWorkingConditions{
		WorkingCondition: model.LookupItem{
			Id: item.WorkingCondition.Id,
		},
		PauseTemplate: &model.LookupItem{},
	}

	if item.PauseTemplate != nil {
		out.PauseTemplate = &model.LookupItem{
			Id: item.PauseTemplate.Id,
		}
	}

	return out
}
