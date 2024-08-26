package pg

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/webitel/webitel-go-kit/logging/wlog"
	otelpgx "github.com/webitel/webitel-go-kit/tracing/pgx"

	"github.com/webitel/webitel-wfm/infra/storage/dbsql/cluster"
)

var _ cluster.Queryer = &Database{}

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

func (c *Database) Exec(ctx context.Context, query string, args []any) error {
	if _, err := c.cli.Exec(ctx, query, args...); err != nil {
		return cluster.ParseError(err)
	}

	return nil
}

func (c *Database) Query(ctx context.Context, query string, args []any) (cluster.Rows, error) {
	r, err := c.cli.Query(ctx, query, args...)
	if err != nil {
		return nil, cluster.ParseError(err)
	}

	return NewRowsAdapter(r), nil
}

func (c *Database) Close() {
	c.cli.Close()
}

func (c *Database) Stdlib() *sql.DB {
	return stdlib.OpenDBFromPool(c.cli)
}
