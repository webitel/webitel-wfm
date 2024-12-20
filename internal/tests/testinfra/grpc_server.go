package testinfra

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/webitel/engine/auth_manager"
	"github.com/webitel/webitel-go-kit/logging/wlog"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"

	authmock "github.com/webitel/webitel-wfm/gen/go/mocks/infra/webitel/auth"
	"github.com/webitel/webitel-wfm/infra/server"
)

const BufSize = 1024 * 1024

type TestServer struct {
	Ctx context.Context

	Server *server.Server
	Lis    *bufconn.Listener

	Session *auth_manager.Session
}

func NewTestServer(t *testing.T, log *wlog.Logger) *TestServer {
	t.Helper()

	session := &auth_manager.Session{
		DomainId: 1,
		UserId:   3,
		RoleIds:  []int{1},
		Expire:   time.Now().Add(1 * time.Hour).Unix(),
	}

	a := authmock.NewMockManager(t)
	a.EXPECT().GetSession(mock.AnythingOfType("string")).RunAndReturn(func(token string) (*auth_manager.Session, error) {
		session.Id = token
		session.Token = token

		return session, nil
	}).Maybe()

	server, err := server.New(log, a)
	if err != nil {
		t.Errorf("grpc server: %v", err)
	}

	return &TestServer{
		Server:  server,
		Session: session,
		Lis:     bufconn.Listen(BufSize),
		Ctx:     NewOutgoingContext(t, 5*time.Second),
	}
}

func (s *TestServer) Serve() error {
	return s.Server.Serve(s.Lis)
}

func NewOutgoingContext(t *testing.T, duration time.Duration) context.Context {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), duration)
	t.Cleanup(func() {
		cancel()
	})

	md := metadata.New(map[string]string{
		"X-Webitel-Access": "super-auth-token",
	})

	return metadata.NewOutgoingContext(ctx, md)
}
