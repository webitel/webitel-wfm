package webitel

import (
	"context"
	"strings"
	"time"

	"github.com/webitel/webitel-go-kit/logging/wlog"
	otelgrpc "github.com/webitel/webitel-go-kit/tracing/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/webitel/webitel-wfm/infra/registry"
	"github.com/webitel/webitel-wfm/infra/registry/resolver"
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
		"loadBalancingConfig": [ { "selector": {} } ],
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

func New(log *wlog.Logger, discovery registry.Discovery, target string) (*grpc.ClientConn, error) {
	opts := []grpc.DialOption{
		grpc.WithResolvers(resolver.NewBuilder(log, discovery, resolver.WithInsecure(true))),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(retryPolicy),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
		grpc.WithChainUnaryInterceptor(timeoutUnaryInterceptor(10*time.Second), authUnaryInterceptor()),
	}

	// Set up a connection to the server with service config and create the channel.
	//
	// TODO: the recommended approach is to fetch the retry configuration from the name resolver
	//		 (which is part of the service config) rather than defining it on the client side.
	cli, err := grpc.NewClient("discovery:///"+target, opts...)
	if err != nil {
		return nil, err
	}

	return cli, nil
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

func timeoutUnaryInterceptor(timeout time.Duration) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		if timeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, timeout)
			defer cancel()
		}

		var p registry.Peer

		return invoker(registry.NewPeerContext(ctx, &p), method, req, reply, cc, opts...)
	}
}
