package server

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"go.uber.org/fx/fxtest"

	"github.com/webitel/webitel-go-kit/infra/health"

	"github.com/webitel/webitel-wfm/infra/leader"
)

type standby struct{}

func (standby) Run(ctx context.Context, _ func(context.Context) error, _ func()) { <-ctx.Done() }

func TestLeaderRunnerIsALivenessCheck(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	lc := fxtest.NewLifecycle(t)
	h := health.New(health.DefaultConfig(), log)

	startLeader(h, leader.New(standby{}, nil, log, lc))
	lc.RequireStart()

	if err := h.Start(context.Background()); err != nil {
		t.Fatal(err)
	}

	defer func() { _ = h.Stop(context.Background()) }()

	for _, check := range h.Snapshot().Checks {
		if check.Name == "leader-runner" && check.Group == health.GroupLiveness {
			lc.RequireStop()

			return
		}
	}

	t.Fatal("no leader-runner liveness check in the registry")
}
