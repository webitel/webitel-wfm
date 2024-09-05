package pg

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/webitel/webitel-go-kit/logging/wlog"
	otelpgx "github.com/webitel/webitel-go-kit/tracing/pgx"

	"github.com/webitel/webitel-wfm/infra/storage/dbsql/cluster"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql/errors"
	"github.com/webitel/webitel-wfm/pkg/werror"
)

var _ cluster.Database = &Database{}

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

func (db *Database) Exec(ctx context.Context, query string, args ...any) error {
	res, err := db.cli.Exec(ctx, query, args...)
	if err != nil {
		return errors.ParseError(err)
	}

	if res.RowsAffected() == 0 {
		return werror.NewDBNoRowsErr("pg.exec.rows_affected")
	}

	return nil
}

func (db *Database) Query(ctx context.Context, query string, args ...any) (cluster.Rows, error) {
	r, err := db.cli.Query(ctx, query, args...)
	if err != nil {
		return nil, errors.ParseError(err)
	}

	return NewRowsAdapter(r), nil
}

func (db *Database) SendBatch(ctx context.Context, queries []*cluster.Query) cluster.BatchResults {
	b := pgx.Batch{}
	for _, query := range queries {
		b.Queue(query.SQL, query.Args...)
	}

	return NewBatchResultsAdapter(db.cli.SendBatch(ctx, &b))
}

func (db *Database) Close() {
	db.cli.Close()
}

func (db *Database) Stdlib() *sql.DB {
	return stdlib.OpenDBFromPool(db.cli)
}
