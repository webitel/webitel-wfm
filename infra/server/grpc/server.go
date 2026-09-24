package server

import (
	"context"
	"fmt"
	"log/slog"
	"net"

	"buf.build/go/protovalidate"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.uber.org/fx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	intrcp "github.com/webitel/webitel-go-kit/pkg/interceptors"

	"github.com/webitel/webitel-wfm/config"
	"github.com/webitel/webitel-wfm/infra/server/interceptor"
	"github.com/webitel/webitel-wfm/infra/webitel/auth"
)

var Module = fx.Module("grpc_server",
	fx.Provide(New),
	fx.Invoke(Run),
)

type Server struct {
	*grpc.Server
}

// New provides a new gRPC server.
func New(log *slog.Logger, authcli auth.Manager) (*Server, error) {
	val, err := protovalidate.New()
	if err != nil {
		return nil, fmt.Errorf("construct protovalidate rules: %w", err)
	}

	s := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler(otelgrpc.WithMessageEvents(otelgrpc.SentEvents, otelgrpc.ReceivedEvents))),
		grpc.ChainUnaryInterceptor(
			interceptor.ErrUnaryServerInterceptor(),
			intrcp.RecoveryUnaryServerInterceptor(log),
			interceptor.LoggingUnaryServerInterceptor(log),
			interceptor.AuthUnaryServerInterceptor(authcli),
			interceptor.ValidateUnaryServerInterceptor(val),
		),
	)

	srv := &Server{s}

	// Register reflection service on gRPC server.
	reflection.Register(srv.Server)

	return srv, nil
}

// Run binds the listener once every handler is registered. It is invoked from
// the module rather than appended in New so that its OnStop runs before the
// dependencies underneath close.
func Run(lc fx.Lifecycle, cfg *config.Config, log *slog.Logger, srv *Server) {
	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			l, err := net.Listen("tcp", cfg.Service.Addr)
			if err != nil {
				return err
			}

			log.Info("listening gRPC requests", slog.String("listen", cfg.Service.Addr))

			go func() {
				if err := srv.Serve(l); err != nil {
					log.Error("grpc server stopped", slog.Any("error", err))
				}
			}()

			return nil
		},
		OnStop: func(ctx context.Context) error {
			// Drain, but not past the shutdown budget: fx abandons the hooks
			// queued behind this one once the context expires, and those close
			// the pools and flush telemetry.
			done := make(chan struct{})

			go func() {
				srv.GracefulStop()
				close(done)
			}()

			select {
			case <-done:
			case <-ctx.Done():
				log.Warn("graceful stop exceeded the shutdown budget, closing connections")
				srv.Stop()
			}

			return nil
		},
	})
}
