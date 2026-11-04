package dbsql

import (
	"context"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/jackc/pgx/v5"

	"github.com/webitel/webitel-go-kit/pkg/errors"
)

var txOptions = pgx.TxOptions{IsoLevel: pgx.ReadCommitted}

type txBeginner interface {
	BeginTx(ctx context.Context, opts pgx.TxOptions) (pgx.Tx, error)
}

var (
	_ Store = (*DB)(nil)
	_ Store = txStore{}
)

func (d *DB) InTx(ctx context.Context, fn func(ctx context.Context, tx Store) error) error {
	pool, err := d.pm.Primary()
	if err != nil {
		return errNode{err: err}.fail()
	}

	conn, err := pool.Acquire(ctx)
	if err != nil {
		return ParseError(err)
	}
	defer conn.Release()

	return inTx(ctx, conn, fn)
}

func inTx(ctx context.Context, db txBeginner, fn func(ctx context.Context, tx Store) error) error {
	tx, err := db.BeginTx(ctx, txOptions)
	if err != nil {
		return ParseError(err)
	}

	defer func() {
		if p := recover(); p != nil {
			// WithoutCancel: on a canceled ctx pgx cannot send ROLLBACK and drops the connection.
			_ = tx.Rollback(context.WithoutCancel(ctx))

			panic(p)
		}
	}()

	err = fn(ctx, txStore{tx: tx})
	if err == nil {
		// Canceled after the last statement: roll back rather than let Commit drop the connection.
		err = ParseError(ctx.Err())
	}

	if err != nil {
		if rbErr := tx.Rollback(context.WithoutCancel(ctx)); rbErr != nil {
			return errors.Wrap(err, errors.WithValue("rollback", rbErr.Error()))
		}

		return err
	}

	return ParseError(tx.Commit(ctx))
}

type txStore struct {
	tx pgx.Tx
}

func (s txStore) Primary() Node          { return txNode(s) }
func (s txStore) StandbyPreferred() Node { return txNode(s) }

func (s txStore) InTx(ctx context.Context, fn func(ctx context.Context, tx Store) error) error {
	return fn(ctx, s)
}

type txNode struct {
	tx pgx.Tx
}

func (n txNode) Select(ctx context.Context, dest any, query string, args ...any) error {
	return ParseError(pgxscan.Select(ctx, n.tx, dest, query, args...))
}

func (n txNode) Get(ctx context.Context, dest any, query string, args ...any) error {
	return ParseError(pgxscan.Get(ctx, n.tx, dest, query, args...))
}

func (n txNode) Exec(ctx context.Context, query string, args ...any) error {
	_, err := n.tx.Exec(ctx, query, args...)

	return ParseError(err)
}

func (n txNode) Batch() BatchNode {
	return &batch{tx: n.tx}
}
