package cluster

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	mockdbsql "github.com/webitel/webitel-wfm/gen/go/mocks/infra/storage/dbsql"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql/scanner"
)

func TestRandomNodePicker(t *testing.T) {
	db := mockdbsql.NewMockDatabase(t)
	sc := scanner.MustNewDBScan()
	t.Run("pick_node", func(t *testing.T) {
		nodes := []CheckedNode{
			{
				Node: dbsql.New("db://user:pass@shimba", db, sc),
			},
			{
				Node: dbsql.New("db://user:pass@boomba", db, sc),
			},
			{
				Node: dbsql.New("db://user:pass@looken", db, sc),
			},
		}

		np := new(RandomNodePicker)
		pickedNodes := make(map[string]struct{})
		for range 100 {
			pickedNodes[np.PickNode(nodes).Node.Addr()] = struct{}{}
		}

		expectedNodes := map[string]struct{}{"db://user@boomba": {}, "db://user@looken": {}, "db://user@shimba": {}}
		assert.Equal(t, expectedNodes, pickedNodes)
	})

	t.Run("compare_nodes", func(t *testing.T) {
		a := CheckedNode{
			Node: dbsql.New("shimba", db, sc),
			Info: NodeInfo{
				ClusterRole:    NodeRolePrimary,
				NetworkLatency: 10 * time.Millisecond,
				ReplicaLag:     1,
			},
		}

		b := CheckedNode{
			Node: dbsql.New("boomba", db, sc),
			Info: NodeInfo{
				ClusterRole:    NodeRoleStandby,
				NetworkLatency: 20 * time.Millisecond,
				ReplicaLag:     2,
			},
		}

		np := new(RandomNodePicker)
		for _, nodeA := range []CheckedNode{a, b} {
			for _, nodeB := range []CheckedNode{a, b} {
				assert.Equal(t, 0, np.CompareNodes(nodeA, nodeB))
			}
		}
	})
}

func TestRoundRobinNodePicker(t *testing.T) {
	db := mockdbsql.NewMockDatabase(t)
	sc := scanner.MustNewDBScan()
	t.Run("pick_node", func(t *testing.T) {
		nodes := []CheckedNode{
			{
				Node: dbsql.New("shimba", db, sc),
			},
			{
				Node: dbsql.New("boomba", db, sc),
			},
			{
				Node: dbsql.New("looken", db, sc),
			},
			{
				Node: dbsql.New("tooken", db, sc),
			},
			{
				Node: dbsql.New("chicken", db, sc),
			},
			{
				Node: dbsql.New("cooken", db, sc),
			},
		}

		np := new(RoundRobinNodePicker)
		var pickedNodes []string
		for range len(nodes) * 3 {
			pickedNodes = append(pickedNodes, np.PickNode(nodes).Node.Addr())
		}

		expectedNodes := []string{
			"shimba", "boomba", "looken", "tooken", "chicken", "cooken",
			"shimba", "boomba", "looken", "tooken", "chicken", "cooken",
			"shimba", "boomba", "looken", "tooken", "chicken", "cooken",
		}

		assert.Equal(t, expectedNodes, pickedNodes)
	})

	t.Run("compare_nodes", func(t *testing.T) {
		a := CheckedNode{
			Node: dbsql.New("shimba", db, sc),
			Info: NodeInfo{
				ClusterRole:    NodeRolePrimary,
				NetworkLatency: 10 * time.Millisecond,
				ReplicaLag:     1,
			},
		}

		b := CheckedNode{
			Node: dbsql.New("boomba", db, sc),
			Info: NodeInfo{
				ClusterRole:    NodeRoleStandby,
				NetworkLatency: 20 * time.Millisecond,
				ReplicaLag:     2,
			},
		}

		np := new(RoundRobinNodePicker)
		assert.Equal(t, 1, np.CompareNodes(a, b))
		assert.Equal(t, -1, np.CompareNodes(b, a))
		assert.Equal(t, 0, np.CompareNodes(a, a))
		assert.Equal(t, 0, np.CompareNodes(b, b))
	})
}

