package cluster

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/huandu/go-sqlbuilder"
	"github.com/webitel/webitel-go-kit/logging/wlog"

	"github.com/webitel/webitel-wfm/infra/health"
	"github.com/webitel/webitel-wfm/infra/shutdown"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql/builder"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql/scanner"
)

// Default values for Cluster config
const (
	DefaultUpdateInterval = time.Second * 5
	DefaultUpdateTimeout  = time.Second
)

type nodeWaiter struct {
	ch            chan Node
	stateCriteria NodeStateCriteria
}

// AliveNodes of Store
type AliveNodes struct {
	Alive     []Node
	Primaries []Node
	Standbys  []Node
}

// Store represents a store that manages a Cluster of nodes.
// It provides methods for retrieving information about the nodes in the Cluster,
// as well as closing the store and checking for any errors.
type Store interface {
	Close() error
	Err() error

	Nodes() []Node
	Alive() Node
	Primary() Node
	Standby() Node
	StandbyPreferred() Node
	Node(criteria NodeStateCriteria) Node

	WaitForPrimary(ctx context.Context) (Node, error)
	WaitForStandby(ctx context.Context) (Node, error)
	WaitForPrimaryPreferred(ctx context.Context) (Node, error)
	WaitForStandbyPreferred(ctx context.Context) (Node, error)
	WaitForAlive(ctx context.Context) (Node, error)

	SQL() *builder.Builder
}

// ForecastStore is a type alias for plumbing it through Wire.
type ForecastStore Store

// Cluster consists of number of 'nodes' of a single SQL database.
// Background goroutine periodically checks nodes and updates their status.
type Cluster struct {
	tracer Tracer

	// Configuration
	update         bool
	updateInterval time.Duration
	updateTimeout  time.Duration
	checker        NodeChecker
	picker         NodePicker

	// Status
	updateStopper chan struct{}
	aliveNodes    atomic.Value
	nodes         []Node
	errCollector  errorsCollector

	// Notification
	muWaiters sync.Mutex
	waiters   []nodeWaiter

	builder *builder.Builder
	scanner scanner.Scanner
}

