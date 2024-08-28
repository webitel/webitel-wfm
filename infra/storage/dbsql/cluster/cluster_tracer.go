package cluster

import "github.com/webitel/webitel-go-kit/logging/wlog"

// Tracer is a set of hooks to run at various stages of background nodes status update.
// Any particular hook may be nil. Functions may be called concurrently from different goroutines.
type Tracer struct {

	// UpdateNodes is called when before updating nodes status.
	UpdateNodes func()

	// UpdatedNodes is called after all nodes are updated. The nodes is a list of currently alive nodes.
	UpdatedNodes func(nodes AliveNodes)

	// NodeDead is called when it is determined that specified node is dead.
	NodeDead func(node Node, err error)

	// NodeAlive is called when it is determined that specified node is alive.
	NodeAlive func(node Node)

	// NotifiedWaiters is called when all callers of 'WaitFor*' functions have been notified.
	NotifiedWaiters func()
}

func DefaultTracer(log *wlog.Logger) Tracer {
	return Tracer{
		UpdateNodes:  func() {},
		UpdatedNodes: func(_ AliveNodes) {},
		NodeDead: func(node Node, err error) {
			log.Warn("node is dead", wlog.Any("node", node), wlog.Err(err))
		},
		NodeAlive: func(node Node) {
			log.Debug("node is alive", wlog.Any("node", node))
		},
	}
}
