package cmd

import (
	"context"
	"fmt"

	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/database"
	"github.com/urfave/cli/v2"
	"github.com/webitel/webitel-go-kit/logging/wlog"

	"github.com/webitel/webitel-wfm/config"
	"github.com/webitel/webitel-wfm/infra/health"
	"github.com/webitel/webitel-wfm/infra/shutdown"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql/cluster"
	"github.com/webitel/webitel-wfm/migrations"
)

func migrate(cfg *config.Config, log *wlog.Logger) *cli.Command {
	return &cli.Command{
		Name:    "migrate",
		Aliases: []string{"m"},
		Usage:   "Execute database migrations",
		Action: func(c *cli.Context) error {
			m := newMigrator(cfg, log)
			m.shutdown.WatchForShutdownSignals()

			if err := m.run(c.Context); err != nil {
				m.shutdown.Shutdown(nil, err)

			}

			<-m.doneCh
			m.shutdown.Shutdown(nil, nil)

			return nil
		},
	}
}

type migrator struct {
	cfg *config.Config
	log *wlog.Logger

	health   *health.CheckRegistry
	shutdown *shutdown.Tracker

	dbsql cluster.Cluster

	doneCh chan struct{}
}

func newMigrator(cfg *config.Config, log *wlog.Logger) *migrator {
	return &migrator{
		cfg:      cfg,
		log:      log,
		shutdown: shutdown.NewTracker(log),
		health:   health.NewCheckRegistry(log),
		doneCh:   make(chan struct{}),
	}
}

func (m *migrator) run(ctx context.Context) error {
	defer close(m.doneCh)
	cl, err := sqlStorage(ctx, m.cfg, m.log, m.health, m.shutdown)
	if err != nil {
		return err
	}

	goose.SetLogger(newLogger(m.log))
	goose.SetVerbose(true)
	store, err := database.NewStore(database.DialectPostgres, "wfm_schema_version")
	if err != nil {
		return err
	}

	noopDialect := goose.Dialect("")
	provider, err := goose.NewProvider(noopDialect, cl.Primary().Stdlib(), migrations.Embed, goose.WithStore(store))
	if err != nil {
		return err
	}

	res, err := provider.Up(ctx)
	if err != nil {
		return err
	}

	for i, r := range res {
		fields := []wlog.Field{
			wlog.Int("num", i),
			wlog.Duration("elapsed", r.Duration),
			wlog.String("direction", r.Direction),
			wlog.Any("empty", r.Empty),
			wlog.String("path", r.Source.Path),
			wlog.Int64("version", r.Source.Version),
			wlog.String("type", string(r.Source.Type)),
		}

		log := m.log.With(fields...)
		if r.Error != nil {
			log.Error("unable to apply migration", wlog.Err(r.Error))
		} else {
			log.Info("applied migration")
		}
	}

	return nil
}

type migrateLogger struct {
	log *wlog.Logger
}

func newLogger(log *wlog.Logger) *migrateLogger {
	return &migrateLogger{log: log}
}

func (l *migrateLogger) Printf(format string, args ...interface{}) {
	l.log.Info(fmt.Sprintf(format, args...))
}

func (l *migrateLogger) Fatalf(format string, args ...interface{}) {
	l.log.Error(fmt.Sprintf(format, args...))
}
