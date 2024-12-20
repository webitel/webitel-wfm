package pg

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/webitel/webitel-go-kit/logging/wlog"
	otelpgx "github.com/webitel/webitel-go-kit/tracing/pgx"

	"github.com/webitel/webitel-wfm/infra/storage/dbsql/batch"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql/scanner"
)

type Database struct {
	log *wlog.Logger
	cli *pgxpool.Pool
}

func NewDatabase(ctx context.Context, log *wlog.Logger, dsn string) (*Database, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse dsn: %v", err)
	}

	cfg.ConnConfig.Tracer = otelpgx.NewTracer(otelpgx.WithTrimSQLInSpanName())
	dbpool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create connection pool: %v", err)
	}

	if err := dbpool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping database: %v", err)
	}

	return &Database{log: log, cli: dbpool}, nil
}

func (db *Database) Exec(ctx context.Context, query string, args ...any) (int64, error) {
	res, err := db.cli.Exec(ctx, query, args...)
	if err != nil {
		return 0, err
	}

	return res.RowsAffected(), nil
}

func (db *Database) Query(ctx context.Context, query string, args ...any) (scanner.Rows, error) {
	r, err := db.cli.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	return NewRowsAdapter(r), nil
}

func (db *Database) Batch() batch.Batcher {
	return newBatch(db.cli)
}

func (db *Database) Close() {
	db.cli.Close()
}

func (db *Database) Stdlib() *sql.DB {
	return stdlib.OpenDBFromPool(db.cli)
}
