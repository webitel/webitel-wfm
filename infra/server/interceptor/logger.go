package interceptor

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// LoggingUnaryServerInterceptor returns a new unary server interceptor for logging requests.
func LoggingUnaryServerInterceptor(log *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()
		ip := "<not found>"
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			ip = getClientIp(md)
		}

		h, err := handler(ctx, req)

		// TODO: Client errors (that do not appear as an application logic error)
		// 	should be logged as DEBUG or INFO level.
		// Context-aware, so the record carries the request's trace ids.
		log.DebugContext(ctx, "processed request", slog.Any("error", err),
			slog.String("client_ip", ip),
			slog.Any("method", info.FullMethod),
			slog.String("duration", time.Since(start).String()),
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
