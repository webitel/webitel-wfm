package logger

import (
	"context"

	gogrpc "buf.build/gen/go/webitel/logger/grpc/go/_gogrpc"
	pb "buf.build/gen/go/webitel/logger/protocolbuffers/go"
	"github.com/webitel/webitel-go-kit/logging/wlog"

	"github.com/webitel/webitel-wfm/infra/webitel"
)

type ConfigService struct {
	log *wlog.Logger
	cli gogrpc.ConfigServiceClient
}

func NewConfigServiceClient(log *wlog.Logger, conn *webitel.ConnectionManager[*webitel.Connection]) (*ConfigService, error) {
	cli, err := conn.Connection()
	if err != nil {
		return nil, err
	}

	return &ConfigService{log: log, cli: gogrpc.NewConfigServiceClient(cli.Client())}, nil
}

func (c *ConfigService) Active(ctx context.Context, domainId int64, object string) (bool, error) {
	req := &pb.CheckConfigStatusRequest{
		ObjectName: object,
		DomainId:   domainId,
	}

	out, err := c.cli.CheckConfigStatus(ctx, req)
	if err != nil {
		return false, err
	}

	return out.IsEnabled, nil
}
