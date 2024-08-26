package testinfra

import (
	"context"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

type TestGrpcClient struct {
	Conn *grpc.ClientConn
}

func NewTestGrpcClient(t *testing.T, lis *bufconn.Listener) *grpc.ClientConn {
	t.Helper()
	dialer := func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}

	conn, err := grpc.NewClient("127.0.0.1", grpc.WithContextDialer(dialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("dial grpc listener: %v", err)
	}

	return conn
}
