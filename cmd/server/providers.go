package server

import (
	"context"
	"errors"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"go.uber.org/fx"

	"github.com/webitel/webitel-go-kit/appconfig"
	"github.com/webitel/webitel-go-kit/infra/discovery"
	"github.com/webitel/webitel-go-kit/infra/health"
	healthhttp "github.com/webitel/webitel-go-kit/infra/health/http"
	"github.com/webitel/webitel-go-kit/infra/pgw"
	"github.com/webitel/webitel-go-kit/infra/pubsub/rabbitmq"
	rabbitslog "github.com/webitel/webitel-go-kit/infra/pubsub/rabbitmq/pkg/adapter/slog"
	"github.com/webitel/webitel-go-kit/pkg/cache"

	"github.com/webitel/webitel-wfm/config"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql"
	"github.com/webitel/webitel-wfm/infra/webitel/auth"
	"github.com/webitel/webitel-wfm/infra/webitel/engine"
	"github.com/webitel/webitel-wfm/infra/webitel/logger"
	"github.com/webitel/webitel-wfm/internal/model"

	// Drivers register themselves on import: the consul discovery provider, and
	// the OTel exporters that OTEL_*_EXPORTER selects at runtime.
	_ "github.com/webitel/webitel-go-kit/infra/discovery/consul"
	_ "github.com/webitel/webitel-go-kit/infra/otel/sdk/log/otlp"
	_ "github.com/webitel/webitel-go-kit/infra/otel/sdk/log/stdout"
	_ "github.com/webitel/webitel-go-kit/infra/otel/sdk/metric/otlp"
	_ "github.com/webitel/webitel-go-kit/infra/otel/sdk/metric/stdout"
	_ "github.com/webitel/webitel-go-kit/infra/otel/sdk/trace/otlp"
	_ "github.com/webitel/webitel-go-kit/infra/otel/sdk/trace/stdout"
)

const (
	// cacheTTL bounds how long a storage entry may be served without a database read.
	cacheTTL = 24 * time.Hour

	// defaultCacheEntries applies when cache.size is unset; maxCacheEntries
	// catches a value still written in bytes, which ristretto would turn into
	// a counter array large enough to exhaust memory at startup.
	defaultCacheEntries = 1024
	maxCacheEntries     = 1 << 20

	// writeTimeout stops a broker applying TCP pushback from parking a publish
	// forever with the publisher mutex held.
	writeTimeout = 5 * time.Second
)

func ProvideHealth(log *slog.Logger) *health.Registry {
	return health.New(health.DefaultConfig(), log)
}

func ProvideProbes(cfg *config.Config, h *health.Registry) *healthhttp.Server {
	return healthhttp.NewServer(h, cfg.Service.ProbeAddr)
}

func startHealth(lc fx.Lifecycle, log *slog.Logger, cfg *config.Config, h *health.Registry, probes *healthhttp.Server) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := h.Start(ctx); err != nil {
				return err
			}

			if err := probes.Start(); err != nil {
				return err
			}

			log.Info("serving health probes", slog.String("listen", cfg.Service.ProbeAddr))

			return nil
		},
		OnStop: func(ctx context.Context) error {
			return health.Shutdown(ctx, h, probes)
		},
	})
}

func ProvideDiscovery(cfg *config.Config, log *slog.Logger) (discovery.DiscoveryProvider, error) {
	return discovery.DefaultFactory.CreateProvider(discovery.ProviderConsul, log, cfg.Consul.Addr,
		discovery.WithHeartbeat[discovery.DiscoveryProvider](true),
		discovery.WithTimeout[discovery.DiscoveryProvider](30*time.Second),
		discovery.WithTags[discovery.DiscoveryProvider]("version="+model.Version),
	)
}

func registerService(lc fx.Lifecycle, cfg *config.Config, log *slog.Logger, dp discovery.DiscoveryProvider) {
	instance := &discovery.ServiceInstance{
		Id:      cfg.Service.NodeID,
		Name:    model.ServiceName,
		Version: model.Version,
		Metadata: map[string]string{
			"commit":         model.Commit,
			"commitDate":     model.CommitDate,
			"branch":         model.Branch,
			"buildTimestamp": model.BuildTimestamp,
		},
		Endpoints: []string{
			(&url.URL{Scheme: "grpc", Host: cfg.Service.Addr}).String(),
		},
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			log.Info("registering service in discovery", slog.Any("service", instance))

			return dp.Register(ctx, instance)
		},
		OnStop: func(ctx context.Context) error {
			return dp.Deregister(ctx, instance)
		},
	})
}

