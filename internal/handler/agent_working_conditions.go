package handler

import (
	"context"

	"google.golang.org/grpc"

	pb "github.com/webitel/webitel-wfm/gen/go/api/wfm"
	"github.com/webitel/webitel-wfm/internal/model"
	"github.com/webitel/webitel-wfm/internal/model/options"
	"github.com/webitel/webitel-wfm/internal/service"
)

type AgentWorkingConditions struct {
	pb.UnimplementedAgentWorkingConditionsServiceServer

	service service.AgentWorkingConditionsManager
}

func NewAgentWorkingConditions(sr grpc.ServiceRegistrar, service service.AgentWorkingConditionsManager) *AgentWorkingConditions {
	s := &AgentWorkingConditions{
		service: service,
	}

	pb.RegisterAgentWorkingConditionsServiceServer(sr, s)

	return s
}

func (a *AgentWorkingConditions) ReadAgentWorkingConditions(ctx context.Context, req *pb.ReadAgentWorkingConditionsRequest) (*pb.ReadAgentWorkingConditionsResponse, error) {
	read, err := options.NewRead(ctx, options.WithID(req.GetAgentId()))
	if err != nil {
		return nil, err
	}

	out, err := a.service.ReadAgentWorkingConditions(ctx, read)
	if err != nil {
		return nil, err
	}

	return &pb.ReadAgentWorkingConditionsResponse{Item: out.MarshalProto()}, nil
}

func (a *AgentWorkingConditions) UpdateAgentWorkingConditions(ctx context.Context, req *pb.UpdateAgentWorkingConditionsRequest) (*pb.UpdateAgentWorkingConditionsResponse, error) {
	read, err := options.NewRead(ctx, options.WithID(req.GetAgentId()))
	if err != nil {
		return nil, err
	}

	out, err := a.service.UpdateAgentWorkingConditions(ctx, read, unmarshalAgentWorkingConditionsProto(req.GetItem()))
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
