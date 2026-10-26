package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/webitel/webitel-wfm/cmd/migrate"
	"github.com/webitel/webitel-wfm/cmd/server"
	"github.com/webitel/webitel-wfm/internal/model"
)

// CI stamps the build into these with -ldflags "-X github.com/webitel/webitel-wfm/cmd.version=…";
// the targets come from webitel/reusable-configs (golang/sync.yml) and are shared by synced workflows.
var (
	version string
	commit  string
	branch  string
)

// Run the default command.
func Run() error {
	model.SetBuild(version, commit, branch)

	app := &cli.App{
		Name:      "webitel-wfm",
		Usage:     "Effective planning of human resources in the Webitel",
		Version:   fmt.Sprintf("%s, %s@%s at %s, %s", model.Version, model.Branch, model.Commit, model.CommitDate, model.BuildTimestamp),
		Compiled:  time.Now(),
		Copyright: "Webitel, 2024",
		Commands: []*cli.Command{
			server.CMD(),
			migrate.CMD(),
		},
	}

	return app.Run(os.Args)
}
