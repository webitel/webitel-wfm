package interceptor

import (
	"context"
	"strings"
	"time"

	"github.com/webitel/webitel-go-kit/logging/wlog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
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

		// TODO: Client errors (that do not appear as an application logic error)
		// 	should be logged as DEBUG or INFO level.
		log.Debug("processed request", wlog.Err(err),
			wlog.String("client_ip", ip),
			wlog.Any("method", info.FullMethod),
			wlog.String("duration", time.Since(start).String()),
		)

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
