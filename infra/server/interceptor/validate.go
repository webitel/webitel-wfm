package interceptor

import (
	"context"
	"errors"

	"github.com/bufbuild/protovalidate-go"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"

	"github.com/webitel/webitel-wfm/pkg/werror"
)

var ErrValidation = werror.InvalidArgument("validate input message", werror.WithID("interceptor.validate"))

func ValidateUnaryServerInterceptor(val *protovalidate.Validator) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if v, ok := req.(proto.Message); ok {
			if err := val.Validate(v); err != nil {
				var ve *protovalidate.ValidationError
				if ok := errors.As(err, &ve); ok {
					wrappers := make([]werror.Wrapper, 0)
					for _, violation := range ve.Violations {
						wrappers = append(wrappers, werror.WithValue(violation.GetFieldPath()+"["+violation.GetConstraintId()+"]",
							violation.GetMessage()),
						)
					}

					return nil, werror.Wrap(ErrValidation, wrappers...)
				}

				return nil, werror.Wrap(err, werror.WithCause(err))
			}
		}

		return handler(ctx, req)
	}
}
