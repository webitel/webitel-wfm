package interceptor

import (
	"context"
	"strings"
	"time"

	"github.com/webitel/webitel-go-kit/logging/wlog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/webitel/webitel-wfm/infra/server/grpccontext"
)

// LoggingUnaryServerInterceptor returns a new unary server interceptor for logging requests.
func LoggingUnaryServerInterceptor(log *wlog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()

		ip := "<not found>"
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			ip = getClientIp(md)
		}

		h, err := handler(ctx, req)

		// var f wlog.Field
		s := grpccontext.FromContext(ctx)
		if err != nil {
			f := wlog.Err(err)
			// var appError apperrors.AppError

			// switch {
			// case errors.As(err, &appError):
			// 	// f = wlog.ObjectErr(appError.SetRequestId(s.RequestId))
			// default:
			// 	f = wlog.Err(err)
			// }

			log.Error("processed request", f, wlog.String("client_ip", ip),
				wlog.Any("method", info.FullMethod), wlog.Any("duration", time.Since(start)))
		} else {
			log.Debug("processed request", wlog.String("client_ip", ip),
				wlog.Any("method", info.FullMethod), wlog.Any("duration", time.Since(start)), wlog.String("request_id", s.RequestId))
		}

		return h, err
	}
}

func getClientIp(info metadata.MD) string {
	ip := strings.Join(info.Get("x-real-ip"), ",")
	if ip == "" {
		ip = strings.Join(info.Get("x-forwarded-for"), ",")
	}

	return ip
}
