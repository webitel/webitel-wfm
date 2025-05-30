package consul

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/webitel/webitel-go-kit/logging/wlog"

	"github.com/webitel/webitel-wfm/infra/registry"
	"github.com/webitel/webitel-wfm/pkg/werror"
)

type Datacenter string

const (
	SingleDatacenter Datacenter = "SINGLE"
	MultiDatacenter  Datacenter = "MULTI"
)

// Client is consul client config
type Client struct {
	log *wlog.Logger

	dc     Datacenter
	cli    *api.Client
	ctx    context.Context
	cancel context.CancelFunc

	// resolve service entry endpoints
	resolver ServiceResolver

	// healthcheck time interval in seconds
	healthcheckInterval int

	// heartbeat enable heartbeat
	heartbeat bool

	// deregisterCriticalServiceAfter time interval in seconds
	deregisterCriticalServiceAfter int

	// serviceChecks user custom checks
	serviceChecks api.AgentServiceChecks
}

func defaultResolver(_ context.Context, entries []*api.ServiceEntry) []*registry.ServiceInstance {
	services := make([]*registry.ServiceInstance, 0, len(entries))
	for _, entry := range entries {
		var version string
		for _, tag := range entry.Service.Tags {
			ss := strings.SplitN(tag, "=", 2)
			if len(ss) == 2 && ss[0] == "version" {
				version = ss[1]
			}
		}

		endpoints := make([]string, 0)
		for scheme, addr := range entry.Service.TaggedAddresses {
			if scheme == "lan_ipv4" || scheme == "wan_ipv4" || scheme == "lan_ipv6" || scheme == "wan_ipv6" {
				continue
			}

			endpoints = append(endpoints, addr.Address)
		}

		if len(endpoints) == 0 && entry.Service.Address != "" && entry.Service.Port != 0 {
			endpoints = append(endpoints, fmt.Sprintf("grpc://%s:%d", entry.Service.Address, entry.Service.Port))
		}

		services = append(services, &registry.ServiceInstance{
			ID:        entry.Service.ID,
			Name:      entry.Service.Service,
			Metadata:  entry.Service.Meta,
			Version:   version,
			Endpoints: endpoints,
		})
	}

	return services
}

// ServiceResolver is used to resolve service endpoints
type ServiceResolver func(ctx context.Context, entries []*api.ServiceEntry) []*registry.ServiceInstance

// Service get services from consul
func (c *Client) Service(ctx context.Context, service string, index uint64, passingOnly bool) ([]*registry.ServiceInstance, uint64, error) {
	if c.dc == MultiDatacenter {
		return c.multiDCService(ctx, service, index, passingOnly)
	}

	opts := &api.QueryOptions{
		WaitIndex:  index,
		WaitTime:   time.Second * 55,
		Datacenter: string(c.dc),
	}

	opts = opts.WithContext(ctx)
	if c.dc == SingleDatacenter {
		opts.Datacenter = ""
	}

	entries, meta, err := c.singleDCEntries(service, "", passingOnly, opts)
	if err != nil {
		return nil, 0, err
	}

	return c.resolver(ctx, entries), meta.LastIndex, nil
}

func (c *Client) multiDCService(ctx context.Context, service string, index uint64, passingOnly bool) ([]*registry.ServiceInstance, uint64, error) {
	opts := &api.QueryOptions{
		WaitIndex: index,
		WaitTime:  time.Second * 55,
	}

	opts = opts.WithContext(ctx)

	var instances []*registry.ServiceInstance
	dcs, err := c.cli.Catalog().Datacenters()
	if err != nil {
		return nil, 0, err
	}

	for _, dc := range dcs {
		opts.Datacenter = dc
		e, m, err := c.singleDCEntries(service, "", passingOnly, opts)
		if err != nil {
			return nil, 0, err
		}

		ins := c.resolver(ctx, e)
		for _, in := range ins {
			if in.Metadata == nil {
				in.Metadata = make(map[string]string, 1)
			}

			in.Metadata["dc"] = dc
		}

		instances = append(instances, ins...)
		opts.WaitIndex = m.LastIndex
	}

	return instances, opts.WaitIndex, nil
}

func (c *Client) singleDCEntries(service, tag string, passingOnly bool, opts *api.QueryOptions) ([]*api.ServiceEntry, *api.QueryMeta, error) {
	return c.cli.Health().Service(service, tag, passingOnly, opts)
}

