package logger

import (
	"context"

	"github.com/webitel/engine/discovery"
	"github.com/webitel/webitel-go-kit/logging/wlog"

	"github.com/webitel/webitel-wfm/infra/health"
	"github.com/webitel/webitel-wfm/infra/shutdown"
	"github.com/webitel/webitel-wfm/infra/webitel"
)

var serviceName = "logger"

type Client struct {
	*ConfigService

	Сonn *webitel.ConnectionManager[*webitel.Connection]
}

func New(log *wlog.Logger, sd discovery.ServiceDiscovery) (*Client, error) {
	c, err := webitel.New[*webitel.Connection](log, sd, serviceName)
	if err != nil {
		return nil, err
	}

	cfgSvc, err := NewConfigServiceClient(log, c)
	if err != nil {
		return nil, err
	}

	return &Client{Сonn: c, ConfigService: cfgSvc}, nil
}

func (c *Client) Shutdown(p *shutdown.Process) error {
	c.Сonn.Stop()

	return nil
}

func (c *Client) HealthCheck(ctx context.Context) []health.CheckResult {
	_, err := c.Сonn.Connection()
	if err != nil {
		return []health.CheckResult{{Name: serviceName, Err: err}}
	}

	return []health.CheckResult{{Name: serviceName, Err: nil}}
}
