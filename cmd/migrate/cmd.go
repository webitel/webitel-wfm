package migrate

import (
	"context"
	"fmt"
	"strings"

	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/database"
	"github.com/urfave/cli/v2"
	"github.com/webitel/webitel-go-kit/logging/wlog"
	"go.uber.org/fx"

	"github.com/webitel/webitel-wfm/cmd/server"
	"github.com/webitel/webitel-wfm/config"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql/cluster"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql/pg"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql/scanner"
	"github.com/webitel/webitel-wfm/migrations"
)

func CMD() *cli.Command {
	return &cli.Command{
		Name:    "migrate",
		Aliases: []string{"m"},
		Usage:   "Execute database migrations",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "config_file",
				Usage: "Path to the configuration file",
			},
		},
		Action: func(c *cli.Context) error {
			cfg, err := config.LoadMigrateConfig()
			if err != nil {
				return err
			}

			app := fx.New(
				fx.Provide(
					func() *config.Config { return cfg },
					server.ProvideLogger,
				),
				fx.Invoke(func(cfg *config.Config, log *wlog.Logger) error {
					return run(c.Context, cfg, log)
				}),
				fx.NopLogger,
			)

			return app.Start(c.Context)
		},
	}
}

func run(ctx context.Context, cfg *config.Config, log *wlog.Logger) error {
	nodes := make([]dbsql.Node, 0, 1)

	for _, dsn := range strings.Fields(cfg.Postgres.DSN) {
		db, err := pg.New(ctx, log, dsn)
		if err != nil {
			return err
		}

		nodes = append(nodes, dbsql.New(dsn, db, scanner.MustNewDBScan()))
	}

	cl, err := cluster.New(log, nodes, cluster.WithUpdate())
	if err != nil {
		return err
	}
	defer cl.Close()

	goose.SetLogger(newLogger(log))
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

	if len(res) == 0 {
		log.Info("database is up to date")
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

		log := log.With(fields...)
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
