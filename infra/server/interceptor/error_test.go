package interceptor

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/webitel/webitel-wfm/pkg/werror"
)

func TestErrUnaryServerInterceptor(t *testing.T) {
	interceptor := ErrUnaryServerInterceptor()
	info := &grpc.UnaryServerInfo{
		FullMethod: "/FakeService/FakeMethod",
	}

	t.Run("not nil AppError received", func(t *testing.T) {
		_, err := interceptor(context.Background(), nil, info, func(context.Context, any) (any, error) {
			return nil, werror.NewBadRequestError("testing").SetId("grpc.interceptor.testing")
		})

		require.Error(t, err)
		assert.EqualError(t, err, `rpc error: code = InvalidArgument desc = {"id":"grpc.interceptor.testing","detail":"testing","request_id":""}`)
	})

	t.Run("not nil err received", func(t *testing.T) {
		_, err := interceptor(context.Background(), nil, info, func(context.Context, any) (any, error) {
			return nil, fmt.Errorf("testing")
		})

		require.Error(t, err)
		assert.EqualError(t, err, "testing")
	})

	t.Run("nil err received", func(t *testing.T) {
		_, err := interceptor(context.Background(), nil, info, func(context.Context, any) (any, error) {
			return nil, nil
		})

		require.NoError(t, err)
	})
}