func TestLatencyNodePicker(t *testing.T) {
	db := mockdbsql.NewMockDatabase(t)
	sc := scanner.MustNewDBScan()
	t.Run("pick_node", func(t *testing.T) {
		nodes := []CheckedNode{
			{
				Node: dbsql.New("shimba", db, sc),
			},
			{
				Node: dbsql.New("boomba", db, sc),
			},
			{
				Node: dbsql.New("looken", db, sc),
			},
			{
				Node: dbsql.New("tooken", db, sc),
			},
			{
				Node: dbsql.New("chicken", db, sc),
			},
			{
				Node: dbsql.New("cooken", db, sc),
			},
		}

		np := new(LatencyNodePicker)
		pickedNodes := make(map[string]struct{})
		for range 100 {
			pickedNodes[np.PickNode(nodes).Node.Addr()] = struct{}{}
		}

		expectedNodes := map[string]struct{}{
			"shimba": {},
		}

		assert.Equal(t, expectedNodes, pickedNodes)
	})

	t.Run("compare_nodes", func(t *testing.T) {
		a := CheckedNode{
			Node: dbsql.New("shimba", db, sc),
			Info: NodeInfo{
				ClusterRole:    NodeRolePrimary,
				NetworkLatency: 10 * time.Millisecond,
				ReplicaLag:     1,
			},
		}

		b := CheckedNode{
			Node: dbsql.New("boomba", db, sc),
			Info: NodeInfo{
				ClusterRole:    NodeRoleStandby,
				NetworkLatency: 20 * time.Millisecond,
				ReplicaLag:     2,
			},
		}

		np := new(LatencyNodePicker)

		assert.Equal(t, -1, np.CompareNodes(a, b))
		assert.Equal(t, 1, np.CompareNodes(b, a))
		assert.Equal(t, 0, np.CompareNodes(a, a))
		assert.Equal(t, 0, np.CompareNodes(b, b))
	})
}

func TestReplicationNodePicker(t *testing.T) {
	db := mockdbsql.NewMockDatabase(t)
	sc := scanner.MustNewDBScan()
	t.Run("pick_node", func(t *testing.T) {
		nodes := []CheckedNode{
			{
				Node: dbsql.New("shimba", db, sc),
			},
			{
				Node: dbsql.New("boomba", db, sc),
			},
			{
				Node: dbsql.New("looken", db, sc),
			},
			{
				Node: dbsql.New("tooken", db, sc),
			},
			{
				Node: dbsql.New("chicken", db, sc),
			},
			{
				Node: dbsql.New("cooken", db, sc),
			},
		}

		np := new(ReplicationNodePicker)
		pickedNodes := make(map[string]struct{})
		for range 100 {
			pickedNodes[np.PickNode(nodes).Node.Addr()] = struct{}{}
		}

		expectedNodes := map[string]struct{}{
			"shimba": {},
		}

		assert.Equal(t, expectedNodes, pickedNodes)
	})

	t.Run("compare_nodes", func(t *testing.T) {
		a := CheckedNode{
			Node: dbsql.New("shimba", db, sc),
			Info: NodeInfo{
				ClusterRole:    NodeRolePrimary,
				NetworkLatency: 10 * time.Millisecond,
				ReplicaLag:     1,
			},
		}

		b := CheckedNode{
			Node: dbsql.New("boomba", db, sc),
			Info: NodeInfo{
				ClusterRole:    NodeRoleStandby,
				NetworkLatency: 20 * time.Millisecond,
				ReplicaLag:     2,
			},
		}

		np := new(ReplicationNodePicker)
		assert.Equal(t, -1, np.CompareNodes(a, b))
		assert.Equal(t, 1, np.CompareNodes(b, a))
		assert.Equal(t, 0, np.CompareNodes(a, a))
		assert.Equal(t, 0, np.CompareNodes(b, b))
	})
}
