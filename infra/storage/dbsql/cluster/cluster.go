package cluster

import (
	"context"
	"errors"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/webitel/webitel-go-kit/logging/wlog"

	"github.com/webitel/webitel-wfm/infra/health"
	"github.com/webitel/webitel-wfm/infra/shutdown"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql"
)

// Default values for Cluster config.
const (
	DefaultUpdateInterval = time.Second * 15
	DefaultUpdateTimeout  = time.Second * 11
)

// Store represents a store that manages a Cluster of nodes.
// It provides methods for retrieving information about the nodes in the Cluster,
// as well as closing the store and checking for any errors.
type Store interface {
	Close() error
	Err() error

	Alive() dbsql.Node
	Primary() dbsql.Node
	Standby() dbsql.Node
	StandbyPreferred() dbsql.Node

	WaitForPrimary(ctx context.Context) (dbsql.Node, error)
	WaitForStandby(ctx context.Context) (dbsql.Node, error)
	WaitForPrimaryPreferred(ctx context.Context) (dbsql.Node, error)
	WaitForStandbyPreferred(ctx context.Context) (dbsql.Node, error)
	WaitForAlive(ctx context.Context) (dbsql.Node, error)
}

// ForecastStore is a type alias for plumbing it through Wire.
type ForecastStore Store

// Cluster consists of number of 'nodes' of a single SQL database.
// Background goroutine periodically checks nodes and updates their status.
type Cluster struct {
	update         bool
	updateInterval time.Duration
	updateTimeout  time.Duration
	checker        NodeChecker
	picker         NodePicker
	tracer         Tracer

	nodes        []dbsql.Node
	checkedNodes atomic.Value
	stop         context.CancelFunc

	subscribersMu sync.Mutex
	subscribers   []updateSubscriber
}

// New constructs Cluster object representing a single 'Cluster' of SQL database.
// Close function must be called when a Cluster isn't necessary anymore.
func New(log *wlog.Logger, nodes []dbsql.Node, opts ...Option) (*Cluster, error) {
	if len(nodes) == 0 {
		return nil, errors.New("please provide at least one database node")
	}

	// prepare internal 'stop' context
	ctx, stopFn := context.WithCancel(context.Background())
	cl := &Cluster{
		updateInterval: DefaultUpdateInterval,
		updateTimeout:  DefaultUpdateTimeout,
		checker:        PostgreSQLChecker,
		picker:         new(RandomNodePicker),
		tracer:         DefaultTracer(log),

		nodes: nodes,

		stop: stopFn,
	}

	for _, opt := range opts {
		opt(cl)
	}

	// Store initial nodes state
	cl.checkedNodes.Store(CheckedNodes{})

	if cl.update {
		// Start update routine.
		go cl.backgroundNodesUpdate(ctx)

		// Wait for node update results.
		time.Sleep(1 * time.Second)
	}

	return cl, nil
}

func (cl *Cluster) Shutdown(p *shutdown.Process) error {
	// Wait for all user code to finish before shutting down databases.
	<-p.ServicesShutdownCompleted.Done()
	<-p.OutstandingTasks.Done()

	return cl.Close()
}

// Close databases and stop node updates.
func (cl *Cluster) Close() error {
	cl.stop()

	// close all nodes underlying connection pools
	var err error
	discovered := cl.checkedNodes.Load().(CheckedNodes).discovered
	for _, node := range discovered {
		if closer, ok := any(node).(io.Closer); ok {
			err = errors.Join(err, closer.Close())
		}
	}

	// discard any collected state of nodes
	cl.checkedNodes.Store(CheckedNodes{})

	return nil
}

func (cl *Cluster) HealthCheck(ctx context.Context) []health.CheckResult {
	var reportError error
	if n := cl.Primary(); n == nil {
		reportError = errors.New("primary database not alive")
	}

	return []health.CheckResult{{
		Name: "primary-database",
		Err:  reportError,
	}}
}

// Err returns cause of nodes most recent check failures.
// In most cases error is a list of errors of type CheckNodeErrors, original errors
// could be extracted using `errors.As`.
// Example:
//
//	var cerrs NodeCheckErrors
//	if errors.As(err, &cerrs) {
//	    for _, cerr := range cerrs {
//	        fmt.Printf("node: %s, err: %s\n", cerr.Node(), cerr.Err())
//	    }
//	}
func (cl *Cluster) Err() error {
	return cl.checkedNodes.Load().(CheckedNodes).Err()
}