func ProvideStorage(lc fx.Lifecycle, cfg *config.Config, h *health.Registry) (dbsql.Store, error) {
	db, err := newDB(cfg.Postgres)
	if err != nil {
		return nil, err
	}

	h.Critical("primary-database", db.HealthCheck)
	lc.Append(fx.Hook{OnStop: func(context.Context) error { return db.Close() }})

	return db, nil
}

func ProvideForecastStorage(lc fx.Lifecycle, cfg *config.Config, h *health.Registry) (dbsql.ForecastStore, error) {
	db, err := newDB(cfg.Forecast)
	if err != nil {
		return nil, err
	}

	h.Critical("forecast-database", db.HealthCheck)
	lc.Append(fx.Hook{OnStop: func(context.Context) error { return db.Close() }})

	return db, nil
}

// newDB takes the first DSN of a whitespace-separated list as the primary and
// any further ones as standbys.
func newDB(pg appconfig.Postgres) (*dbsql.DB, error) {
	dsns := strings.Fields(pg.DSN)
	if len(dsns) == 0 {
		return nil, errors.New("postgres dsn is required")
	}

	pm, err := pgw.NewPoolManager(context.Background(),
		pgw.WithApplicationName(model.ServiceName),
		pgw.WithTracer(dbsql.NewTracer()),
		pgw.WithPrimaryConfig(pgw.PrimaryConfig{DSN: dsns[0], MaxConns: pg.MaxOpenConns}),
		pgw.WithStandbyConfig(pgw.StandbyConfig{DSN: dsns[1:], MaxConns: pg.MaxOpenConns}),
	)
	if err != nil {
		return nil, err
	}

	return dbsql.New(pm), nil
}

func ProvideCache(cfg *config.Config, log *slog.Logger) cache.RistrettoConfig {
	size := cfg.Cache.Size
	if size <= 0 {
		size = defaultCacheEntries
	}

	if size > maxCacheEntries {
		log.Warn("cache.size counts entries, not bytes; capping",
			slog.Int("configured", size), slog.Int("used", maxCacheEntries),
		)

		size = maxCacheEntries
	}

	return cache.RistrettoConfig{MaxCost: int64(size), TTL: cacheTTL}
}

func ProvideAuth(lc fx.Lifecycle, d discovery.DiscoveryProvider, h *health.Registry) (auth.Manager, error) {
	cli, err := auth.New(d)
	if err != nil {
		return nil, err
	}

	h.Informational("webitel-auth", cli.HealthCheck)
	lc.Append(fx.Hook{OnStop: func(context.Context) error { return cli.Close() }})

	return cli, nil
}

func ProvideEngine(lc fx.Lifecycle, d discovery.DiscoveryProvider) (*engine.Client, error) {
	cli, err := engine.New(d)
	if err != nil {
		return nil, err
	}

	lc.Append(fx.Hook{OnStop: func(context.Context) error { return cli.Close() }})

	return cli, nil
}

func ProvideLoggerClient(lc fx.Lifecycle, d discovery.DiscoveryProvider) (*logger.Client, error) {
	cli, err := logger.New(d)
	if err != nil {
		return nil, err
	}

	lc.Append(fx.Hook{OnStop: func(context.Context) error { return cli.Close() }})

	return cli, nil
}

func ProvidePublisher(lc fx.Lifecycle, cfg *config.Config, log *slog.Logger) (*rabbitmq.MessagePublisher, error) {
	rmqlog := rabbitslog.NewSlogLogger(log)

	rcfg, err := rabbitmq.NewConfig(cfg.Pubsub.URL, rabbitmq.WithWriteTimeout(writeTimeout))
	if err != nil {
		return nil, err
	}

	conn, err := rabbitmq.NewConnection(rcfg, rmqlog)
	if err != nil {
		return nil, err
	}

	// One attempt per message, acknowledged by the broker, as before. A failed
	// publish still costs the library's one-second backoff before it returns.
	pcfg, err := rabbitmq.NewPublisherConfig(rabbitmq.WithPublisherMaxRetries(1))
	if err != nil {
		_ = conn.Close()

		return nil, err
	}

	pub, err := rabbitmq.NewPublisher(conn, pcfg, rmqlog)
	if err != nil {
		_ = conn.Close()

		return nil, err
	}

	lc.Append(fx.Hook{OnStop: func(context.Context) error {
		_ = pub.Close()

		return conn.Close()
	}})

	return pub, nil
}
