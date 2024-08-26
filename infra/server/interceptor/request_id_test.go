package interceptor

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/webitel/webitel-wfm/infra/server/grpccontext"
)

func TestRequestIdUnaryServerInterceptor(t *testing.T) {
	interceptor := RequestIdUnaryServerInterceptor()
	info := &grpc.UnaryServerInfo{
		FullMethod: "/FakeService/FakeMethod",
	}

	t.Run("valid request id", func(t *testing.T) {
		id := uuid.NewString()
		ctx := context.Background()
		md := metadata.New(map[string]string{
			XRequestIDKey: id,
		})

		ctx = metadata.NewIncomingContext(ctx, md)

		handler := func(ctx context.Context, req any) (any, error) {
			g := grpccontext.FromContext(ctx)
			assert.Equal(t, id, g.RequestId)

			return "good", nil
		}

		_, _ = interceptor(ctx, nil, info, handler)
	})

	t.Run("empty incoming request id", func(t *testing.T) {
		ctx := context.Background()
		handler := func(ctx context.Context, req any) (any, error) {
			g := grpccontext.FromContext(ctx)
			assert.NotEmpty(t, g.RequestId)

			return "good", nil
		}

		_, _ = interceptor(ctx, nil, info, handler)
	})
}
