package engine

import (
	"context"
	"strconv"

	gogrpc "buf.build/gen/go/webitel/engine/grpc/go/_gogrpc"
	pb "buf.build/gen/go/webitel/engine/protocolbuffers/go"
	"github.com/webitel/webitel-go-kit/logging/wlog"

	"github.com/webitel/webitel-wfm/infra/webitel"
	"github.com/webitel/webitel-wfm/internal/model"
)

type AgentService struct {
	log *wlog.Logger
	cli gogrpc.AgentServiceClient
}

func NewAgentServiceClient(log *wlog.Logger, conn *webitel.ConnectionManager[*webitel.Connection]) (*AgentService, error) {
	cli, err := conn.Connection()
	if err != nil {
		return nil, err
	}

	return &AgentService{log: log, cli: gogrpc.NewAgentServiceClient(cli.Client())}, nil
}

func (a *AgentService) Agent(ctx context.Context, id int64) (int64, error) {
	agent, err := a.cli.ReadAgent(ctx, &pb.ReadAgentRequest{Id: id})
	if err != nil {
		return 0, webitel.ParseError(err)
	}

	return agent.Id, nil
}

func (a *AgentService) Agents(ctx context.Context, search *model.AgentSearch) ([]int64, error) {
	req := &pb.SearchAgentRequest{
		Size:   -1,
		Fields: []string{"id"},
	}

	if len(search.Ids) > 0 {
		ids := make([]string, 0, len(search.Ids))
		for _, id := range search.Ids {
			ids = append(ids, strconv.FormatInt(id, 10))
		}

		req.Id = ids
	}

	if len(search.SupervisorIds) > 0 {
		ids := make([]uint32, 0, len(search.SupervisorIds))
		for _, id := range search.SupervisorIds {
			ids = append(ids, uint32(id))
		}

		req.SupervisorId = ids
	}

	if len(search.TeamIds) > 0 {
		ids := make([]uint32, 0, len(search.TeamIds))
		for _, id := range search.TeamIds {
			ids = append(ids, uint32(id))
		}

		req.TeamId = ids
	}

	if len(search.SkillIds) > 0 {
		ids := make([]uint32, 0, len(search.SkillIds))
		for _, id := range search.SkillIds {
			ids = append(ids, uint32(id))
		}

		req.SkillId = ids
	}

	agents, err := a.cli.SearchAgent(ctx, req)
	if err != nil {
		return nil, webitel.ParseError(err)
	}

	ids := make([]int64, 0, len(agents.Items))
	for _, id := range agents.Items {
		ids = append(ids, id.Id)
	}

	return ids, nil
}
