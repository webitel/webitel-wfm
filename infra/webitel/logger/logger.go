package logger

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"

	"github.com/webitel/webitel-go-kit/infra/discovery"
	"github.com/webitel/webitel-go-kit/pkg/errors"

	"github.com/webitel/webitel-wfm/infra/webitel"
)

var serviceName = "logger"

type Client struct {
	conn *grpc.ClientConn

	ConfigService *ConfigService
}

func New(dp discovery.Discovery) (*Client, error) {
	conn, err := webitel.New(dp, serviceName)
	if err != nil {
		return nil, err
	}

	return &Client{conn: conn, ConfigService: newConfigServiceClient(conn)}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) HealthCheck(ctx context.Context) error {
	state := c.conn.GetState()
	if state != connectivity.Idle && state != connectivity.Ready {
		return errors.New("service is not ready", errors.WithValue("state", state.String()))
	}

	return nil
}
