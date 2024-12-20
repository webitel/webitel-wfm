package interceptor

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"github.com/webitel/webitel-wfm/pkg/werror"
)

func TestErrUnaryServerInterceptor(t *testing.T) {
	interceptor := ErrUnaryServerInterceptor()
	info := &grpc.UnaryServerInfo{
		FullMethod: "/FakeService/FakeMethod",
	}

	t.Run("not nil werror received", func(t *testing.T) {
		_, err := interceptor(context.Background(), nil, info, func(context.Context, any) (any, error) {
			return nil, werror.New("testing", werror.WithID("server.interceptor.error.testing"), werror.WithCode(codes.InvalidArgument))
		})

		require.Error(t, err)
		assert.EqualError(t, err, `rpc error: code = Code(400) desc = {"id":"server.interceptor.error.testing","detail":"testing","status":"Bad Request"}`)
	})

	t.Run("not nil err received", func(t *testing.T) {
		_, err := interceptor(context.Background(), nil, info, func(context.Context, any) (any, error) {
			return nil, fmt.Errorf("testing")
		})

		require.Error(t, err)
		assert.EqualError(t, err, `rpc error: code = Code(500) desc = {"id":"","detail":"testing","status":"Internal Server Error"}`)
	})

	t.Run("nil err received", func(t *testing.T) {
		_, err := interceptor(context.Background(), nil, info, func(context.Context, any) (any, error) {
			return nil, nil
		})

		require.NoError(t, err)
	})
}