// NewCluster constructs Cluster object representing a single 'Cluster' of SQL database.
// Close function must be called when a Cluster isn't necessary anymore.
func NewCluster(log *wlog.Logger, conns map[string]Database, opts ...Option) (*Cluster, error) {
	if len(conns) == 0 {
		return nil, errors.New("please provide at least one database connection")
	}

	s, err := scanner.NewDBScan()
	if err != nil {
		return nil, fmt.Errorf("create scan API client: %v", err)
	}

	cl := &Cluster{
		tracer:         DefaultTracer(log),
		updateStopper:  make(chan struct{}),
		updateInterval: DefaultUpdateInterval,
		updateTimeout:  DefaultUpdateTimeout,
		checker:        PostgreSQL,
		picker:         PickNodeClosest(),
		nodes:          make([]Node, 0, len(conns)),
		errCollector:   newErrorsCollector(),
		builder:        builder.NewBuilder(sqlbuilder.PostgreSQL),
		scanner:        s,
	}

	for _, opt := range opts {
		opt(cl)
	}

	for i, c := range conns {
		n, err := newNode(i, c, cl.scanner)
		if err != nil {
			return nil, err
		}

		cl.nodes = append(cl.nodes, n)
	}

	// Store initial nodes state.
	cl.aliveNodes.Store(AliveNodes{
		Alive:     cl.nodes,
		Primaries: cl.nodes,
		Standbys:  cl.nodes,
	})

	if cl.update {
		// Start update routine.
		go cl.backgroundNodesUpdate()

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
	close(cl.updateStopper)

	var wg sync.WaitGroup
	wg.Add(len(cl.nodes))
	for _, node := range cl.nodes {
		node := node
		go func() {
			defer wg.Done()
			node.Close()
		}()
	}

	wg.Wait()

	return nil
}

func (cl *Cluster) HealthCheck(ctx context.Context) []health.CheckResult {
	var reportError error
	if n := cl.Primary(); n != nil {
		reportError = errors.New("primary database not alive")
	}

	return []health.CheckResult{{
		Name: "primary-database",
		Err:  reportError,
	}}
}

// Nodes returns list of all nodes
func (cl *Cluster) Nodes() []Node {
	return cl.nodes
}

func (cl *Cluster) nodesAlive() AliveNodes {
	return cl.aliveNodes.Load().(AliveNodes)
}

func (cl *Cluster) addUpdateWaiter(criteria NodeStateCriteria) <-chan Node {
	// Buffered channel is essential.
	// Read WaitForNode function for more information.
	ch := make(chan Node, 1)
	cl.muWaiters.Lock()
	defer cl.muWaiters.Unlock()
	cl.waiters = append(cl.waiters, nodeWaiter{ch: ch, stateCriteria: criteria})

	return ch
}

// WaitForAlive node to appear or until context is canceled
func (cl *Cluster) WaitForAlive(ctx context.Context) (Node, error) {
	return cl.WaitForNode(ctx, Alive)
}

// WaitForPrimary node to appear or until context is canceled
func (cl *Cluster) WaitForPrimary(ctx context.Context) (Node, error) {
	return cl.WaitForNode(ctx, Primary)
}

// WaitForStandby node to appear or until context is canceled
func (cl *Cluster) WaitForStandby(ctx context.Context) (Node, error) {
	return cl.WaitForNode(ctx, Standby)
}

// WaitForPrimaryPreferred node to appear or until context is canceled
func (cl *Cluster) WaitForPrimaryPreferred(ctx context.Context) (Node, error) {
	return cl.WaitForNode(ctx, PreferPrimary)
}

// WaitForStandbyPreferred node to appear or until context is canceled
func (cl *Cluster) WaitForStandbyPreferred(ctx context.Context) (Node, error) {
	return cl.WaitForNode(ctx, PreferStandby)
}

// WaitForNode with specified status to appear or until context is canceled
func (cl *Cluster) WaitForNode(ctx context.Context, criteria NodeStateCriteria) (Node, error) {
	// Node already exists?
	node := cl.Node(criteria)
	if node != nil {
		return node, nil
	}

	ch := cl.addUpdateWaiter(criteria)

	// Node might have appeared while we were adding waiter, recheck
	node = cl.Node(criteria)
	if node != nil {
		return node, nil
	}

	// If a channel is unbuffered, and we are right here when nodes are updated,
	// the update code won't be able to write into a channel and will 'forget' it.
	// Then we will report nil to the caller, either because update code
	// closes channel, or because context is canceled.
	//
	// In both cases, it's not what user wants.
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

// Alive returns node that is considered alive
func (cl *Cluster) Alive() Node {
	return cl.alive(cl.nodesAlive())
}

func (cl *Cluster) alive(nodes AliveNodes) Node {
	if len(nodes.Alive) == 0 {
		return &sqlNode{}
	}

	return cl.picker(nodes.Alive)
}

// Primary returns first available node that is considered alive and is primary (able to execute write operations)
func (cl *Cluster) Primary() Node {
	return cl.primary(cl.nodesAlive())
}

func (cl *Cluster) primary(nodes AliveNodes) Node {
	if len(nodes.Primaries) == 0 {
		return nil
	}

	return cl.picker(nodes.Primaries)
}

// Standby returns node that is considered alive and is standby (unable to execute write operations)
func (cl *Cluster) Standby() Node {
	return cl.standby(cl.nodesAlive())
}

func (cl *Cluster) standby(nodes AliveNodes) Node {
	if len(nodes.Standbys) == 0 {
		return nil
	}

	// select one of standbys
	return cl.picker(nodes.Standbys)
}

// PrimaryPreferred returns primary node if possible, standby otherwise
func (cl *Cluster) PrimaryPreferred() Node {
	return cl.primaryPreferred(cl.nodesAlive())
}

func (cl *Cluster) primaryPreferred(nodes AliveNodes) Node {
	node := cl.primary(nodes)
	if node == nil {
		node = cl.standby(nodes)
	}

	return node
}

// StandbyPreferred returns standby node if possible, primary otherwise
func (cl *Cluster) StandbyPreferred() Node {
	return cl.standbyPreferred(cl.nodesAlive())
}

func (cl *Cluster) standbyPreferred(nodes AliveNodes) Node {
	node := cl.standby(nodes)
	if node == nil {
		node = cl.primary(nodes)
	}

	return node
}

// Node returns Cluster node with specified status.
func (cl *Cluster) Node(criteria NodeStateCriteria) Node {
	return cl.node(cl.nodesAlive(), criteria)
}

func (cl *Cluster) node(nodes AliveNodes, criteria NodeStateCriteria) Node {
	switch criteria {
	case Alive:
		return cl.alive(nodes)
	case Primary:
		return cl.primary(nodes)
	case Standby:
		return cl.standby(nodes)
	case PreferPrimary:
		return cl.primaryPreferred(nodes)
	case PreferStandby:
		return cl.standbyPreferred(nodes)
	default:
		panic(fmt.Sprintf("unknown node state criteria: %d", criteria))
	}
}

// Err returns the combined error including most recent errors for all nodes.
// This error is CollectedErrors or nil.
func (cl *Cluster) Err() error {
	return cl.errCollector.Err()
}

// backgroundNodesUpdate periodically update list of live db nodes
func (cl *Cluster) backgroundNodesUpdate() {
	// Initial update
	cl.updateNodes()

	ticker := time.NewTicker(cl.updateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-cl.updateStopper:
			return
		case <-ticker.C:
			cl.updateNodes()
		}
	}
}

// updateNodes pings all db nodes and stores alive ones in a separate slice
func (cl *Cluster) updateNodes() {
	if cl.tracer.UpdateNodes != nil {
		cl.tracer.UpdateNodes()
	}

	ctx, cancel := context.WithTimeout(context.Background(), cl.updateTimeout)
	defer cancel()

	alive := checkNodes(ctx, cl.nodes, checkExecutor(cl.checker), cl.tracer, &cl.errCollector)
	cl.aliveNodes.Store(alive)

	if cl.tracer.UpdatedNodes != nil {
		cl.tracer.UpdatedNodes(alive)
	}

	cl.notifyWaiters(alive)

	if cl.tracer.NotifiedWaiters != nil {
		cl.tracer.NotifiedWaiters()
	}
}

func (cl *Cluster) notifyWaiters(nodes AliveNodes) {
	cl.muWaiters.Lock()
	defer cl.muWaiters.Unlock()

	if len(cl.waiters) == 0 {
		return
	}

	var nodelessWaiters []nodeWaiter
	// Notify all waiters
	for _, waiter := range cl.waiters {
		node := cl.node(nodes, waiter.stateCriteria)
		if node == nil {
			// Put waiter back
			nodelessWaiters = append(nodelessWaiters, waiter)
			continue
		}

		// We won't block here, read addUpdateWaiter function for more information
		waiter.ch <- node
		// No need to close a channel since we write only once and forget it, so does the 'client'
	}

	cl.waiters = nodelessWaiters
}

func (cl *Cluster) SQL() *builder.Builder {
	return cl.builder
}
