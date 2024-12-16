package dbsql

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/webitel/webitel-go-kit/logging/wlog"

	"github.com/webitel/webitel-wfm/infra/storage/dbsql/batch"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql/pg"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql/scanner"
)

// Database is an abstract database interface that can execute queries.
// This is used to decouple from any particular database library.
type Database interface {
	Exec(ctx context.Context, query string, args ...any) (int64, error)
	Query(ctx context.Context, query string, args ...any) (scanner.Rows, error)

	Batch() batch.Batcher

	Stdlib() *sql.DB

	Close()
}

// NewConnections creates a Connection map from provided dsn strings.
//
// Helps to use NewCluster in tests, so we can create Cluster with mock Connection.
func NewConnections(ctx context.Context, log *wlog.Logger, dsn ...string) (map[string]Database, error) {
	conns := make(map[string]Database, len(dsn))
	for i, d := range dsn {
		c, err := pg.NewDatabase(ctx, log, d)
		if err != nil {
			return nil, fmt.Errorf("provide database [%d]: %v", i, err)
		}

		conns[d] = c
	}

	return conns, nil
}
