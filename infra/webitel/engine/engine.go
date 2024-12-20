package engine

import (
	"context"

	"github.com/webitel/engine/discovery"
	"github.com/webitel/webitel-go-kit/logging/wlog"

	"github.com/webitel/webitel-wfm/infra/health"
	"github.com/webitel/webitel-wfm/infra/shutdown"
	"github.com/webitel/webitel-wfm/infra/webitel"
)

var serviceName = "engine"

type Client struct {
	*AgentService
	*CalendarService

	Conn *webitel.ConnectionManager[*webitel.Connection]
}

func New(log *wlog.Logger, sd discovery.ServiceDiscovery) (*Client, error) {
	c, err := webitel.New[*webitel.Connection](log, sd, serviceName)
	if err != nil {
		return nil, err
	}

	agentSvc, err := NewAgentServiceClient(log, c)
	if err != nil {
		return nil, err
	}

	calendarSvc, err := NewCalendarServiceClient(log, c)
	if err != nil {
		return nil, err
	}

	cli := &Client{
		Conn:            c,
		AgentService:    agentSvc,
		CalendarService: calendarSvc,
	}

	return cli, nil
}

func (c *Client) Shutdown(p *shutdown.Process) error {
	c.Conn.Stop()

	return nil
}

func (c *Client) HealthCheck(ctx context.Context) []health.CheckResult {
	_, err := c.Conn.Connection()
	if err != nil {
		return []health.CheckResult{{Name: serviceName, Err: err}}
	}

	return []health.CheckResult{{Name: serviceName, Err: nil}}
}
