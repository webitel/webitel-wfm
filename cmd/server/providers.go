package server

import (
	"context"
	"strings"
	"time"

	"go.uber.org/fx"

	authmanager "github.com/webitel/engine/pkg/wbt/auth_manager"
	"github.com/webitel/webitel-go-kit/infra/health"
	healthhttp "github.com/webitel/webitel-go-kit/infra/health/http"
	"github.com/webitel/webitel-go-kit/logging/wlog"
	authLogger "github.com/webitel/wlog"

	"github.com/webitel/webitel-wfm/config"
	"github.com/webitel/webitel-wfm/infra/registry"
	"github.com/webitel/webitel-wfm/infra/registry/provider/consul"
	"github.com/webitel/webitel-wfm/infra/storage/cache"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql/cluster"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql/pg"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql/scanner"
	"github.com/webitel/webitel-wfm/infra/webitel/engine"
	"github.com/webitel/webitel-wfm/infra/webitel/logger"
	"github.com/webitel/webitel-wfm/internal/model"
	"github.com/webitel/webitel-wfm/pkg/endpoint"
)

const (
	// sessionCacheSize is the maximum size of sessions to be cached.
	sessionCacheSize = 35000

	// sessionCacheTime is the duration in seconds for which a session will be cached.
	sessionCacheTime = 60 * 5
)

func ProvideLogger(cfg *config.Config) *wlog.Logger {
	log := wlog.NewLogger(&wlog.LoggerConfiguration{
		EnableConsole: cfg.Log.Console,
		ConsoleJson:   cfg.Log.JSON,
		ConsoleLevel:  cfg.Log.Level,
		EnableFile:    cfg.Log.File != "",
		FileJson:      cfg.Log.JSON,
		FileLevel:     cfg.Log.Level,
		FileLocation:  cfg.Log.File,
		EnableExport:  cfg.Log.Otel,
	})

	wlog.InitGlobalLogger(log.With(wlog.String("source", "global")))

	return log
}

func ProvideHealth() *health.Registry {
	return health.New(health.DefaultConfig(), nil)
}

func ProvideProbes(cfg *config.Config, h *health.Registry) *healthhttp.Server {
	return healthhttp.NewServer(h, cfg.Service.ProbeAddr)
}

func startHealth(lc fx.Lifecycle, log *wlog.Logger, cfg *config.Config, h *health.Registry, probes *healthhttp.Server) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := h.Start(ctx); err != nil {
				return err
			}

			if err := probes.Start(); err != nil {
				return err
			}

			log.Info("serving health probes", wlog.String("listen", cfg.Service.ProbeAddr))

			return nil
		},
		OnStop: func(ctx context.Context) error {
			return health.Shutdown(ctx, h, probes)
		},
	})
}

func ProvideDiscovery(cfg *config.Config, log *wlog.Logger) (*consul.Registry, error) {
	return consul.New(log, cfg.Consul.Addr, consul.WithHeartbeat(true), consul.WithTimeout(30*time.Second))
}

func registerService(lc fx.Lifecycle, cfg *config.Config, log *wlog.Logger, reg *consul.Registry) {
	instance := &registry.ServiceInstance{
		ID:      cfg.Service.NodeID,
		Name:    model.ServiceName,
		Version: model.Version,
		Metadata: map[string]string{
			"commit":         model.Commit,
			"commitDate":     model.CommitDate,
			"branch":         model.Branch,
			"buildTimestamp": model.BuildTimestamp,
		},
		Endpoints: []string{
			endpoint.NewEndpoint("grpc", cfg.Service.Addr).String(),
		},
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			log.Info("registering service in discovery", wlog.Any("service", instance))

			return reg.Register(ctx, instance)
		},
		OnStop: func(ctx context.Context) error {
			return reg.Deregister(ctx, instance)
		},
	})
}

func ProvideStorage(lc fx.Lifecycle, cfg *config.Config, log *wlog.Logger, h *health.Registry) (cluster.Store, error) {
	conn, err := newCluster(log, cfg.Postgres.DSN, cluster.WithUpdate())
	if err != nil {
		return nil, err
	}

	h.Critical("primary-database", conn.HealthCheck)
	lc.Append(fx.Hook{OnStop: func(context.Context) error { return conn.Close() }})

	return conn, nil
}

func ProvideForecastStorage(lc fx.Lifecycle, cfg *config.Config, log *wlog.Logger, h *health.Registry) (cluster.ForecastStore, error) {
	conn, err := newCluster(log, cfg.Forecast.DSN, cluster.WithUpdate())
	if err != nil {
		return nil, err
	}

	h.Critical("forecast-database", conn.HealthCheck)
	lc.Append(fx.Hook{OnStop: func(context.Context) error { return conn.Close() }})

	return conn, nil
}

// newCluster builds a cluster from a whitespace-separated list of DSNs.
func newCluster(log *wlog.Logger, dsns string, opts ...cluster.Option) (*cluster.Cluster, error) {
	fields := strings.Fields(dsns)
	nodes := make([]dbsql.Node, 0, len(fields))

	for _, dsn := range fields {
		db, err := pg.New(context.Background(), log, dsn)
		if err != nil {
			return nil, err
		}

		nodes = append(nodes, dbsql.New(dsn, db, scanner.MustNewDBScan()))
	}

	return cluster.New(log, nodes, opts...)
}

func ProvideCache(lc fx.Lifecycle, cfg *config.Config) (cache.Manager, error) {
	c, err := cache.New(&cfg.Cache)
	if err != nil {
		return nil, err
	}

	lc.Append(fx.Hook{OnStop: func(context.Context) error {
		c.Stop()

		return nil
	}})

	return c, nil
}

func ProvideAuth(lc fx.Lifecycle, cfg *config.Config, log *wlog.Logger, h *health.Registry) authmanager.AuthManager {
	al := authLogger.NewLogger(&authLogger.LoggerConfiguration{
		EnableConsole: true,
		ConsoleLevel:  authLogger.LevelDebug,
		EnableExport:  true,
	})

	conn := authmanager.NewAuthManager(sessionCacheSize, sessionCacheTime, cfg.Consul.Addr, al)

	h.Informational("webitel-auth", func(context.Context) error { return nil })

	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			log.Info("connecting to Webitel Auth manager")

			go func() {
				if err := conn.Start(); err != nil {
					log.Error("auth manager stopped", wlog.Err(err))
				}
			}()

			return nil
		},
		OnStop: func(context.Context) error {
			conn.Stop()

			return nil
		},
	})

	return conn
}

func ProvideEngine(lc fx.Lifecycle, log *wlog.Logger, d registry.Discovery) (*engine.Client, error) {
	cli, err := engine.New(log, d)
	if err != nil {
		return nil, err
	}

	lc.Append(fx.Hook{OnStop: func(context.Context) error { return cli.Close() }})

	return cli, nil
}

func ProvideLoggerClient(lc fx.Lifecycle, log *wlog.Logger, d registry.Discovery) (*logger.Client, error) {
	cli, err := logger.New(log, d)
	if err != nil {
		return nil, err
	}

	lc.Append(fx.Hook{OnStop: func(context.Context) error { return cli.Close() }})

	return cli, nil
}
