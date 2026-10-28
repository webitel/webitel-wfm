package storage

import "github.com/webitel/webitel-wfm/infra/storage/dbsql"

// readNode picks the node for a lookup read: the standby, unless the read
// follows the caller's own write, which a lagging standby may not have yet.
func readNode(db dbsql.Store, afterWrite bool) dbsql.Node {
	if afterWrite {
		return db.Primary()
	}

	return db.StandbyPreferred()
}
