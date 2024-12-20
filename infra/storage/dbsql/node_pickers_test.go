package dbsql

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/webitel/webitel-wfm/gen/go/mocks/infra/storage/dbsql"
)

func TestRandom(t *testing.T) {
	db := dbsql.NewMockDatabase(t)
	n1, _ := newNode("shimba", db, nil)
	n2, _ := newNode("boomba", db, nil)
	n3, _ := newNode("looken", db, nil)
	nodes := []Node{n1, n2, n3}
	rr := PickNodeRandom()
	pickedNodes := make(map[string]struct{})
	for i := 0; i < 100; i++ {
		pickedNodes[rr(nodes).Addr()] = struct{}{}
	}
	expectedNodes := map[string]struct{}{"boomba": {}, "looken": {}, "shimba": {}}

	assert.Equal(t, expectedNodes, pickedNodes)
}

func TestPickNodeRoundRobin(t *testing.T) {
	db := dbsql.NewMockDatabase(t)
	n1, _ := newNode("shimba", db, nil)
	n2, _ := newNode("boomba", db, nil)
	n3, _ := newNode("looken", db, nil)
	n4, _ := newNode("tooken", db, nil)
	n5, _ := newNode("chicken", db, nil)
	n6, _ := newNode("cooken", db, nil)
	nodes := []Node{n1, n2, n3, n4, n5, n6}
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
	db := dbsql.NewMockDatabase(t)
	n1, _ := newNode("shimba", db, nil)
	n2, _ := newNode("boomba", db, nil)
	n3, _ := newNode("looken", db, nil)

	nodes := []Node{n1, n2, n3}

	rr := PickNodeClosest()
	assert.Equal(t, nodes[0], rr(nodes))
}
