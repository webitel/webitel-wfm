package cmd

import (
	"context"
	"net"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/urfave/cli/v2"
	authmanager "github.com/webitel/engine/auth_manager"
	"github.com/webitel/webitel-go-kit/logging/wlog"
	"golang.org/x/sync/errgroup"

	// _ "github.com/webitel/webitel-go-kit/otel/sdk/log/otlp"
	// _ "github.com/webitel/webitel-go-kit/otel/sdk/log/stdout"
	// _ "github.com/webitel/webitel-go-kit/otel/sdk/metric/otlp"
	// _ "github.com/webitel/webitel-go-kit/otel/sdk/metric/stdout"
	// _ "github.com/webitel/webitel-go-kit/otel/sdk/trace/otlp"
	// _ "github.com/webitel/webitel-go-kit/otel/sdk/trace/stdout"

	"github.com/webitel/webitel-wfm/config"
	"github.com/webitel/webitel-wfm/infra/health"
	"github.com/webitel/webitel-wfm/infra/pubsub"
	"github.com/webitel/webitel-wfm/infra/registry"
	amdiscovery "github.com/webitel/webitel-wfm/infra/registry/authmanager"
	"github.com/webitel/webitel-wfm/infra/registry/provider/consul"
	"github.com/webitel/webitel-wfm/infra/server"
	"github.com/webitel/webitel-wfm/infra/shutdown"
	"github.com/webitel/webitel-wfm/infra/storage/cache"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql/cluster"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql/pg"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql/scanner"
	"github.com/webitel/webitel-wfm/infra/webitel/engine"
	"github.com/webitel/webitel-wfm/infra/webitel/logger"
	"github.com/webitel/webitel-wfm/pkg/endpoint"
	"github.com/webitel/webitel-wfm/pkg/werror"
)

const (
	// sessionCacheSize is the maximum size of sessions to be cached.
	sessionCacheSize = 35000

	// sessionCacheTime is the duration in seconds for which a session will be cached.
	sessionCacheTime = 60 * 5
)

var serviceInstance = &registry.ServiceInstance{
	Name:    "wfm",
	Version: version,
	Metadata: map[string]string{
		"commit":         commit,
		"commitDate":     commitDate,
		"branch":         branch,
		"buildTimestamp": strconv.FormatInt(buildTimestamp, 10),
	},
}

func api(cfg *config.Config, log *wlog.Logger) *cli.Command {
	return &cli.Command{
		Name:    "api",
		Aliases: []string{"a"},
		Usage:   "Start WFM API server",
		Flags:   apiFlags(cfg),
		Action: func(c *cli.Context) error {
			tracker := shutdown.NewTracker(log)

			// Watch for shutdown signals (SIGTERM, SIGINT)
			// and triggers the graceful shutdown when such a signal is received.
			tracker.WatchForShutdownSignals()

			a, err := newApp(c.Context, cfg, log, tracker)
			if err != nil {
				tracker.Shutdown(nil, err)
			}

			// This blocks until an error is produced
			if err := a.run(c.Context); err != nil {
				tracker.Shutdown(nil, err)
			}

			return err
		},
	}
}

func apiFlags(cfg *config.Config) []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "bind-address",
			Category:    "server",
			Usage:       "address that should be bound to for internal cluster communications",
			Value:       "127.0.0.1:10031",
			Destination: &cfg.Service.Address,
			Aliases:     []string{"b"},
			EnvVars:     []string{"BIND_ADDRESS"},
		},
		&cli.StringFlag{
			Name:        "consul-discovery",
			Category:    "service/discovery",
			Usage:       "service discovery address",
			Value:       "127.0.0.1:8500",
			Destination: &cfg.Consul.Address,
			Aliases:     []string{"c"},
			EnvVars:     []string{"MICRO_REGISTRY_ADDRESS"},
		},
		&cli.StringFlag{
			Name:        "pubsub",
			Category:    "service/pubsub",
			Usage:       "publish/subscribe rabbitmq broker connection string",
			Value:       "amqp://webitel:webitel@127.0.0.1:5672/",
			Destination: &cfg.Pubsub.Address,
			Aliases:     []string{"p"},
			EnvVars:     []string{"MICRO_BROKER_ADDRESS"},
		},
		&cli.StringFlag{
			Name:        "forecast-calculation",
			Category:    "storage/database",
			Usage:       "persistent database driver name and a driver-specific data source name for executing forecast calculation queries",
			Value:       "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable",
			Destination: &cfg.Database.ForecastCalculationDSN,
			Aliases:     []string{"fc"},
		},
		&cli.IntFlag{
			Name:        "cache-size",
			Category:    "storage/cache",
			Usage:       "cache capacity in bytes; must be smaller than the available RAM size for the app, since the cache holds data in memory",
			Value:       1024,
			Destination: &cfg.Cache.Size,
			EnvVars:     []string{"CACHE_SIZE"},
		},
		&cli.DurationFlag{
			Name:     "keep-accepting",
			Category: "service/shutdown",
			Usage:    "duration from the moment we receive a SIGTERM after which we stop accepting new requests",
			Value:    0,
		},
		&cli.DurationFlag{
			Name:     "force-close-tasks-grace",
			Category: "service/shutdown",
			Usage:    "duration (measured from when canceling running tasks) after which the tasks are considered done, even if they're still running",
			Value:    1 * time.Second,
		},
		&cli.DurationFlag{
			Name:     "force-shutdown-grace",
			Category: "service/shutdown",
			Usage:    "grace period after beginning the force shutdown before the shutdown is marked as completed, causing the process to exit",
			Value:    1 * time.Second,
		},
	}
}

