package dbsql

import (
	"context"
	"database/sql"

	"github.com/webitel/webitel-wfm/pkg/werror"
)

var ErrDatabaseNodeDead = werror.Internal("database node is dead", werror.WithID("dbsql.node.dead"))

type NoopNode struct {
	Err error
}

func (n NoopNode) Addr() string {
	return "noop"
}

func (n NoopNode) State() State {
	return Alive
}

func (n NoopNode) SetState(state State) {}

func (n NoopNode) CompareState(state State) bool {
	return false
}

func (n NoopNode) Close() {}

func (n NoopNode) Stdlib() *sql.DB {
	return nil
}

func (n NoopNode) Select(ctx context.Context, dest interface{}, query string, args ...any) error {
	return werror.Wrap(ErrDatabaseNodeDead, werror.WithCause(n.Err))
}

func (n NoopNode) Get(ctx context.Context, dest interface{}, query string, args ...any) error {
	return werror.Wrap(ErrDatabaseNodeDead, werror.WithCause(n.Err))
}

func (n NoopNode) Exec(ctx context.Context, query string, args ...any) error {
	return werror.Wrap(ErrDatabaseNodeDead, werror.WithCause(n.Err))
}

func (n NoopNode) Batch() BatchNode {
	return noopBatchNode{
		err: n.Err,
	}
}

type noopBatchNode struct {
	err error
}

func (n noopBatchNode) Queue(query string, args ...any) {}

func (n noopBatchNode) Select(ctx context.Context, dest any) error {
	return werror.Wrap(ErrDatabaseNodeDead, werror.WithCause(n.err))
}

func (n noopBatchNode) Exec(ctx context.Context) error {
	return werror.Wrap(ErrDatabaseNodeDead, werror.WithCause(n.err))
}
