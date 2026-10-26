package server

import (
	"context"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/webitel/webitel-wfm/config"
)

const stopTimeout = 30 * time.Second

func CMD() *cli.Command {
	return &cli.Command{
		Name:    "server",
		Aliases: []string{"s", "api"},
		Usage:   "Start WFM API server",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "config_file",
				Usage: "Path to the configuration file",
			},
		},
		Action: func(c *cli.Context) error {
			cfg, err := config.LoadServerConfig()
			if err != nil {
				return err
			}

			app := NewApp(cfg)
			if err := app.Start(c.Context); err != nil {
				return err
			}

			<-app.Done()

			ctx, cancel := context.WithTimeout(context.Background(), stopTimeout)
			defer cancel()

			return app.Stop(ctx)
		},
	}
}
