package dbsql

import (
	"math/rand"
	"sync/atomic"
)

// PickNodeRandom returns random node from nodes set
func PickNodeRandom() NodePicker {
	return func(nodes []Node) Node {
		return nodes[rand.Intn(len(nodes))]
	}
}

// PickNodeRoundRobin returns next node based on Round Robin algorithm
func PickNodeRoundRobin() NodePicker {
	var nodeIdx uint32
	return func(nodes []Node) Node {
		n := atomic.AddUint32(&nodeIdx, 1)
		return nodes[(int(n)-1)%len(nodes)]
	}
}

// PickNodeClosest returns node with the least latency
func PickNodeClosest() NodePicker {
	return func(nodes []Node) Node {
		return nodes[0]
	}
}
