package dbsql

import (
	"context"
	"database/sql"

	"github.com/webitel/webitel-wfm/infra/storage/dbsql/scanner"
)

type State int

const (
	Alive State = iota + 1
	Dead
)

type BatchNode interface {
	Queue(query string, args ...any)
	Select(ctx context.Context, dest any) error
	Exec(ctx context.Context) error
}

// Node of single Cluster
type Node interface {
	Addr() string
	State() State
	SetState(state State)
	CompareState(state State) bool

	Close()
	Stdlib() *sql.DB

	Select(ctx context.Context, dest interface{}, query string, args ...any) error
	Get(ctx context.Context, dest interface{}, query string, args ...any) error
	Exec(ctx context.Context, query string, args ...any) error

	Batch() BatchNode
}

// Database is an abstract database interface that can execute queries.
// This is used to decouple from any particular database library.
type Database interface {
	Exec(ctx context.Context, query string, args ...any) (int64, error)
	Query(ctx context.Context, query string, args ...any) (scanner.Rows, error)

	Batch() Batcher

	Stdlib() *sql.DB

	Close()
}

type SqlNode struct {
	state State
	addr  string

	db      Database
	scanner scanner.Scanner
}

// New constructs node from Connection
func New(addr string, db Database, scanner scanner.Scanner) *SqlNode {
	return &SqlNode{
		addr:    addr,
		db:      db,
		scanner: scanner,
	}
}

// Addr returns node's address
func (n *SqlNode) Addr() string {
	return n.addr
}

func (n *SqlNode) State() State {
	return n.state
}

func (n *SqlNode) SetState(state State) {
	n.state = state
}

func (n *SqlNode) CompareState(state State) bool {
	return n.state == state
}

// String implements Stringer
func (n *SqlNode) String() string {
	return n.addr
}

func (n *SqlNode) Close() {
	n.db.Close()
}

func (n *SqlNode) Stdlib() *sql.DB {
	return n.db.Stdlib()
}

func (n *SqlNode) Select(ctx context.Context, dest interface{}, query string, args ...any) error {
	rows, err := n.db.Query(ctx, query, args...)
	if err != nil {
		return ParseError(err)
	}

	if err := n.scanner.ScanAll(dest, rows); err != nil {
		return ParseError(err)
	}

	return nil
}

func (n *SqlNode) Get(ctx context.Context, dest interface{}, query string, args ...any) error {
	rows, err := n.db.Query(ctx, query, args...)
	if err != nil {
		return ParseError(err)
	}

	if err := n.scanner.ScanOne(dest, rows); err != nil {
		return ParseError(err)
	}

	return nil
}

func (n *SqlNode) Exec(ctx context.Context, query string, args ...any) error {
	aff, err := n.db.Exec(ctx, query, args...)
	if err != nil {
		return ParseError(err)
	}

	if aff < 0 {
		return ErrNoRows
	}

	return nil
}

func (n *SqlNode) Batch() BatchNode {
	return newSqlNodeBatch(n.db.Batch())
}
