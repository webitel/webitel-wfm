package engine

import (
	"context"

	gogrpc "buf.build/gen/go/webitel/engine/grpc/go/_gogrpc"
	pb "buf.build/gen/go/webitel/engine/protocolbuffers/go"
	"github.com/webitel/webitel-go-kit/logging/wlog"

	"github.com/webitel/webitel-wfm/infra/webitel"
)

type TeamService struct {
	log *wlog.Logger
	cli gogrpc.AgentTeamServiceClient
}

func newTeamServiceClient(cli *Client) *TeamService {
	return &TeamService{
		log: cli.log,
		cli: gogrpc.NewAgentTeamServiceClient(cli.conn),
	}
}

func (t *TeamService) Team(ctx context.Context, id int64) (*pb.AgentTeam, error) {
	team, err := t.cli.ReadAgentTeam(ctx, &pb.ReadAgentTeamRequest{Id: id})
	if err != nil {
		return nil, webitel.ParseError(err)
	}

	return team, nil
}
