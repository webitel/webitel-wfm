package cluster

import (
	"github.com/webitel/webitel-go-kit/logging/wlog"
)

// Tracer is a set of hooks to be called at various stages of background nodes status update.
// Any particular hook may be nil. Functions may be called concurrently from different goroutines.
type Tracer struct {

	// UpdateNodes is called when before updating nodes status.
	UpdateNodes func()

	// NodesUpdated is called after all nodes are updated. The nodes is a list of currently alive nodes.
	NodesUpdated func(nodes CheckedNodes)

	// NodeDead is called when it is determined that specified node is dead.
	NodeDead func(err error)

	// NodeAlive is called when it is determined that specified node is alive.
	NodeAlive func(node CheckedNode)

	// WaitersNotified is called when callers of 'WaitForNode' function have been notified.
	WaitersNotified func()
}

func DefaultTracer(log *wlog.Logger) Tracer {
	return Tracer{}
}