// Register register service instance to consul
func (c *Client) Register(_ context.Context, svc *registry.ServiceInstance, enableHealthCheck bool) error {
	addresses := make(map[string]api.ServiceAddress, len(svc.Endpoints))
	checkAddresses := make([]string, 0, len(svc.Endpoints))
	for _, endpoint := range svc.Endpoints {
		raw, err := url.Parse(endpoint)
		if err != nil {
			return err
		}

		addr := raw.Hostname()
		port, _ := strconv.ParseUint(raw.Port(), 10, 16)

		checkAddresses = append(checkAddresses, net.JoinHostPort(addr, strconv.FormatUint(port, 10)))
		addresses[raw.Scheme] = api.ServiceAddress{Address: addr, Port: int(port)}
	}

	asr := &api.AgentServiceRegistration{
		ID:              svc.ID,
		Name:            svc.Name,
		Meta:            svc.Metadata,
		Tags:            []string{fmt.Sprintf("version=%s", svc.Version)},
		TaggedAddresses: addresses,
	}

	if len(checkAddresses) > 0 {
		host, portRaw, _ := net.SplitHostPort(checkAddresses[0])
		port, _ := strconv.ParseInt(portRaw, 10, 32)
		asr.Address = host
		asr.Port = int(port)
	}

	if enableHealthCheck {
		for _, address := range checkAddresses {
			asr.Checks = append(asr.Checks, &api.AgentServiceCheck{
				Notes:                          fmt.Sprintf("TCP health check for address %s", address),
				TCP:                            address,
				Interval:                       fmt.Sprintf("%ds", c.healthcheckInterval),
				DeregisterCriticalServiceAfter: fmt.Sprintf("%ds", c.deregisterCriticalServiceAfter),
				Timeout:                        "5s",
			})
		}

		// custom checks
		asr.Checks = append(asr.Checks, c.serviceChecks...)
	}

	if c.heartbeat {
		asr.Checks = append(asr.Checks, &api.AgentServiceCheck{
			CheckID:                        "service:" + svc.ID + ":ttl:1",
			Notes:                          "TTL heartbeat check",
			TTL:                            fmt.Sprintf("%ds", c.healthcheckInterval*2),
			DeregisterCriticalServiceAfter: fmt.Sprintf("%ds", c.deregisterCriticalServiceAfter),
		})
	}

	if err := c.cli.Agent().ServiceRegister(asr); err != nil {
		return err
	}

	if c.heartbeat {
		go func() {
			ticker := time.NewTicker(time.Second * time.Duration(c.healthcheckInterval))
			defer ticker.Stop()

			for {
				select {
				case <-c.ctx.Done():
					if err := c.cli.Agent().ServiceDeregister(svc.ID); err != nil {
						// TODO: we deregister service using shutdown hook, that cancel a context and trigger channel.
						// 	defined context is using in watch registry for gRPC resolver.
						if !strings.Contains(err.Error(), "response code: 404") {
							c.log.Error("deregister service after context is done", wlog.Err(err), wlog.String("ctx", c.ctx.Err().Error()))
						}
					}

					return
				case <-ticker.C:
					// ensure that unregistered services will not be re-registered by mistake
					if werror.Is(c.ctx.Err(), context.Canceled) || werror.Is(c.ctx.Err(), context.DeadlineExceeded) {
						if err := c.cli.Agent().ServiceDeregister(svc.ID); err != nil {
							c.log.Error("deregister service if service was unregistered", wlog.Err(err))
						}

						return
					}

					if err := c.cli.Agent().UpdateTTLOpts("service:"+svc.ID+":ttl:1", "pass", "pass", new(api.QueryOptions).WithContext(c.ctx)); err != nil {
						if werror.Is(err, context.Canceled) || werror.Is(err, context.DeadlineExceeded) {
							if err := c.cli.Agent().ServiceDeregister(svc.ID); err != nil {
								c.log.Error("deregister service after unsuccessful updating service ttl heartbeat", wlog.Err(err))
							}

							return
						}

						c.log.Error("update ttl heartbeat to consul", wlog.Err(err))

						// when the previous report fails, try to re register the service
						time.Sleep(time.Duration(rand.Intn(5)) * time.Second)
						if err := c.cli.Agent().ServiceRegister(asr); err != nil {
							c.log.Error("re-register service", wlog.Err(err))
						} else {
							c.log.Warn("re-register of service occurred success")
						}
					}
				}
			}
		}()
	}

	return nil
}

// Deregister service by service ID
func (c *Client) Deregister(_ context.Context, serviceID string) error {
	defer c.cancel()

	return c.cli.Agent().ServiceDeregister(serviceID)
}
