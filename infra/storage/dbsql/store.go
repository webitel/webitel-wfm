package dbsql

import (
	"context"
	"sync"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/webitel/webitel-go-kit/infra/pgw"
	"github.com/webitel/webitel-go-kit/pkg/errors"
)

var errDatabaseNodeDead = errors.Internal("database node is dead", errors.WithID("dbsql.node.dead"))

// Node runs queries against one role of the cluster and maps driver errors
// onto the API error contract.
type Node interface {
	Select(ctx context.Context, dest any, query string, args ...any) error
	Get(ctx context.Context, dest any, query string, args ...any) error
	Exec(ctx context.Context, query string, args ...any) error

	Batch() BatchNode
}

// BatchNode collects queries and runs them in one round trip.
type BatchNode interface {
	Queue(query string, args ...any)
	Select(ctx context.Context, dest any) error
}

type Store interface {
	Primary() Node
	StandbyPreferred() Node
}

// ForecastStore is the same contract bound to the forecast database.
type ForecastStore Store

// DB routes queries through a go-kit pool manager. Connections are acquired
// directly rather than going through the manager's query methods: its error
// parser replaces driver errors with types that carry no SQLSTATE, which
// ParseError needs to keep constraint violations distinguishable.
type DB struct {
	pm *pgw.PoolManager

	closeOnce sync.Once
}

func New(pm *pgw.PoolManager) *DB {
	return &DB{pm: pm}
}

func (d *DB) Primary() Node {
	pool, err := d.pm.Primary()
	if err != nil {
		return errNode{err: err}
	}

	return node{pool: pool}
}

func (d *DB) StandbyPreferred() Node {
	pool, err := d.pm.StandbyPreferred()
	if err != nil {
		return errNode{err: err}
	}

	return node{pool: pool}
}

// HealthCheck reports the primary unhealthy while the pool manager has no
// connected primary to hand out; the manager keeps retrying in the background.
func (d *DB) HealthCheck(context.Context) error {
	_, err := d.pm.Primary()

	return err
}

func (d *DB) Close() error {
	d.closeOnce.Do(d.pm.Close)

	return nil
}

type node struct {
	pool *pgw.Pool
}

func (n node) Select(ctx context.Context, dest any, query string, args ...any) error {
	return ParseError(n.pool.AcquireFunc(ctx, func(conn *pgxpool.Conn) error {
		return pgxscan.Select(ctx, conn, dest, query, args...)
	}))
}

func (n node) Get(ctx context.Context, dest any, query string, args ...any) error {
	return ParseError(n.pool.AcquireFunc(ctx, func(conn *pgxpool.Conn) error {
		return pgxscan.Get(ctx, conn, dest, query, args...)
	}))
}

func (n node) Exec(ctx context.Context, query string, args ...any) error {
	return ParseError(n.pool.AcquireFunc(ctx, func(conn *pgxpool.Conn) error {
		_, err := conn.Exec(ctx, query, args...)

		return err
	}))
}

func (n node) Batch() BatchNode {
	return &batch{pool: n.pool}
}

// errNode stands in for a role that is unreachable right now.
type errNode struct {
	err error
}

func (n errNode) Select(context.Context, any, string, ...any) error { return n.fail() }
func (n errNode) Get(context.Context, any, string, ...any) error    { return n.fail() }
func (n errNode) Exec(context.Context, string, ...any) error        { return n.fail() }
func (n errNode) Batch() BatchNode                                  { return errBatch(n) }

func (n errNode) fail() error {
	return errors.Wrap(errDatabaseNodeDead, errors.WithCause(n.err))
}

type errBatch errNode

func (b errBatch) Queue(string, ...any)              {}
func (b errBatch) Select(context.Context, any) error { return errNode(b).fail() }
