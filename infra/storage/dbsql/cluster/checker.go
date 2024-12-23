package cluster

import (
	"context"

	"github.com/webitel/webitel-wfm/infra/storage/dbsql"
)

// PostgreSQL checks whether PostgreSQL server is primary or not.
func PostgreSQL(ctx context.Context, db dbsql.Node) (bool, error) {
	return check(ctx, db, "SELECT NOT pg_is_in_recovery()")
}

// check executes a specified Query on specified database pool. Query must return single boolean
// value that signals if that pool is connected to primary or not. All errors are returned as is.
func check(ctx context.Context, db dbsql.Node, query string) (bool, error) {
	var primary bool
	if err := db.Get(ctx, &primary, query); err != nil {
		return false, err
	}

	return primary, nil
}
