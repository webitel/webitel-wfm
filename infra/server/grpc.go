package server

import (
	"fmt"

	"github.com/bufbuild/protovalidate-go"
	"github.com/webitel/engine/auth_manager"
	"github.com/webitel/webitel-go-kit/logging/wlog"
	otelgrpc "github.com/webitel/webitel-go-kit/tracing/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/webitel/webitel-wfm/infra/server/interceptor"
	"github.com/webitel/webitel-wfm/infra/shutdown"
)

type Server struct {
	*grpc.Server
}

// New provides a new gRPC server.
func New(log *wlog.Logger, authcli auth_manager.AuthManager) (*Server, error) {
	val, err := protovalidate.New(protovalidate.WithFailFast(true))
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

	// Register reflection service on gRPC server.
	reflection.Register(s)

	return &Server{s}, nil
}

func (s *Server) Shutdown(p *shutdown.Process) error {
	s.Server.GracefulStop()
	p.MarkOutstandingRequestsCompleted()

	return nil
}
