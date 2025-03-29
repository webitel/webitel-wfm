package cluster

import (
	"context"
	"errors"
	"slices"
	"sync"

	"github.com/webitel/webitel-wfm/infra/storage/dbsql"
)

// NodeStateCriterion represents a node selection criterion.
type NodeStateCriterion uint8

const (

	// Alive is a criterion to choose any alive node.
	Alive NodeStateCriterion = iota + 1

	// Primary is a criterion to choose primary node.
	Primary

	// Standby is a criterion to choose standby node.
	Standby

	// PreferPrimary is a criterion to choose primary or any alive node.
	PreferPrimary

	// PreferStandby is a criterion to choose standby or any alive node.
	PreferStandby

	// maxNodeCriterion is for testing purposes only
	// all new criteria must be added above this constant.
	maxNodeCriterion //nolint:unused
)

// CheckedNodes holds references to all available cluster nodes.
type CheckedNodes struct {
	discovered []dbsql.Node
	alive      []CheckedNode
	primaries  []CheckedNode
	standbys   []CheckedNode
	err        error
}

// Discovered returns a list of nodes discovered in cluster.
func (c CheckedNodes) Discovered() []dbsql.Node {
	return c.discovered
}

// Alive returns a list of all successfully checked nodes rewarding their cluster role.
func (c CheckedNodes) Alive() []CheckedNode {
	return c.alive
}

// Primaries returns list of all successfully checked nodes with a primary role.
func (c CheckedNodes) Primaries() []CheckedNode {
	return c.primaries
}

// Standbys returns list of all successfully checked nodes with a standby role.
func (c CheckedNodes) Standbys() []CheckedNode {
	return c.standbys
}

// Err holds information about cause of node check failure.
func (c CheckedNodes) Err() error {
	return c.err
}

// CheckedNode contains the most recent state of single cluster node.
type CheckedNode struct {
	Node dbsql.Node
	Info NodeInfoProvider
}

// checkNodes takes slice of nodes, checks them in parallel and returns the alive ones.
func checkNodes(ctx context.Context, nodes []dbsql.Node, checkFn NodeChecker, compareFn func(a, b CheckedNode) int, tracer Tracer) CheckedNodes {
	var (
		mu   sync.Mutex
		errs NodeCheckErrors
		wg   sync.WaitGroup
	)

	checked := make([]CheckedNode, 0, len(nodes))
	wg.Add(len(nodes))
	for _, node := range nodes {
		go func(node dbsql.Node) {
			defer wg.Done()

			info, err := checkFn(ctx, node)
			if err != nil {
				cerr := NodeCheckError{
					node: node,
					err:  err,
				}

				if tracer.NodeDead != nil {
					tracer.NodeDead(cerr)
				}

				mu.Lock()
				defer mu.Unlock()
				errs = append(errs, cerr)
				return
			}

			cn := CheckedNode{
				Node: node,
				Info: info,
			}

			if !node.CompareState(dbsql.Alive) {
				if tracer.NodeAlive != nil {
					tracer.NodeAlive(cn)
				}
			}

			node.SetState(dbsql.Alive)
			mu.Lock()
			defer mu.Unlock()
			checked = append(checked, cn)
		}(node)
	}

	wg.Wait()
	slices.SortFunc(checked, compareFn)

	alive := make([]CheckedNode, 0, len(checked))

	// in almost all cases there is only one primary node in cluster
	primaries := make([]CheckedNode, 0, 1)
	standbys := make([]CheckedNode, 0, len(checked))
	for _, cn := range checked {
		switch cn.Info.Role() {
		case NodeRolePrimary:
			primaries = append(primaries, cn)
			alive = append(alive, cn)
		case NodeRoleStandby:
			standbys = append(standbys, cn)
			alive = append(alive, cn)
		default:
			// treat node with undetermined role as dead
			cerr := NodeCheckError{
				node: cn.Node,
				err:  errors.New("cannot determine node role"),
			}

			errs = append(errs, cerr)
			if tracer.NodeDead != nil {
				tracer.NodeDead(cerr)
			}
		}
	}

	res := CheckedNodes{
		discovered: nodes,
		alive:      alive,
		primaries:  primaries,
		standbys:   standbys,
		err: func() error {
			if len(errs) != 0 {
				return errs
			}

			return nil
		}(),
	}

	return res
}

// pickNodeByCriterion is a helper function to pick a single node by given criterion.
func pickNodeByCriterion(nodes CheckedNodes, picker NodePicker, criterion NodeStateCriterion) dbsql.Node {
	var subset []CheckedNode

	switch criterion {
	case Alive:
		subset = nodes.alive
	case Primary:
		subset = nodes.primaries
	case Standby:
		subset = nodes.standbys
	case PreferPrimary:
		if subset = nodes.primaries; len(subset) == 0 {
			subset = nodes.standbys
		}
	case PreferStandby:
		if subset = nodes.standbys; len(subset) == 0 {
			subset = nodes.primaries
		}
	}

	if len(subset) == 0 {
		return nil
	}

	return picker.PickNode(subset).Node
}
