package cluster

import (
	"math/rand/v2"
	"sync/atomic"
	"time"
)

// NodePicker decides which node must be used from a given set.
// It also provides an comparer to be used to pre-sort nodes for better performance.
type NodePicker interface {

	// PickNode returns a single node from a given set.
	PickNode(nodes []CheckedNode) CheckedNode

	// CompareNodes is a comparison function to be used to sort checked nodes.
	CompareNodes(a, b CheckedNode) int
}

// RandomNodePicker returns random node on each call and does not sort checked nodes.
type RandomNodePicker struct{}

// PickNode returns random node from picker
func (*RandomNodePicker) PickNode(nodes []CheckedNode) CheckedNode {
	return nodes[rand.IntN(len(nodes))]
}

// CompareNodes always treats nodes as equal, effectively not changing nodes order.
func (*RandomNodePicker) CompareNodes(_, _ CheckedNode) int {
	return 0
}

// RoundRobinNodePicker returns the next node based on Round Robin algorithm
// and tries to preserve nodes order across checks.
type RoundRobinNodePicker struct {
	idx uint32
}

// PickNode returns next node in Round-Robin sequence.
func (r *RoundRobinNodePicker) PickNode(nodes []CheckedNode) CheckedNode {
	n := atomic.AddUint32(&r.idx, 1)

	return nodes[(int(n)-1)%len(nodes)]
}

// CompareNodes performs lexicographical comparison of two nodes.
func (r *RoundRobinNodePicker) CompareNodes(a, b CheckedNode) int {
	aName, bName := a.Node.Addr(), b.Node.Addr()
	if aName < bName {
		return -1
	}

	if aName > bName {
		return 1
	}

	return 0
}

// LatencyNodePicker returns node with the least latency and
// sorts checked nodes by reported latency ascending.
//
// WARNING: This picker requires that NodeInfoProvider can report
// node's network latency otherwise code will panic!
type LatencyNodePicker struct{}

// PickNode returns node with the least network latency.
func (*LatencyNodePicker) PickNode(nodes []CheckedNode) CheckedNode {
	return nodes[0]
}

// CompareNodes performs nodes comparison based on reported network latency
func (*LatencyNodePicker) CompareNodes(a, b CheckedNode) int {
	aLatency := a.Info.(interface{ Latency() time.Duration }).Latency()
	bLatency := b.Info.(interface{ Latency() time.Duration }).Latency()
	if aLatency < bLatency {
		return -1
	}

	if aLatency > bLatency {
		return 1
	}

	return 0
}

// ReplicationNodePicker returns node with the smallest replication lag
// and sorts checked nodes by reported replication lag ascending.
// Note that replication lag reported by checkers can vastly differ
// from the real situation on standby server.
//
// WARNING: This picker requires that NodeInfoProvider can report
// node's replication lag otherwise code will panic!
type ReplicationNodePicker struct{}

// PickNode returns node with the lowest replication lag value.
func (*ReplicationNodePicker) PickNode(nodes []CheckedNode) CheckedNode {
	return nodes[0]
}

// CompareNodes performs nodes comparison based on reported replication lag.
func (*ReplicationNodePicker) CompareNodes(a, b CheckedNode) int {
	aLag := a.Info.(interface{ ReplicationLag() int }).ReplicationLag()
	bLag := b.Info.(interface{ ReplicationLag() int }).ReplicationLag()
	if aLag < bLag {
		return -1
	}

	if aLag > bLag {
		return 1
	}

	return 0
}
