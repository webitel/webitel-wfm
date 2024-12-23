package authmanager

import (
	"context"
	"net"
	"strconv"
	"time"

	"github.com/webitel/engine/discovery"

	"github.com/webitel/webitel-wfm/infra/registry"
	"github.com/webitel/webitel-wfm/pkg/endpoint"
)

var _ discovery.ServiceDiscovery = (*Discovery)(nil)

type Discovery struct {
	discovery registry.Discovery
}

func New(discovery registry.Discovery) *Discovery {
	return &Discovery{
		discovery: discovery,
	}
}

func (d *Discovery) RegisterService(name string, pubHost string, pubPort int, ttl time.Duration, criticalTtl time.Duration) error {
	panic("implement me")
}

func (d *Discovery) Shutdown() {
	panic("implement me")
}

func (d *Discovery) GetByName(serviceName string) (discovery.ListConnections, error) {
	svcs, err := d.discovery.GetService(context.Background(), serviceName)
	if err != nil {
		return nil, err
	}

	conns := make([]*discovery.ServiceConnection, 0, len(svcs))
	for _, svc := range svcs {
		var (
			host string
			port int
		)

		for _, e := range svc.Endpoints {
			if e != "" {
				url, err := endpoint.ParseEndpoint([]string{e}, endpoint.Scheme("grpc", false))
				if err != nil {
					return nil, err
				}

				h, p, err := net.SplitHostPort(url)
				if err != nil {
					continue
				}

				host = h
				i, err := strconv.Atoi(p)
				if err != nil {
					continue
				}

				port = i
			}
		}

		conns = append(conns, &discovery.ServiceConnection{
			Id:      svc.ID,
			Service: svc.Name,
			Host:    host,
			Port:    port,
		})
	}

	return conns, nil
}
