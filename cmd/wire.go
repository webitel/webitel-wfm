//go:build wireinject
// +build wireinject

package cmd

import (
	"context"

	"github.com/google/wire"
	"github.com/webitel/webitel-go-kit/logging/wlog"
	"google.golang.org/grpc"

	"github.com/webitel/webitel-wfm/config"
	"github.com/webitel/webitel-wfm/infra"
	"github.com/webitel/webitel-wfm/infra/health"
	"github.com/webitel/webitel-wfm/infra/registry"
	"github.com/webitel/webitel-wfm/infra/registry/provider/consul"
	"github.com/webitel/webitel-wfm/infra/server"
	"github.com/webitel/webitel-wfm/infra/shutdown"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql/cluster"
	"github.com/webitel/webitel-wfm/infra/webitel/logger"
	"github.com/webitel/webitel-wfm/internal/handler"
	"github.com/webitel/webitel-wfm/internal/service"
	"github.com/webitel/webitel-wfm/internal/storage"
)

func initResources(context.Context, *config.Config, *wlog.Logger, *health.CheckRegistry, *shutdown.Tracker) (*resources, error) {
	panic(wire.Build(sqlStorage, wire.Bind(new(cluster.Store), new(*cluster.Cluster)), auth, infra.Set,
		serviceDiscovery, wire.Bind(new(registry.Discovery), new(*consul.Registry)),
		wire.FieldsOf(new(*config.Config), "Cache", "Pubsub"),
		wire.FieldsOf(new(*logger.Client), "ConfigService"),
		wire.Struct(new(resources), "*")),
	)
}

func initHandlers(*resources, cluster.ForecastStore) (*handler.Handlers, error) {
	panic(wire.Build(storage.Set, service.Set, handler.Set, wire.Bind(new(grpc.ServiceRegistrar), new(*server.Server)),
		wire.FieldsOf(new(*resources), "grpcServer", "cache", "storage", "engine", "audit"),
		wire.Struct(new(handler.Handlers), "*"),
	))
}
