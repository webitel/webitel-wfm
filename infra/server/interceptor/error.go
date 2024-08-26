package interceptor

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/webitel/webitel-wfm/pkg/werror"
)

func ErrUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		h, err := handler(ctx, req)
		if err != nil {
			var rpc *werror.RPCError

			switch v := err.(type) {
			case werror.AuthForbiddenError:
				rpc = werror.NewRPCError(v.Id(), codes.PermissionDenied, v.RPCError())
			case werror.AuthInvalidSessionError, werror.AuthInvalidTokenError:
				if e, ok := v.(interface {
					Id() string
					RPCError() string
				}); ok {
					rpc = werror.NewRPCError(e.Id(), codes.Unauthenticated, e.RPCError())
				}
			case werror.DBEntityConflictError, werror.DBCheckViolationError, werror.DBNotNullViolationError,
				werror.DBUniqueViolationError, werror.DBForeignKeyViolationError:
				if e, ok := v.(interface {
					Id() string
					RPCError() string
				}); ok {
					rpc = werror.NewRPCError(e.Id(), codes.Aborted, e.RPCError())
				}
			case werror.DBNoRowsError:
				rpc = werror.NewRPCError(v.Id(), codes.NotFound, v.RPCError())
			case werror.ValidationError:
				rpc = werror.NewRPCError(v.Id(), codes.InvalidArgument, v.RPCError())
			case werror.DBInternalError:
				rpc = werror.NewRPCError(v.Id(), codes.Internal, v.RPCError())
			}

			if rpc == nil {
				rpc = werror.NewRPCError("server.interceptor.error", codes.Internal, err.Error())
			}

			return h, status.Error(rpc.RPCCode, rpc.JSON())
		}

		return h, err
	}
}
