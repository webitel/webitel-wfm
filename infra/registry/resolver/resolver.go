package resolver

import (
	"context"
	"time"

	"github.com/webitel/webitel-go-kit/logging/wlog"
	"google.golang.org/grpc/attributes"
	"google.golang.org/grpc/resolver"

	"github.com/webitel/webitel-wfm/infra/registry"
	"github.com/webitel/webitel-wfm/pkg/endpoint"
	"github.com/webitel/webitel-wfm/pkg/subset"
	"github.com/webitel/webitel-wfm/pkg/werror"
)

type discoveryResolver struct {
	log *wlog.Logger

	w  registry.Watcher
	cc resolver.ClientConn

	ctx    context.Context
	cancel context.CancelFunc

	insecure    bool
	debugLog    bool
	selectorKey string
	subsetSize  int
}

func (r *discoveryResolver) watch() {
	for {
		select {
		case <-r.ctx.Done():
			return
		default:
		}

		ins, err := r.w.Next()
		if err != nil {
			if werror.Is(err, context.Canceled) {
				return
			}

			r.log.Error("watch discovery endpoint", wlog.Err(err))
			time.Sleep(time.Second)

			continue
		}

		r.update(ins)
	}
}

func (r *discoveryResolver) update(ins []*registry.ServiceInstance) {
	var (
		endpoints = make(map[string]struct{})
		filtered  = make([]*registry.ServiceInstance, 0, len(ins))
	)

	for _, in := range ins {
		ept, err := endpoint.ParseEndpoint(in.Endpoints, endpoint.Scheme("grpc", !r.insecure))
		if err != nil {
			r.log.Error("parse discovery endpoint", wlog.Err(err), wlog.String("endpoint", in.String()))

			continue
		}

		if ept == "" {
			continue
		}

		// filter redundant endpoints
		if _, ok := endpoints[ept]; ok {
			continue
		}

		endpoints[ept] = struct{}{}
		filtered = append(filtered, in)
	}

	if r.subsetSize != 0 {
		filtered = subset.Subset(r.selectorKey, filtered, r.subsetSize)
	}

	addrs := make([]resolver.Address, 0, len(filtered))
	for _, in := range filtered {
		ept, _ := endpoint.ParseEndpoint(in.Endpoints, endpoint.Scheme("grpc", !r.insecure))
		addr := resolver.Address{
			ServerName: in.Name,
			Attributes: parseAttributes(in.Metadata).WithValue("rawServiceInstance", in),
			Addr:       ept,
		}

		addrs = append(addrs, addr)
	}

	if len(addrs) == 0 {
		r.log.Warn("zero endpoint found, refused to write", wlog.Any("instances", ins), wlog.Any("filtered", filtered))

		return
	}

	if err := r.cc.UpdateState(resolver.State{Addresses: addrs}); err != nil {
		r.log.Error("update addr state", wlog.Err(err), wlog.Any("addr", addrs))
	}

	if r.debugLog {
		r.log.Info("update instances", wlog.Any("instances", filtered))
	}
}

func (r *discoveryResolver) Close() {
	r.cancel()
	if err := r.w.Stop(); err != nil {
		r.log.Error("watch stop", wlog.Err(err))
	}
}

func (r *discoveryResolver) ResolveNow(_ resolver.ResolveNowOptions) {}

func parseAttributes(md map[string]string) (a *attributes.Attributes) {
	for k, v := range md {
		a = a.WithValue(k, v)
	}

	return a
}
