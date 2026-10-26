package webitel

import (
	"context"
	"strings"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/webitel/webitel-go-kit/infra/discovery"
	resolver "github.com/webitel/webitel-go-kit/infra/transport/gRPC/resolver/discovery"
	"github.com/webitel/webitel-go-kit/pkg/errors"

	"github.com/webitel/webitel-wfm/infra/server/grpccontext"
)

var (
	ErrInternal = errors.Internal("internal server error", errors.WithID("webitel.connection.service"))
	ErrNoRows   = errors.NotFound("no rows in result set", errors.WithID("webitel.connection.service"))
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

// New dials a Webitel service with retries, waiting for it to come up, and
// forwards the signed-in user's token with every call.
func New(dp discovery.Discovery, target string) (*grpc.ClientConn, error) {
	return Dial(dp, target,
		grpc.WithDefaultServiceConfig(retryPolicy),
		grpc.WithChainUnaryInterceptor(timeoutUnaryInterceptor(10*time.Second), authUnaryInterceptor()),
	)
}

// Dial connects to a service registered in discovery.
func Dial(dp discovery.Discovery, target string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	opts = append([]grpc.DialOption{
		grpc.WithResolvers(resolver.NewBuilder(dp, resolver.WithInsecure(true))),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	}, opts...)

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
			return errors.Wrap(ErrNoRows, errors.WithCause(st.Err()))
		default:
			if strings.Contains(st.Message(), "no rows in result set") {
				return errors.Wrap(ErrNoRows, errors.WithCause(st.Err()))
			}

			return errors.Wrap(ErrInternal, errors.WithCause(st.Err()))
		}
	}

	return errors.Wrap(ErrInternal, errors.WithCause(err))
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

		return invoker(ctx, method, req, reply, cc, opts...)
	}
}
