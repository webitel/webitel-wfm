package direct

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/webitel/webitel-wfm/infra/registry"
)

const (
	defaultWeight = 100
)

var (
	_ registry.WeightedNode        = (*Node)(nil)
	_ registry.WeightedNodeBuilder = (*Builder)(nil)
)

// Node is endpoint instance
type Node struct {
	registry.Node

	// last lastPick timestamp
	lastPick int64
}

// Builder is direct node builder
type Builder struct{}

// Build create node
func (*Builder) Build(n registry.Node) registry.WeightedNode {
	return &Node{Node: n, lastPick: 0}
}

func (n *Node) Pick() registry.DoneFunc {
	now := time.Now().UnixNano()
	atomic.StoreInt64(&n.lastPick, now)

	return func(context.Context, registry.DoneInfo) {}
}

// Weight is node effective weight
func (n *Node) Weight() float64 {
	if n.InitialWeight() != nil {
		return float64(*n.InitialWeight())
	}

	return defaultWeight
}

func (n *Node) PickElapsed() time.Duration {
	return time.Duration(time.Now().UnixNano() - atomic.LoadInt64(&n.lastPick))
}

func (n *Node) Raw() registry.Node {
	return n.Node
}
