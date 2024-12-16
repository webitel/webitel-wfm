package webitel

import (
	"context"
	"fmt"
	"strings"

	"github.com/webitel/engine/discovery"
	otelgrpc "github.com/webitel/webitel-go-kit/tracing/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/webitel/webitel-wfm/infra/server/grpccontext"
	"github.com/webitel/webitel-wfm/pkg/werror"
)

var (
	ErrInternal = werror.Internal("internal server error", werror.WithID("webitel.connection.service"))
	ErrNoRows   = werror.NotFound("no rows in result set", werror.WithID("webitel.connection.service"))
)

var (
	// see https://github.com/grpc/grpc/blob/master/doc/service_config.md to know more about service config
	retryPolicy = `{
		"loadBalancingConfig": [ { "round_robin": {} } ],
		"methodConfig": [
			{
         		"timeout": "5.000000001s",
   				"waitForReady": true,
   				"retryPolicy": {
    				"MaxAttempts": 4,
    				"InitialBackoff": ".01s",
    				"MaxBackoff": ".01s",
    				"BackoffMultiplier": 1.0,
    				"RetryableStatusCodes": [ "UNAVAILABLE" ]
   				}
 			}
		]
	}`
)

type Connection struct {
	svc *discovery.ServiceConnection
	cli *grpc.ClientConn
}

func NewConnection(svc *discovery.ServiceConnection) (*Connection, error) {
	target := fmt.Sprintf("%s:%d", svc.Host, svc.Port)
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(retryPolicy),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
		grpc.WithChainUnaryInterceptor(authUnaryInterceptor()),
	}

	// Set up a connection to the server with service config and create the channel.
	//
	// TODO: the recommended approach is to fetch the retry configuration from the name resolver
	//		 (which is part of the service config) rather than defining it on the client side.
	cli, err := grpc.NewClient(target, opts...)
	if err != nil {
		return nil, err
	}

	return &Connection{svc: svc, cli: cli}, nil
}

func (c *Connection) Ready() bool {
	switch c.cli.GetState() {
	case connectivity.Idle, connectivity.Ready:
		return true
	default:
		return false
	}
}

func (c *Connection) Name() string {
	return c.svc.Id
}

func (c *Connection) Close() error {
	if err := c.cli.Close(); err != nil {
		return err
	}

	return nil
}

func (c *Connection) Client() *grpc.ClientConn {
	return c.cli
}

func ParseError(err error) error {
	st, ok := status.FromError(err)
	if ok {
		switch st.Code() {
		case codes.NotFound, codes.PermissionDenied:
			return werror.Wrap(ErrNoRows, werror.WithCause(st.Err()))
		default:
			if strings.Contains(st.Message(), "no rows in result set") {
				return werror.Wrap(ErrNoRows, werror.WithCause(st.Err()))
			}

			return werror.Wrap(ErrInternal, werror.WithCause(st.Err()))
		}
	}

	return werror.Wrap(ErrInternal, werror.WithCause(err))
}

func authUnaryInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		s := grpccontext.FromContext(ctx)
		md := metadata.New(map[string]string{
			"X-Webitel-Access": s.SignedInUser.Token,
		})

		return invoker(metadata.NewOutgoingContext(ctx, md), method, req, reply, cc, opts...)
	}
}
