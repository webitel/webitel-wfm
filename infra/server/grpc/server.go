package server

import (
	"context"
	"fmt"
	"net"

	"buf.build/go/protovalidate"
	"github.com/webitel/engine/pkg/wbt/auth_manager"
	"github.com/webitel/webitel-go-kit/logging/wlog"
	otelgrpc "github.com/webitel/webitel-go-kit/tracing/grpc"
	"go.uber.org/fx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/webitel/webitel-wfm/config"
	"github.com/webitel/webitel-wfm/infra/server/interceptor"
)

var Module = fx.Module("grpc_server",
	fx.Provide(New),
	fx.Invoke(Run),
)

type Server struct {
	*grpc.Server
}

// New provides a new gRPC server.
func New(log *wlog.Logger, authcli auth_manager.AuthManager) (*Server, error) {
	val, err := protovalidate.New()
	if err != nil {
		return nil, fmt.Errorf("construct protovalidate rules: %w", err)
	}

	s := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler(otelgrpc.WithMessageEvents(otelgrpc.SentEvents, otelgrpc.ReceivedEvents))),
		grpc.ChainUnaryInterceptor(
			interceptor.ErrUnaryServerInterceptor(),
			interceptor.RecoveryUnaryServerInterceptor(log),
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
func Run(lc fx.Lifecycle, cfg *config.Config, log *wlog.Logger, srv *Server) {
	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			l, err := net.Listen("tcp", cfg.Service.Addr)
			if err != nil {
				return err
			}

			log.Info("listening gRPC requests", wlog.String("listen", cfg.Service.Addr))

			go func() {
				if err := srv.Serve(l); err != nil {
					log.Error("grpc server stopped", wlog.Err(err))
				}
			}()

			return nil
		},
		OnStop: func(context.Context) error {
			srv.GracefulStop()

			return nil
		},
	})
}
