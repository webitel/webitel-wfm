package interceptor

import (
	"context"
	"errors"
	"strconv"
	"strings"

	validatepb "buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
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
						fields := make([]string, 0, len(ve.Violations))
						for _, f := range violation.Proto.GetField().GetElements() {
							field := *f.FieldName
							if f.Subscript != nil {
								var subscript string
								switch s := f.Subscript.(type) {
								case *validatepb.FieldPathElement_Index:
									subscript = strconv.FormatUint(s.Index, 10)
								case *validatepb.FieldPathElement_BoolKey:
									subscript = strconv.FormatBool(s.BoolKey)
								case *validatepb.FieldPathElement_IntKey:
									subscript = strconv.FormatInt(s.IntKey, 10)
								case *validatepb.FieldPathElement_UintKey:
									subscript = strconv.FormatUint(s.UintKey, 10)
								case *validatepb.FieldPathElement_StringKey:
									subscript = s.StringKey
								}

								field = field + "[" + subscript + "]"
							}

							fields = append(fields, field)
						}

						wrappers = append(wrappers, werror.WithValue(strings.Join(fields, ".")+"["+violation.Proto.GetConstraintId()+"]",
							violation.Proto.GetMessage()),
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
