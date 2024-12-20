package infra

import (
	"github.com/google/wire"

	"github.com/webitel/webitel-wfm/infra/pubsub"
	"github.com/webitel/webitel-wfm/infra/server"
	"github.com/webitel/webitel-wfm/infra/storage/cache"
	"github.com/webitel/webitel-wfm/infra/webitel/engine"
	"github.com/webitel/webitel-wfm/infra/webitel/logger"
)

var Set = wire.NewSet(cache.New, wire.Bind(new(cache.Manager), new(*cache.Cache)),
	server.New, engine.New, pubsub.New, logger.New, logger.NewAudit)
