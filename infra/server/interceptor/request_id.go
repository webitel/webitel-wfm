package interceptor

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/webitel/webitel-wfm/infra/server/grpccontext"
)

// XRequestIDKey is metadata key name for request ID.
var XRequestIDKey = "x-request-id"

// RequestIdUnaryServerInterceptor returns a server interceptor function to set request ID
// to context and logger.
//   - If request ID is already set in metadata, it will be used instead of generating a new one.
//   - If request ID is not set in metadata, a new one will be generated.
//   - If request ID cannot be generated, an empty string will be used.
func RequestIdUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		return handler(grpccontext.SetRequestId(ctx, handleRequestID(ctx)), req)
	}
}

func handleRequestID(ctx context.Context) string {
	newUUID, err := uuid.NewUUID()
	if err != nil {
		return ""
	}

	id := newUUID.String()

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return id
	}

	header, ok := md[XRequestIDKey]
	if !ok || len(header) == 0 {
		return id
	}

	requestID := header[0]
	if requestID == "" {
		return id
	}

	return requestID
}
