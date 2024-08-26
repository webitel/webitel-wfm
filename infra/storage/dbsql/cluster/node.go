package cluster

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/georgysavva/scany/v2/dbscan"
)

type NodeState int

const (
	NodeAlive NodeState = iota + 1
	NodeDead
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

// NodeChecker is a signature for functions that check if a specific node is alive and is primary.
// Returns true for primary and false if not.
// If error is returned, the node is considered dead.
// Check function can be used to perform a query returning single boolean value that signals
// if node is primary or not.
type NodeChecker func(ctx context.Context, db Node) (bool, error)

// NodePicker is a signature for functions that determine how to pick single node from set of nodes.
// Nodes passed to the picker function are sorted according to latency (from lowest to greatest).
type NodePicker func(nodes []Node) Node

// Queryer is an abstract database interface that can execute queries.
// This is used to decouple from any particular database library.
type Queryer interface {
	Stdlib() *sql.DB
	Close()

	Exec(ctx context.Context, query string, args []any) error
	Query(ctx context.Context, query string, args []any) (Rows, error)
}

// Rows is an abstract database rows that dbscan can iterate over and get the data from.
// This interface is used to decouple from any particular database library.
type Rows interface {
	Close() error
	Err() error
	Next() bool
	Columns() ([]string, error)
	Scan(dest ...interface{}) error
	NextResultSet() bool
}

// Node of single Cluster
type Node interface {
	Addr() string
	Stdlib() *sql.DB
	Close()

	Select(ctx context.Context, dest interface{}, query string, args ...any) error
	Get(ctx context.Context, dest interface{}, query string, args ...any) error
	Exec(ctx context.Context, query string, args ...any) error

	State() NodeState
	SetState(state NodeState)
	CompareState(state NodeState) bool
}

var _ Node = &sqlNode{}

type sqlNode struct {
	state   NodeState
	addr    string
	db      Queryer
	scanner *dbscan.API
}

// newNode constructs node from Connection
func newNode(addr string, db Queryer) (*sqlNode, error) {
	scanner, err := dbscan.NewAPI()
	if err != nil {
		return nil, fmt.Errorf("create scan API client: %v", err)
	}

	return &sqlNode{addr: addr, db: db, scanner: scanner}, nil
}

// Addr returns node's address
func (n *sqlNode) Addr() string {
	return n.addr
}

func (n *sqlNode) State() NodeState {
	return n.state
}

func (n *sqlNode) SetState(state NodeState) {
	n.state = state
}

func (n *sqlNode) CompareState(state NodeState) bool {
	return n.state == state
}

// String implements Stringer
func (n *sqlNode) String() string {
	return n.addr
}

func (n *sqlNode) Select(ctx context.Context, dest interface{}, query string, args ...any) error {
	rows, err := n.db.Query(ctx, query, args)
	if err != nil {
		return err
	}

	if err := n.scanner.ScanAll(dest, rows); err != nil {
		// TODO: in case of unique / check / foreign key violations the pgx return
		// 	error only after rows.Close call (or while rows.Next returns false).
		// 	That's why we should parse database error while scanning rows also, instead of:
		// 	return apperrors.NewDBInternalError("dbsql.cluster.scan_all", err)
		return ParseError(err)
	}

	return nil
}

func (n *sqlNode) Get(ctx context.Context, dest interface{}, query string, args ...any) error {
	rows, err := n.db.Query(ctx, query, args)
	if err != nil {
		return err
	}

	if err := n.scanner.ScanOne(dest, rows); err != nil {
		return ParseError(err)
	}

	return nil
}

func (n *sqlNode) Exec(ctx context.Context, query string, args ...any) error {
	if err := n.db.Exec(ctx, query, args); err != nil {
		return err
	}

	return nil
}

func (n *sqlNode) Close() {
	n.db.Close()
}

func (n *sqlNode) Stdlib() *sql.DB {
	return n.db.Stdlib()
}
