package pubsub

import (
	"context"

	"go.uber.org/fx"

	"github.com/webitel/webitel-go-kit/logging/wlog"

	"github.com/webitel/webitel-wfm/config"
)

var Module = fx.Module("pubsub",
	fx.Provide(ProvidePubSub),
)

func ProvidePubSub(lc fx.Lifecycle, cfg *config.Config, log *wlog.Logger) (*Manager, error) {
	m, err := New(log, cfg.Pubsub.URL)
	if err != nil {
		return nil, err
	}

	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			log.Info("starting pub/sub connection loop")

			return m.Start()
		},
		OnStop: func(context.Context) error {
			m.Stop()

			return nil
		},
	})

	return m, nil
}
