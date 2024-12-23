package cluster

import (
	"testing"

	"github.com/stretchr/testify/assert"

	mockdbsql "github.com/webitel/webitel-wfm/gen/go/mocks/infra/storage/dbsql"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql"
)

func TestRandom(t *testing.T) {
	db := mockdbsql.NewMockDatabase(t)
	n1 := dbsql.New("shimba", db, nil)
	n2 := dbsql.New("boomba", db, nil)
	n3 := dbsql.New("looken", db, nil)
	nodes := []dbsql.Node{n1, n2, n3}
	rr := PickNodeRandom()
	pickedNodes := make(map[string]struct{})
	for i := 0; i < 100; i++ {
		pickedNodes[rr(nodes).Addr()] = struct{}{}
	}
	expectedNodes := map[string]struct{}{"boomba": {}, "looken": {}, "shimba": {}}

	assert.Equal(t, expectedNodes, pickedNodes)
}

func TestPickNodeRoundRobin(t *testing.T) {
	db := mockdbsql.NewMockDatabase(t)
	n1 := dbsql.New("shimba", db, nil)
	n2 := dbsql.New("boomba", db, nil)
	n3 := dbsql.New("looken", db, nil)
	n4 := dbsql.New("tooken", db, nil)
	n5 := dbsql.New("chicken", db, nil)
	n6 := dbsql.New("cooken", db, nil)
	nodes := []dbsql.Node{n1, n2, n3, n4, n5, n6}
	iterCount := len(nodes) * 3

	rr := PickNodeRoundRobin()

	var pickedNodes []string
	for i := 0; i < iterCount; i++ {
		pickedNodes = append(pickedNodes, rr(nodes).Addr())
	}

	expectedNodes := []string{
		"shimba", "boomba", "looken", "tooken", "chicken", "cooken",
		"shimba", "boomba", "looken", "tooken", "chicken", "cooken",
		"shimba", "boomba", "looken", "tooken", "chicken", "cooken",
	}
	assert.Equal(t, expectedNodes, pickedNodes)
}

func TestClosest(t *testing.T) {
	db := mockdbsql.NewMockDatabase(t)
	n1 := dbsql.New("shimba", db, nil)
	n2 := dbsql.New("boomba", db, nil)
	n3 := dbsql.New("looken", db, nil)

	nodes := []dbsql.Node{n1, n2, n3}

	rr := PickNodeClosest()
	assert.Equal(t, nodes[0], rr(nodes))
}
