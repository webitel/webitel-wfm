package logger

import (
	"context"

	gogrpc "buf.build/gen/go/webitel/logger/grpc/go/_gogrpc"
	pb "buf.build/gen/go/webitel/logger/protocolbuffers/go"
	"github.com/webitel/webitel-go-kit/logging/wlog"
	"google.golang.org/grpc"
)

type ConfigService struct {
	log *wlog.Logger
	cli gogrpc.ConfigServiceClient
}

func newConfigServiceClient(log *wlog.Logger, conn *grpc.ClientConn) *ConfigService {
	return &ConfigService{
		log: log,
		cli: gogrpc.NewConfigServiceClient(conn),
	}
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