type app struct {
	cfg *config.Config
	log *wlog.Logger

	shutdown *shutdown.Tracker
	health   *health.CheckRegistry

	resources *resources

	// startedCh closed once the app has finished starting.
	startedCh chan struct{}

	eg *errgroup.Group
}

type resources struct {
	grpcServer *server.Server
	storage    cluster.Store
	cache      cache.Manager
	authcli    authmanager.AuthManager
	engine     *engine.Client
	loggercli  *logger.Client
	audit      *logger.Audit
	registry   *consul.Registry
	ps         *pubsub.Manager
}

//nolint:unused
func (r *resources) registerShutdownAndHealthHooks(tracker *shutdown.Tracker, checker *health.CheckRegistry) error {
	elem := reflect.ValueOf(r).Elem()
	elemType := elem.Type()

	for i := 0; i < elem.NumField(); i++ {
		fieldType := elemType.Field(i).Type
		fieldValue := elem.Field(i)
		fieldName := fieldType.Name()

		if fieldType.Implements(reflect.TypeOf((*shutdown.Handler)(nil)).Elem()) {
			hook, ok := fieldValue.Interface().(shutdown.Handler)
			if ok {
				if err := tracker.RegisterShutdownHandler(fieldName, hook); err != nil {
					return err
				}
			}
		}

		if fieldType.Implements(reflect.TypeOf((*health.Check)(nil)).Elem()) {
			hook, ok := fieldValue.Interface().(health.Check)
			if ok {
				checker.Register(hook)
			}
		}
	}

	return nil
}

func newApp(ctx context.Context, cfg *config.Config, log *wlog.Logger, tracker *shutdown.Tracker) (*app, error) {
	startedCh := make(chan struct{})

	// Notify anyone who might be listening to that the app has finished starting.
	// This can be used by, e.g., app tests.
	defer close(startedCh)

	check := health.NewCheckRegistry(log)
	// service := otelsdk.WithResource(resource.NewSchemaless(semconv.ServiceName(serviceName),
	// 	semconv.ServiceVersion(version),
	// 	semconv.ServiceInstanceID(cfg.Service.NodeID),
	// 	semconv.ServiceNamespace("webitel"),
	// ))
	//
	// shutdownFunc, err := otelsdk.Setup(ctx, service, otelsdk.WithLogLevel(otellog.SeverityDebug))
	// if err != nil {
	// 	return nil, err
	// }

	// if err := tracker.RegisterShutdownHandlerFunc("otel", func(p *shutdown.Process) error { return shutdownFunc(ctx) }); err != nil {
	// 	return nil, err
	// }

	// Initialize all application resources (database, cache, servers, etc...)
	// using generated code by github.com/google/wire.
	res, err := initResources(ctx, cfg, log, check, tracker)
	if err != nil {
		return nil, err
	}

	// Iterates over struct fields to find those which implement
	// shutdown.Handler interface and register shutdown and healthcheck hooks.
	// TODO: reflect.Value.Interface: cannot return value obtained from unexported field or method
	// if err = res.registerShutdownAndHealthHooks(tracker, check); err != nil {
	// 	return nil, err
	// }

	// Initialize database cluster with checks for executing forecast
	// calculation queries.
	fs, err := forecastStorage(ctx, cfg, log, check, tracker)
	if err != nil {
		return nil, err
	}

	// Create handlers for gRPC service using generated code
	// by github.com/google/wire.
	_, err = initHandlers(res, fs)
	if err != nil {
		return nil, err
	}

	return &app{
		cfg:       cfg,
		log:       log,
		health:    health.NewCheckRegistry(log),
		shutdown:  tracker,
		resources: res,
		startedCh: startedCh,
		eg:        &errgroup.Group{},
	}, nil
}

