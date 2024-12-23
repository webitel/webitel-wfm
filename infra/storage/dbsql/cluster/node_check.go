package cluster

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/webitel/webitel-wfm/infra/storage/dbsql"
)

// Checker is a signature for functions that check if a specific node is alive and is primary.
// Returns true for primary and false if not.
// If error is returned, the node is considered dead.
// Check function can be used to perform a Query returning single boolean value that signals
// if node is primary or not.
type Checker func(ctx context.Context, db dbsql.Node) (bool, error)

type checkedNode struct {
	Node    dbsql.Node
	Latency time.Duration
}

type checkedNodesList []checkedNode

var _ sort.Interface = checkedNodesList{}

func (list checkedNodesList) Len() int {
	return len(list)
}

func (list checkedNodesList) Less(i, j int) bool {
	return list[i].Latency < list[j].Latency
}

func (list checkedNodesList) Swap(i, j int) {
	list[i], list[j] = list[j], list[i]
}

func (list checkedNodesList) Nodes() []dbsql.Node {
	res := make([]dbsql.Node, 0, len(list))
	for _, item := range list {
		res = append(res, item.Node)
	}

	return res
}

type groupedCheckedNodes struct {
	Primaries checkedNodesList
	Standbys  checkedNodesList
}

// Alive returns merged primaries and standbys sorted by latency. Primaries and standbys are expected to be
// sorted beforehand.
func (nodes groupedCheckedNodes) Alive() []dbsql.Node {
	res := make([]dbsql.Node, len(nodes.Primaries)+len(nodes.Standbys))

	var i int
	for len(nodes.Primaries) > 0 && len(nodes.Standbys) > 0 {
		if nodes.Primaries[0].Latency < nodes.Standbys[0].Latency {
			res[i] = nodes.Primaries[0].Node
			nodes.Primaries = nodes.Primaries[1:]
		} else {
			res[i] = nodes.Standbys[0].Node
			nodes.Standbys = nodes.Standbys[1:]
		}

		i++
	}

	for j := 0; j < len(nodes.Primaries); j++ {
		res[i] = nodes.Primaries[j].Node
		i++
	}

	for j := 0; j < len(nodes.Standbys); j++ {
		res[i] = nodes.Standbys[j].Node
		i++
	}

	return res
}

type checkExecutorFunc func(ctx context.Context, node dbsql.Node) (bool, time.Duration, error)

// checkNodes takes slice of nodes, checks them in parallel and returns the alive ones.
// Accepts customizable executor which enables time-independent tests for node sorting based on 'latency'.
func checkNodes(ctx context.Context, nodes []dbsql.Node, executor checkExecutorFunc, tracer Tracer, errCollector *errorsCollector) AliveNodes {
	checkedNodes := groupedCheckedNodes{
		Primaries: make(checkedNodesList, 0, len(nodes)),
		Standbys:  make(checkedNodesList, 0, len(nodes)),
	}

	var mu sync.Mutex
	var wg sync.WaitGroup
	wg.Add(len(nodes))
	for _, item := range nodes {
		go func(n dbsql.Node, wg *sync.WaitGroup) {
			defer wg.Done()

			primary, duration, err := executor(ctx, n)
			if err != nil {
				n.SetState(dbsql.Dead)

				if tracer.NodeDead != nil {
					tracer.NodeDead(n, err)
				}

				if errCollector != nil {
					errCollector.Add(n.Addr(), err, time.Now())
				}

				return
			}

			if errCollector != nil {
				errCollector.Remove(n.Addr())
			}

			if ok := n.CompareState(dbsql.Alive); !ok {
				if tracer.NodeAlive != nil {
					tracer.NodeAlive(n)
				}
			}

			n.SetState(dbsql.Alive)

			nl := checkedNode{Node: n, Latency: duration}

			mu.Lock()
			defer mu.Unlock()
			if primary {
				checkedNodes.Primaries = append(checkedNodes.Primaries, nl)
			} else {
				checkedNodes.Standbys = append(checkedNodes.Standbys, nl)
			}
		}(item, &wg)
	}
	wg.Wait()

	sort.Sort(checkedNodes.Primaries)
	sort.Sort(checkedNodes.Standbys)

	return AliveNodes{
		Alive:     checkedNodes.Alive(),
		Primaries: checkedNodes.Primaries.Nodes(),
		Standbys:  checkedNodes.Standbys.Nodes(),
	}
}

// checkExecutor returns checkExecutorFunc which can execute supplied check.
func checkExecutor(checker Checker) checkExecutorFunc {
	return func(ctx context.Context, node dbsql.Node) (bool, time.Duration, error) {
		ts := time.Now()
		primary, err := checker(ctx, node)
		d := time.Since(ts)
		if err != nil {
			return false, d, err
		}

		return primary, d, nil
	}
}
