package logger

import (
	"context"

	"google.golang.org/grpc"

	gogrpc "buf.build/gen/go/webitel/logger/grpc/go/_gogrpc"
	pb "buf.build/gen/go/webitel/logger/protocolbuffers/go"
)

type ConfigService struct {
	cli gogrpc.ConfigServiceClient
}

func newConfigServiceClient(conn *grpc.ClientConn) *ConfigService {
	return &ConfigService{cli: gogrpc.NewConfigServiceClient(conn)}
}

func (c *ConfigService) Active(ctx context.Context, domainID int64, object string) (bool, error) {
	req := &pb.CheckConfigStatusRequest{
		ObjectName: object,
		DomainId:   domainID,
	}

	out, err := c.cli.CheckConfigStatus(ctx, req)
	if err != nil {
		return false, err
	}

	return out.GetIsEnabled(), nil
}
