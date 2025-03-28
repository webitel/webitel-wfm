package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/urfave/cli/v2"
	"github.com/webitel/webitel-go-kit/logging/wlog"

	"github.com/webitel/webitel-wfm/config"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql"
)

var (

	// version is the APP's semantic version.
	version = "0.0.0"

	// commit is the git commit used to build the api.
	commit     = "hash"
	commitDate = time.Now().String()

	branch         = "branch"
	buildTimestamp = ""
)

// Run the default command
func Run() error {
	cfg := config.New()
	log := wlog.NewLogger(&wlog.LoggerConfiguration{
		EnableConsole: true,
		ConsoleLevel:  wlog.LevelDebug,
		EnableExport:  true,
	})

	wlog.InitGlobalLogger(log.With(wlog.String("source", "global")))

	def := &cli.App{
		Name:      "webitel-wfm",
		Usage:     "Effective planning of human resources in the Webitel",
		Version:   fmt.Sprintf("%s, %s@%s at %s, %d", version, branch, commit, commitDate, buildTimestamp),
		Compiled:  time.Now(),
		Copyright: "Webitel, 2024",
		Action: func(c *cli.Context) error {
			return nil
		},
		Commands: []*cli.Command{api(cfg, log), migrate(cfg, log)},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "log-level",
				Category:    "observability/logging",
				Usage:       "application log level",
				Value:       "debug",
				Destination: &cfg.Logger.Level,
				Aliases:     []string{"l"},
				EnvVars:     []string{"MICRO_LOG_LEVEL"},
			},
		},
	}

	def.Flags = append(def.Flags, dbsql.Flags(cfg)...)
	if err := def.Run(os.Args); err != nil {
		return err
	}

	return nil
}
