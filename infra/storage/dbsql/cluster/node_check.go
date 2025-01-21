package cluster

import (
	"context"
	"math"
	"time"

	"github.com/webitel/webitel-wfm/infra/storage/dbsql"
)

// NodeRole represents a role of node in SQL cluster (usually primary/standby).
type NodeRole uint8

const (

	// NodeRoleUnknown used to report node with an unconventional role in cluster.
	NodeRoleUnknown NodeRole = iota

	// NodeRolePrimary used to report node with a primary role in cluster.
	NodeRolePrimary

	// NodeRoleStandby used to report node with a standby role in cluster.
	NodeRoleStandby
)

// NodeInfoProvider information about single cluster node.
type NodeInfoProvider interface {

	// Role reports a role of node in cluster.
	// For SQL servers, it is usually either primary or standby.
	Role() NodeRole
}

// NodeInfo contains various information about single cluster node.
type NodeInfo struct {

	// Role contains determined node's role in cluster.
	ClusterRole NodeRole `db:"role"`

	// Latency stores time that has been spent to send check request.
	// and receive response from server
	NetworkLatency time.Duration `db:"network_latency"`

	// ReplicaLag represents how far behind is data on standby
	// in comparison to primary. As determination of real replication
	// lag is a tricky task and value type vary from one DBMS to another
	// (e.g., bytes count lag, time delta lag etc.) this field contains
	// abstract value for sorting purposes only.
	ReplicaLag int `db:"replication_lag"`
}

// Role reports determined role of node in cluster.
func (n NodeInfo) Role() NodeRole {
	return n.ClusterRole
}

// Latency reports time spend on query execution from client's point of view.
// It can be used in LatencyNodePicker to determine node with fastest response time.
func (n NodeInfo) Latency() time.Duration {
	return n.NetworkLatency
}

// ReplicationLag reports data replication delta on standby.
// It can be used in ReplicationNodePicker to determine node with most up-to-date data.
func (n NodeInfo) ReplicationLag() int {
	return n.ReplicaLag
}

// NodeChecker is a function that can perform request to SQL node and retrieve various information.
type NodeChecker func(context.Context, dbsql.Node) (NodeInfoProvider, error)

// PostgreSQLChecker checks state on PostgreSQL node.
// It reports appropriate information for PostgreSQL nodes version 10 and higher.
func PostgreSQLChecker(ctx context.Context, db dbsql.Node) (NodeInfoProvider, error) {
	start := time.Now()

	var nodeInfo NodeInfo
	query := `SELECT ((pg_is_in_recovery())::int + 1) AS role,
				-- primary node has no replication lag
				COALESCE(pg_last_wal_receive_lsn() - pg_last_wal_replay_lsn(), 0) AS replication_lag`

	if err := db.Get(ctx, &nodeInfo, query); err != nil {
		return nil, err
	}

	nodeInfo.NetworkLatency = time.Since(start)

	// determine proper replication lag value
	// by default we assume that replication is not started - hence maximum int value
	// see: https://www.postgresql.org/docs/current/functions-admin.html#FUNCTIONS-RECOVERY-CONTROL
	if nodeInfo.ReplicaLag == 0 && nodeInfo.ClusterRole == NodeRoleStandby {
		nodeInfo.ReplicaLag = math.MaxInt
	}

	return nodeInfo, nil
}
