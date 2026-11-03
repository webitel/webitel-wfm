package interceptor

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/webitel/webitel-go-kit/pkg/errors"

	"github.com/webitel/webitel-wfm/internal/model"
)

func TestErrUnaryServerInterceptor(t *testing.T) {
	interceptor := ErrUnaryServerInterceptor()
	info := &grpc.UnaryServerInfo{
		FullMethod: "/FakeService/FakeMethod",
	}

	t.Run("not nil werror received", func(t *testing.T) {
		_, err := interceptor(context.Background(), nil, info, func(context.Context, any) (any, error) {
			return nil, errors.New("testing", errors.WithID("server.interceptor.error.testing"), errors.WithCode(codes.InvalidArgument))
		})

		require.Error(t, err)
		assert.EqualError(t, err, `rpc error: code = Code(400) desc = {"id":"server.interceptor.error.testing","detail":"testing","code":400,"status":"Bad Request"}`)
	})

	t.Run("business refusal", func(t *testing.T) {
		v := model.Violation{OperatorID: 7, Date: pgtype.Date{Time: time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC), Valid: true}, Reason: "service.shift.overlap"}
		_, err := interceptor(context.Background(), nil, info, func(context.Context, any) (any, error) {
			return nil, model.Refuse(model.NewRefusal("server.interceptor.error.refused", "refused"), v)
		})

		st, ok := status.FromError(err)
		require.True(t, ok)

		var got rpcError
		require.NoError(t, json.Unmarshal([]byte(st.Message()), &got))
		assert.Equal(t, "server.interceptor.error.refused", got.ID)
		assert.Equal(t, int32(400), got.Code)
		assert.Equal(t, "Bad Request", got.Status)
		assert.Contains(t, got.Detail, "violations = operator 7 on 2026-11-02: service.shift.overlap")
	})

	t.Run("not nil err received", func(t *testing.T) {
		_, err := interceptor(context.Background(), nil, info, func(context.Context, any) (any, error) {
			return nil, fmt.Errorf("testing")
		})

		require.Error(t, err)
		assert.EqualError(t, err, `rpc error: code = Code(500) desc = {"id":"","detail":"testing","code":500,"status":"Internal Server Error"}`)
	})

	t.Run("nil err received", func(t *testing.T) {
		_, err := interceptor(context.Background(), nil, info, func(context.Context, any) (any, error) {
			return nil, nil //nolint:nilnil // a handler that succeeds with an empty response
		})

		require.NoError(t, err)
	})
}
