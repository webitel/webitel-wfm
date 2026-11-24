package server

import (
	"go.uber.org/fx"

	"github.com/webitel/webitel-wfm/config"
	"github.com/webitel/webitel-wfm/infra/leader"
	grpcsrv "github.com/webitel/webitel-wfm/infra/server/grpc"
	"github.com/webitel/webitel-wfm/infra/webitel/logger"
	"github.com/webitel/webitel-wfm/internal/handler"
	"github.com/webitel/webitel-wfm/internal/service"
	"github.com/webitel/webitel-wfm/internal/storage"
)

func NewApp(cfg *config.Config) *fx.App {
	return fx.New(MainModule(cfg), fx.StopTimeout(stopTimeout))
}

func MainModule(cfg *config.Config) fx.Option {
	return fx.Options(
		fx.Provide(
			func() *config.Config { return cfg },
			ProvideLogger,
			ProvideHealth,
			ProvideProbes,
			ProvideDiscovery,
			ProvideLeaderElector,
			fx.Annotate(leader.New, fx.ParamTags(``, leader.TaskGroup)),
			ProvideStorage,
			ProvideForecastStorage,
			ProvideAuth,
			ProvideEngine,
			ProvideLoggerClient,
			ProvidePublisher,
			func(c *logger.Client) *logger.ConfigService { return c.ConfigService },
			logger.NewAudit,
		),

		storage.Module,
		service.Module,
		handler.Module,
		grpcsrv.Module,

		// Invoked last so their OnStop hooks run first, in reverse: the leader
		// stops its tasks and releases the key, the node leaves rotation, then
		// stops accepting, and only after that do the dependencies underneath
		// wind down.
		fx.Invoke(registerService),
		fx.Invoke(startHealth),
		fx.Invoke(startLeader),
	)
}