// Node returns cluster node with specified status.
func (cl *Cluster) Node(criterion NodeStateCriterion) dbsql.Node {
	node := pickNodeByCriterion(cl.checkedNodes.Load().(CheckedNodes), cl.picker, criterion)
	if node == nil {
		return dbsql.NoopNode{
			Err: cl.Err(),
		}
	}

	return node
}

// Alive returns node that is considered alive.
func (cl *Cluster) Alive() dbsql.Node {
	return cl.Node(Alive)
}

// Primary returns first available node that is considered alive and is primary (able to execute write operations).
func (cl *Cluster) Primary() dbsql.Node {
	return cl.Node(Primary)
}

// Standby returns node that is considered alive and is standby (unable to execute write operations).
func (cl *Cluster) Standby() dbsql.Node {
	return cl.Node(Standby)
}

// PrimaryPreferred returns primary node if possible, standby otherwise.
func (cl *Cluster) PrimaryPreferred() dbsql.Node {
	return cl.Node(PreferPrimary)
}

// StandbyPreferred returns standby node if possible, primary otherwise.
func (cl *Cluster) StandbyPreferred() dbsql.Node {
	return cl.Node(PreferStandby)
}

// WaitForNode with specified status to appear or until context is canceled.
func (cl *Cluster) WaitForNode(ctx context.Context, criterion NodeStateCriterion) (dbsql.Node, error) {
	// Node already exists?
	node := cl.Node(criterion)
	if node != nil {
		return node, nil
	}

	ch := cl.addUpdateSubscriber(criterion)

	// Node might have appeared while we were adding waiter, recheck
	node = cl.Node(criterion)
	if node != nil {
		return node, nil
	}

	// If a channel is unbuffered, and we are right here when nodes are updated,
	// the update code won't be able to write into a channel and will 'forget' it.
	// Then we will report nil to the caller, either because update code
	//  closes the channel or because context is canceled.
	//
	// In both cases its not what user wants.
	//
	// We can solve it by doing cl.Node(ns) if/when we are about to return nil.
	// But if another update runs between channel read and cl.Node(ns) AND no
	// nodes have requested status, we will still return nil.
	//
	// Also, code becomes more complex.
	//
	// Wait for the node to appear...
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case node := <-ch:
		return node, nil
	}
}

// WaitForAlive node to appear or until context is canceled.
func (cl *Cluster) WaitForAlive(ctx context.Context) (dbsql.Node, error) {
	return cl.WaitForNode(ctx, Alive)
}

// WaitForPrimary node to appear or until context is canceled.
func (cl *Cluster) WaitForPrimary(ctx context.Context) (dbsql.Node, error) {
	return cl.WaitForNode(ctx, Primary)
}

// WaitForStandby node to appear or until context is canceled.
func (cl *Cluster) WaitForStandby(ctx context.Context) (dbsql.Node, error) {
	return cl.WaitForNode(ctx, Standby)
}

// WaitForPrimaryPreferred node to appear or until context is canceled.
func (cl *Cluster) WaitForPrimaryPreferred(ctx context.Context) (dbsql.Node, error) {
	return cl.WaitForNode(ctx, PreferPrimary)
}

// WaitForStandbyPreferred node to appear or until context is canceled.
func (cl *Cluster) WaitForStandbyPreferred(ctx context.Context) (dbsql.Node, error) {
	return cl.WaitForNode(ctx, PreferStandby)
}

// backgroundNodesUpdate periodically checks the list of registered nodes.
func (cl *Cluster) backgroundNodesUpdate(ctx context.Context) {
	// initial update
	cl.updateNodes(ctx)

	ticker := time.NewTicker(cl.updateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cl.updateNodes(ctx)
		}
	}
}

// updateNodes performs a new round of cluster state check
// and notifies all subscribers afterward.
func (cl *Cluster) updateNodes(ctx context.Context) {
	if cl.tracer.UpdateNodes != nil {
		cl.tracer.UpdateNodes()
	}

	ctx, cancel := context.WithTimeout(ctx, cl.updateTimeout)
	defer cancel()

	checked := checkNodes(ctx, cl.nodes, cl.checker, cl.picker.CompareNodes, cl.tracer)
	cl.checkedNodes.Store(checked)

	if cl.tracer.NodesUpdated != nil {
		cl.tracer.NodesUpdated(checked)
	}

	cl.notifyUpdateSubscribers(checked)

	if cl.tracer.WaitersNotified != nil {
		cl.tracer.WaitersNotified()
	}
}
