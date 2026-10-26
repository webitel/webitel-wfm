package migrate

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/database"
	"github.com/urfave/cli/v2"
	"go.uber.org/fx"

	"github.com/webitel/webitel-wfm/cmd/server"
	"github.com/webitel/webitel-wfm/config"
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
				fx.Invoke(func(cfg *config.Config, log *slog.Logger) error {
					return run(c.Context, cfg, log)
				}),
				fx.NopLogger,
			)

			if err := app.Start(c.Context); err != nil {
				return err
			}

			// Stop flushes whatever the otel log bridge has buffered; without
			// it a failed migration is never reported to the collector.
			return app.Stop(c.Context)
		},
	}
}

func run(ctx context.Context, cfg *config.Config, log *slog.Logger) error {
	dsns := strings.Fields(cfg.Postgres.DSN)
	if len(dsns) == 0 {
		return errors.New("postgres dsn is required")
	}

	conf, err := pgxpool.ParseConfig(dsns[0])
	if err != nil {
		return err
	}

	db := stdlib.OpenDB(*conf.ConnConfig)
	defer db.Close()

	goose.SetLogger(newLogger(log))
	goose.SetVerbose(true)
	store, err := database.NewStore(database.DialectPostgres, "wfm_schema_version")
	if err != nil {
		return err
	}

	noopDialect := goose.Dialect("")
	provider, err := goose.NewProvider(noopDialect, db, migrations.Embed, goose.WithStore(store))
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
		fields := []any{
			slog.Int("num", i),
			slog.Duration("elapsed", r.Duration),
			slog.String("direction", r.Direction),
			slog.Any("empty", r.Empty),
			slog.String("path", r.Source.Path),
			slog.Int64("version", r.Source.Version),
			slog.String("type", string(r.Source.Type)),
		}

		log := log.With(fields...)
		if r.Error != nil {
			log.Error("unable to apply migration", slog.Any("error", r.Error))
		} else {
			log.Info("applied migration")
		}
	}

	return nil
}

type migrateLogger struct {
	log *slog.Logger
}

func newLogger(log *slog.Logger) *migrateLogger {
	return &migrateLogger{log: log}
}

func (l *migrateLogger) Printf(format string, args ...interface{}) {
	l.log.Info(fmt.Sprintf(format, args...))
}

func (l *migrateLogger) Fatalf(format string, args ...interface{}) {
	l.log.Error(fmt.Sprintf(format, args...))
}