func (a *app) run(ctx context.Context) error {
	// Verify registered health checks.
	var err error
	checks := a.health.RunAll(ctx)
	for _, check := range checks {
		if check.Err != nil {
			a.log.Error("healthcheck was unsuccessful", wlog.String("check", check.Name), wlog.Err(check.Err))
			err = werror.Wrap(err, werror.WithValue(check.Name, check.Err))
		}
	}

	if err != nil {
		return err
	}

	// Start server requests listening, serve all application resources.
	a.eg.Go(func() error {
		return a.resources.authcli.Start()
	})

	a.eg.Go(func() error {
		return a.resources.ps.Start()
	})

	a.eg.Go(func() error {
		l, err := net.Listen("tcp", a.cfg.Service.Address)
		if err != nil {
			return err
		}

		return a.resources.grpcServer.Serve(l)
	})

	a.eg.Go(func() error {
		serviceInstance = &registry.ServiceInstance{
			ID:       a.cfg.Service.NodeID,
			Name:     serviceInstance.Name,
			Version:  serviceInstance.Version,
			Metadata: serviceInstance.Metadata,
			Endpoints: []string{
				endpoint.NewEndpoint("grpc", a.cfg.Service.Address).String(),
			},
		}

		return a.resources.registry.Register(ctx, serviceInstance)
	})

	if err := a.eg.Wait(); err != nil {
		return err
	}

	return nil
}

func serviceDiscovery(ctx context.Context, cfg *config.Config, log *wlog.Logger, health *health.CheckRegistry, tracker *shutdown.Tracker) (*consul.Registry, error) {
	opts := []consul.Option{
		consul.WithHeartbeat(true),
		consul.WithTimeout(time.Second * 30),
	}

	reg, err := consul.New(log, cfg.Consul.Address, opts...)
	if err != nil {
		return nil, err
	}

	f := func(p *shutdown.Process) error {
		return reg.Deregister(ctx, serviceInstance)
	}

	if err := tracker.RegisterShutdownHandlerFunc("consul", f); err != nil {
		return nil, err
	}

	return reg, nil
}

func sqlStorage(ctx context.Context, cfg *config.Config, log *wlog.Logger) (*cluster.Cluster, error) {
	dsns := strings.Fields(cfg.Database.DSN)
	nodes := make([]dbsql.Node, 0, len(dsns))
	for _, dsn := range dsns {
		db, err := pg.New(ctx, log, dsn)
		if err != nil {
			return nil, err
		}

		nodes = append(nodes, dbsql.New(dsn, db, scanner.MustNewDBScan()))
	}

	conn, err := cluster.New(log, nodes, cluster.WithUpdate())
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func forecastStorage(ctx context.Context, cfg *config.Config, log *wlog.Logger, health *health.CheckRegistry, tracker *shutdown.Tracker) (*cluster.Cluster, error) {
	const scope = "forecast-sql-storage"
	db, err := pg.New(ctx, log, cfg.Database.ForecastCalculationDSN)
	if err != nil {
		return nil, err
	}

	conn, err := cluster.New(log, []dbsql.Node{dbsql.New(cfg.Database.ForecastCalculationDSN, db, scanner.MustNewDBScan())})
	if err != nil {
		return nil, err
	}

	if err := tracker.RegisterShutdownHandler(scope, conn); err != nil {
		return nil, err
	}

	health.Register(conn)

	return conn, nil
}

func auth(discovery registry.Discovery, health *health.CheckRegistry, tracker *shutdown.Tracker) (authmanager.AuthManager, error) {
	const scope = "webitel-auth"
	conn := authmanager.NewAuthManager(sessionCacheSize, sessionCacheTime, amdiscovery.New(discovery))
	shutdownFunc := func(p *shutdown.Process) error {
		conn.Stop()

		return nil
	}

	if err := tracker.RegisterShutdownHandlerFunc(scope, shutdownFunc); err != nil {
		return nil, err
	}

	health.RegisterFunc(scope, func(ctx context.Context) error {
		return nil
	})

	return conn, nil
}
