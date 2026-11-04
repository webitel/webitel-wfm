package engine

import (
	"context"
	"strconv"

	gogrpc "buf.build/gen/go/webitel/engine/grpc/go/_gogrpc"
	pb "buf.build/gen/go/webitel/engine/protocolbuffers/go"

	"github.com/webitel/webitel-wfm/infra/webitel"
	"github.com/webitel/webitel-wfm/internal/model"
)

type AgentService struct {
	cli gogrpc.AgentServiceClient
}

func newAgentServiceClient(cli *Client) *AgentService {
	return &AgentService{cli: gogrpc.NewAgentServiceClient(cli.conn)}
}

func (a *AgentService) Agent(ctx context.Context, id int64) (int64, error) {
	agent, err := a.cli.ReadAgent(ctx, &pb.ReadAgentRequest{Id: id})
	if err != nil {
		return 0, webitel.ParseError(err)
	}

	return agent.GetId(), nil
}

func (a *AgentService) Agents(ctx context.Context, search *model.AgentSearch) ([]int64, error) {
	req := &pb.SearchAgentRequest{
		Size:   -1,
		Fields: []string{"id"},
	}

	if len(search.IDs) > 0 {
		ids := make([]string, 0, len(search.IDs))
		for _, id := range search.IDs {
			ids = append(ids, strconv.FormatInt(id, 10))
		}

		req.Id = ids
	}

	if len(search.SupervisorIDs) > 0 {
		ids := make([]uint32, 0, len(search.SupervisorIDs))
		for _, id := range search.SupervisorIDs {
			ids = append(ids, uint32(id))
		}

		req.SupervisorId = ids
	}

	if len(search.TeamIDs) > 0 {
		ids := make([]uint32, 0, len(search.TeamIDs))
		for _, id := range search.TeamIDs {
			ids = append(ids, uint32(id))
		}

		req.TeamId = ids
	}

	if len(search.SkillIDs) > 0 {
		ids := make([]uint32, 0, len(search.SkillIDs))
		for _, id := range search.SkillIDs {
			ids = append(ids, uint32(id))
		}

		req.SkillId = ids
	}

	agents, err := a.cli.SearchAgent(ctx, req)
	if err != nil {
		return nil, webitel.ParseError(err)
	}

	ids := make([]int64, 0, len(agents.GetItems()))
	for _, id := range agents.GetItems() {
		ids = append(ids, id.GetId())
	}

	return ids, nil
}
