package cluster

import (
	"math/rand"
	"sync/atomic"

	"github.com/webitel/webitel-wfm/infra/storage/dbsql"
)

// NodeStateCriteria for choosing a node
type NodeStateCriteria int

const (

	// Alive for choosing any alive node
	Alive NodeStateCriteria = iota + 1

	// Primary for choosing primary node
	Primary

	// Standby for choosing standby node
	Standby

	// PreferPrimary for choosing primary or any alive node
	PreferPrimary

	// PreferStandby for choosing standby or any alive node
	PreferStandby
)

// Picker is a signature for functions that determine how to pick single node from set of nodes.
// Nodes passed to the picker function are sorted according to latency (from lowest to greatest).
type Picker func(nodes []dbsql.Node) dbsql.Node

// PickNodeRandom returns random node from nodes set
func PickNodeRandom() Picker {
	return func(nodes []dbsql.Node) dbsql.Node {
		return nodes[rand.Intn(len(nodes))]
	}
}

// PickNodeRoundRobin returns next node based on Round Robin algorithm
func PickNodeRoundRobin() Picker {
	var nodeIdx uint32
	return func(nodes []dbsql.Node) dbsql.Node {
		n := atomic.AddUint32(&nodeIdx, 1)

		return nodes[(int(n)-1)%len(nodes)]
	}
}

// PickNodeClosest returns node with the least latency
func PickNodeClosest() Picker {
	return func(nodes []dbsql.Node) dbsql.Node {
		return nodes[0]
	}
}
