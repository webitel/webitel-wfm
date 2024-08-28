package dbsql

import (
	"context"
	"fmt"

	"github.com/webitel/webitel-go-kit/logging/wlog"

	"github.com/webitel/webitel-wfm/infra/storage/dbsql/cluster"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql/pg"
)

// NewConnections creates a Connection map from provided dsn strings.
//
// Helps to use NewCluster in tests, so we can create Cluster with mock Connection.
func NewConnections(ctx context.Context, log *wlog.Logger, dsn ...string) (map[string]cluster.Database, error) {
	conns := make(map[string]cluster.Database, len(dsn))
	for i, d := range dsn {
		c, err := pg.NewDatabase(ctx, log, d)
		if err != nil {
			return nil, fmt.Errorf("provide database [%d]: %v", i, err)
		}

		conns[d] = c
	}

	return conns, nil
}
