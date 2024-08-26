package webitel

import (
	"errors"
	"fmt"
	"time"

	"github.com/webitel/engine/discovery"
	"github.com/webitel/webitel-go-kit/logging/wlog"
)

const (
	watcherInterval = 5 * 1000
)

type ConnectionManager[T any] struct {
	svc string
	log *wlog.Logger

	discovery discovery.ServiceDiscovery
	pool      discovery.Pool

	stop chan struct{}
}

func New[T any](log *wlog.Logger, sd discovery.ServiceDiscovery, svc string) (*ConnectionManager[T], error) {
	c := &ConnectionManager[T]{
		svc:       svc,
		log:       log,
		discovery: sd,
		pool:      discovery.NewPoolConnections(),
		stop:      make(chan struct{}),
	}

	if err := c.recheckConnections(); err != nil {
		return nil, err
	}

	if conn := c.pool.All(); len(conn) < 1 {
		return nil, fmt.Errorf("no connections available: %s", svc)
	}

	return c, nil
}

func (c *ConnectionManager[T]) Start() error {
	c.log.Debug(fmt.Sprintf("watcher [%s] started", c.svc))

	for {
		select {
		case <-c.stop:
			c.log.Debug(fmt.Sprintf("watcher [%s] received stop signal", c.svc))

			return nil
		case <-time.After(time.Duration(watcherInterval) * time.Millisecond):
			if err := c.recheckConnections(); err != nil {
				return err
			}
		}
	}
}

func (c *ConnectionManager[T]) Stop() {
	close(c.stop)
	if c.pool != nil {
		c.pool.CloseAllConnections()
	}
}

func (c *ConnectionManager[T]) Connection() (T, error) {
	conn, err := c.pool.Get(discovery.StrategyRoundRobin)
	if err != nil {
		var zero T

		return zero, err
	}

	return conn.(T), nil
}

func (c *ConnectionManager[T]) recheckConnections() error {
	list, err := c.discovery.GetByName(c.svc)
	if err != nil {
		return fmt.Errorf("get service list: %w", err)
	}

	for _, v := range list {
		if _, err := c.pool.GetById(v.Id); errors.Is(err, discovery.ErrNotFoundConnection) {
			mgr, err := NewConnection(v)
			if err != nil {
				c.log.Warn("register new connection", wlog.Err(err), wlog.String("id", v.Id), wlog.String("host", v.Host))

				continue
			}

			c.pool.Append(mgr)
		}
	}

	c.pool.RecheckConnections(list.Ids())
	if conn := c.pool.All(); len(conn) < 1 {
		return fmt.Errorf("no connections available: %s", c.svc)
	}

	return nil
}
