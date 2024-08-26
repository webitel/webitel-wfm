package pg

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// WithTxFunc represents a function that will be executed within transaction.
type WithTxFunc func(ctx context.Context, tx pgx.Tx) error

// WithTx executes a function within transaction.
func WithTx(ctx context.Context, db *pgxpool.Tx, fn WithTxFunc) error {
	t, err := db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("db.BeginTxx(): %w", err)
	}

	if err = fn(ctx, t); err != nil {
		if errRollback := t.Rollback(ctx); errRollback != nil {
			return fmt.Errorf("Tx.Rollback: %w", err)
		}

		return fmt.Errorf("Tx.WithTxFunc: %w", err)
	}

	if err = t.Commit(ctx); err != nil {
		return fmt.Errorf("Tx.Commit: %w", err)
	}

	return nil
}
