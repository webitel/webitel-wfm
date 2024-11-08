package cmd

import (
	"context"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/urfave/cli/v2"
	authmanager "github.com/webitel/engine/auth_manager"
	"github.com/webitel/engine/discovery"
	"github.com/webitel/webitel-go-kit/logging/wlog"
	"golang.org/x/sync/errgroup"

	// _ "github.com/webitel/webitel-go-kit/otel/sdk/log/otlp"
	// _ "github.com/webitel/webitel-go-kit/otel/sdk/log/stdout"
	// _ "github.com/webitel/webitel-go-kit/otel/sdk/metric/otlp"
	// _ "github.com/webitel/webitel-go-kit/otel/sdk/metric/stdout"
	// _ "github.com/webitel/webitel-go-kit/otel/sdk/trace/otlp"
	// _ "github.com/webitel/webitel-go-kit/otel/sdk/trace/stdout"

	"github.com/webitel/webitel-wfm/config"
	pb "github.com/webitel/webitel-wfm/gen/go/api/wfm"
	"github.com/webitel/webitel-wfm/infra/health"
	"github.com/webitel/webitel-wfm/infra/pubsub"
	"github.com/webitel/webitel-wfm/infra/server"
	"github.com/webitel/webitel-wfm/infra/shutdown"
	"github.com/webitel/webitel-wfm/infra/storage/cache"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql/cluster"
	"github.com/webitel/webitel-wfm/infra/webitel/engine"
	"github.com/webitel/webitel-wfm/infra/webitel/logger"
	"github.com/webitel/webitel-wfm/internal/handler"
)

const (
	serviceName                  = "wfm"
	serviceTTL                   = time.Second * 30
	serviceDeregisterCriticalTTL = time.Minute * 2

	// sessionCacheSize is the maximum size of sessions to be cached.
	sessionCacheSize = 35000

	// sessionCacheTime is the duration in seconds for which a session will be cached.
	sessionCacheTime = 60 * 5
)

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
		&cli.Int64Flag{
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

	grpcServer *server.Server
	resources  *resources

	// startedCh closed once the app has finished starting.
	startedCh chan struct{}

	eg *errgroup.Group
}

type resources struct {
	storage   cluster.Store
	cache     cache.Manager
	authcli   authmanager.AuthManager
	engine    *engine.Client
	loggercli *logger.Client
	audit     *logger.Audit
	sd        discovery.ServiceDiscovery
	ps        *pubsub.Manager
}

type handlers struct {
	pauseTemplate          *handler.PauseTemplate
	shiftTemplate          *handler.ShiftTemplate
	workingCondition       *handler.WorkingCondition
	agentWorkingConditions *handler.AgentWorkingConditions
	agentAbsence           *handler.AgentAbsence
	forecastCalculation    *handler.ForecastCalculation
	workingSchedule        *handler.WorkingSchedule
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

	// Initialize database cluster with checks for executing forecast
	// calculation queries.
	fs, err := forecastStorage(ctx, cfg, log, check, tracker)
	if err != nil {
		return nil, err
	}

	// Create handlers for gRPC service using generated code
	// by github.com/google/wire.
	h, err := initHandlers(log, res, fs)
	if err != nil {
		return nil, err
	}

	// Create gRPC server and register handlers.
	grpcServer, err := rpcServer(log, cfg, h, res.authcli, tracker)
	if err != nil {
		return nil, err
	}

	return &app{
		cfg:        cfg,
		log:        log,
		health:     health.NewCheckRegistry(log),
		shutdown:   tracker,
		grpcServer: grpcServer,
		resources:  res,
		startedCh:  startedCh,
		eg:         &errgroup.Group{},
	}, nil
}

func (a *app) run(ctx context.Context) error {
	// Verify registered health checks.
	// success := true
	checks := a.health.RunAll(ctx)
	for _, check := range checks {
		if check.Err != nil {
			// success = false
			a.log.Error("healthcheck was unsuccessful", wlog.String("check", check.Name), wlog.Err(check.Err))
		}
	}

	// TODO: stop application execution
	// if !success {
	// 	return
	// }

	// Start server requests listening, serve all application resources.
	a.eg.Go(func() error {
		return a.resources.authcli.Start()
	})

	a.eg.Go(func() error {
		if err := a.resources.engine.Conn.Start(); err != nil {
			return err
		}

		return nil
	})

	a.eg.Go(func() error {
		return a.resources.ps.Start()
	})

	a.eg.Go(func() error {
		if err := a.resources.loggercli.Conn.Start(); err != nil {
			return err
		}

		return nil
	})

	a.eg.Go(func() error {
		l, err := net.Listen("tcp", a.cfg.Service.Address)
		if err != nil {
			return err
		}

		return a.grpcServer.Serve(l)
	})

	a.eg.Go(func() error {
		host, port, err := net.SplitHostPort(a.cfg.Service.Address)
		if err != nil {
			return err
		}

		pi, err := strconv.Atoi(port)
		if err != nil {
			return err
		}

		return a.resources.sd.RegisterService(serviceName, host, pi, serviceTTL, serviceDeregisterCriticalTTL)
	})

	if err := a.eg.Wait(); err != nil {
		return err
	}

	return nil
}

