package interceptor

import (
	"context"
	"errors"
	"fmt"

	"github.com/bufbuild/protovalidate-go"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"

	"github.com/webitel/webitel-wfm/pkg/werror"
)

func ValidateUnaryServerInterceptor(val *protovalidate.Validator) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if v, ok := req.(proto.Message); ok {
			if err := val.Validate(v); err != nil {
				var ve *protovalidate.ValidationError
				if ok := errors.As(err, &ve); ok {
					return nil, werror.NewValidationError("interceptor.validate", fmt.Sprintf("%T", v),
						ve.Violations[0].GetFieldPath(), ve.Violations[0].GetConstraintId(), ve.Violations[0].GetMessage())
				}

				return nil, err
			}
		}

		return handler(ctx, req)
	}
}
