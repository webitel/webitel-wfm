package dbsql

import (
	"time"

	"github.com/urfave/cli/v2"

	"github.com/webitel/webitel-wfm/config"
)

func Flags(cfg *config.Config) []cli.Flag {
	const category = "storage/database"

	return []cli.Flag{
		&cli.StringFlag{
			Name:        "db-dsn",
			Category:    category,
			Usage:       "persistent database driver name and a driver-specific data source name",
			EnvVars:     []string{"WEBITEL_DBO_ADDRESS"},
			Value:       "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable",
			Destination: &cfg.Database.DSN,
		},
		&cli.DurationFlag{
			Name:     "db-conn-ttl",
			Category: category,
			Usage:    "set the maximum amount of time a connection may be reused",
			EnvVars:  []string{"WEBITEL_DBO_CONN_TTL"},
			Value:    time.Second * 30,
		},
		&cli.IntFlag{
			Name:     "db-conn-max",
			Category: category,
			Usage:    "set the maximum number of open connections to the database",
			EnvVars:  []string{"WEBITEL_DBO_CONN_MAX"},
			Value:    3,
		},
		&cli.IntFlag{
			Name:     "db-idle-max",
			Category: category,
			Usage:    "set the maximum number of connections in the idle connection pool",
			EnvVars:  []string{"WEBITEL_DBO_IDLE_MAX"},
			Value:    3,
		},
	}
}