func serviceDiscovery(ctx context.Context, cfg *config.Config, health *health.CheckRegistry, tracker *shutdown.Tracker) (discovery.ServiceDiscovery, error) {
	const scope = "consul-service-discovery"
	f := func() (bool, error) {
		// TODO: review consul health checks
		// ch := health.RunAll(ctx)
		// for _, c := range ch {
		// 	if c.Err != nil {
		// 		return false, c.Err
		// 	}
		// }

		return true, nil
	}

	conn, err := discovery.NewServiceDiscovery(cfg.Service.NodeID, cfg.Consul.Address, f)
	if err != nil {
		return nil, err
	}

	shutdownFunc := func(p *shutdown.Process) error {
		conn.Shutdown()
		p.MarkServicesShutdownCompleted(nil)

		return nil
	}

	if err := tracker.RegisterShutdownHandlerFunc(scope, shutdownFunc); err != nil {
		return nil, err
	}

	health.RegisterFunc(scope, func(ctx context.Context) error {
		_, err := conn.GetByName(serviceName)
		if err != nil {
			return err
		}

		return nil
	})

	return conn, nil
}

func inmemoryCache() (cache.Manager, error) {
	conn, err := cache.New(1024)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func sqlStorage(ctx context.Context, cfg *config.Config, log *wlog.Logger, health *health.CheckRegistry, tracker *shutdown.Tracker) (*cluster.Cluster, error) {
	const scope = "sql-storage"
	dsns := strings.Fields(cfg.Database.DSN)
	conns, err := dbsql.NewConnections(ctx, log, dsns...)
	if err != nil {
		return nil, err
	}

	conn, err := cluster.NewCluster(log, conns, cluster.WithUpdate())
	if err != nil {
		return nil, err
	}

	if err := tracker.RegisterShutdownHandler(scope, conn); err != nil {
		return nil, err
	}

	health.Register(conn)

	return conn, nil
}

func forecastStorage(ctx context.Context, cfg *config.Config, log *wlog.Logger, health *health.CheckRegistry, tracker *shutdown.Tracker) (*cluster.Cluster, error) {
	const scope = "forecast-sql-storage"
	conns, err := dbsql.NewConnections(ctx, log, cfg.Database.ForecastCalculationDSN)
	if err != nil {
		return nil, err
	}

	conn, err := cluster.NewCluster(log, conns, cluster.WithUpdate(), cluster.WithForecastScan())
	if err != nil {
		return nil, err
	}

	if err := tracker.RegisterShutdownHandler(scope, conn); err != nil {
		return nil, err
	}

	health.Register(conn)

	return conn, nil
}

func auth(sd discovery.ServiceDiscovery, health *health.CheckRegistry, tracker *shutdown.Tracker) (authmanager.AuthManager, error) {
	const scope = "webitel-auth"
	conn := authmanager.NewAuthManager(sessionCacheSize, sessionCacheTime, sd)
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

func webitelEngine(log *wlog.Logger, sd discovery.ServiceDiscovery, health *health.CheckRegistry, tracker *shutdown.Tracker) (*engine.Client, error) {
	const scope = "webitel-engine"
	conn, err := engine.New(log, sd)
	if err != nil {
		return nil, err
	}

	if err := tracker.RegisterShutdownHandler(scope, conn); err != nil {
		return nil, err
	}

	health.Register(conn)

	return conn, nil
}

func rpcServer(log *wlog.Logger, cfg *config.Config, h *handlers, authcli authmanager.AuthManager, tracker *shutdown.Tracker) (*server.Server, error) {
	const scope = "grpc-server"
	srv, err := server.New(log, authcli)
	if err != nil {
		return nil, err
	}

	if err := tracker.RegisterShutdownHandler(scope, srv); err != nil {
		return nil, err
	}

	// Register gRPC services.
	pb.RegisterPauseTemplateServiceServer(srv, h.pauseTemplate)
	pb.RegisterShiftTemplateServiceServer(srv, h.shiftTemplate)
	pb.RegisterWorkingConditionServiceServer(srv, h.workingCondition)
	pb.RegisterAgentWorkingConditionsServiceServer(srv, h.agentWorkingConditions)
	pb.RegisterAgentAbsenceServiceServer(srv, h.agentAbsence)
	pb.RegisterForecastCalculationServiceServer(srv, h.forecastCalculation)
	pb.RegisterWorkingScheduleServiceServer(srv, h.workingSchedule)

	return srv, nil
}

func pubsubConn(log *wlog.Logger, cfg *config.Config, tracker *shutdown.Tracker) (*pubsub.Manager, error) {
	ps, err := pubsub.New(log, cfg.Pubsub.Address)
	if err != nil {
		return nil, err
	}

	if err := tracker.RegisterShutdownHandler("pubsub", ps); err != nil {
		return nil, err
	}

	return ps, nil
}

func webitelLogger(log *wlog.Logger, sd discovery.ServiceDiscovery) (*logger.Client, error) {
	return logger.New(log, sd)
}

func audit(svc *logger.Client, pub *pubsub.Manager) *logger.Audit {
	return logger.NewAudit(svc.ConfigService, pub)
}
