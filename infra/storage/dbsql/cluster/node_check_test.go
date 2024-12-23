package cluster

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	mockdbsql "github.com/webitel/webitel-wfm/gen/go/mocks/infra/storage/dbsql"
	"github.com/webitel/webitel-wfm/infra/storage/dbsql"
)

func TestCheckedNodesList_Len(t *testing.T) {
	nodes := checkedNodesList{checkedNode{}, checkedNode{}, checkedNode{}}
	require.Equal(t, 3, nodes.Len())
}

func TestCheckedNodesList_Less(t *testing.T) {
	nodes := checkedNodesList{checkedNode{Latency: time.Nanosecond}, checkedNode{Latency: 2 * time.Nanosecond}}
	require.True(t, nodes.Less(0, 1))
	require.False(t, nodes.Less(1, 0))
}

func TestCheckedNodesList_Swap(t *testing.T) {
	nodes := checkedNodesList{checkedNode{Latency: time.Nanosecond}, checkedNode{Latency: 2 * time.Nanosecond}}
	nodes.Swap(0, 1)
	assert.Equal(t, 2*time.Nanosecond, nodes[0].Latency)
	assert.Equal(t, time.Nanosecond, nodes[1].Latency)
}

func TestCheckedNodesList_Sort(t *testing.T) {
	nodes := checkedNodesList{checkedNode{Latency: 2 * time.Nanosecond}, checkedNode{Latency: 3 * time.Nanosecond}, checkedNode{Latency: time.Nanosecond}}
	sort.Sort(nodes)
	for i := range nodes {
		assert.Equal(t, time.Duration(i+1)*time.Nanosecond, nodes[i].Latency)
	}
}

func TestGroupedCheckedNodes_Alive(t *testing.T) {
	// TODO: this test does not cover all the cases but better than nothing
	const count = 10
	var expected []dbsql.Node
	var input groupedCheckedNodes
	for i := 0; i < count; i++ {
		db := mockdbsql.NewMockDatabase(t)
		require.NotNil(t, db)

		n := dbsql.New(fmt.Sprintf("%d", i), db, nil)
		require.NotNil(t, n)

		node := checkedNode{Node: n, Latency: time.Duration(i+1) * time.Nanosecond}
		expected = append(expected, node.Node)
		if i%2 == 0 {
			input.Primaries = append(input.Primaries, node)
		} else {
			input.Standbys = append(input.Standbys, node)
		}
	}

	require.Len(t, expected, count)
	require.NotEmpty(t, input.Primaries)
	require.NotEmpty(t, input.Standbys)
	require.Equal(t, count, len(input.Primaries)+len(input.Standbys))

	alive := input.Alive()
	require.Len(t, alive, count)
	require.Equal(t, expected, alive)
}

func TestCheckNodes(t *testing.T) {
	const count = 100
	var nodes []dbsql.Node
	expected := AliveNodes{Alive: make([]dbsql.Node, count)}
	for i := 0; i < count; i++ {

		db := mockdbsql.NewMockDatabase(t)
		require.NotNil(t, db)

		n := dbsql.New(fmt.Sprintf("%d", i), db, nil)
		require.NotNil(t, n)

		for {
			// Randomize 'order' (latency)
			pos := rand.Intn(count)
			if expected.Alive[pos] == nil {
				expected.Alive[pos] = n
				break
			}
		}

		nodes = append(nodes, n)
	}

	require.Len(t, expected.Alive, count)

	// Fill primaries and standbys
	for i, node := range expected.Alive {
		if i%2 == 0 {
			expected.Primaries = append(expected.Primaries, node)
		} else {
			expected.Standbys = append(expected.Standbys, node)
		}
	}

	require.NotEmpty(t, expected.Primaries)
	require.NotEmpty(t, expected.Standbys)
	require.Equal(t, count, len(expected.Primaries)+len(expected.Standbys))

	executor := func(ctx context.Context, node dbsql.Node) (bool, time.Duration, error) {
		// Alive nodes set the expected 'order' (latency) of all available nodes.
		// Return duration based on that order.
		var duration time.Duration
		for i, alive := range expected.Alive {
			if alive == node {
				duration = time.Duration(i) * time.Nanosecond
				break
			}
		}

		for _, primary := range expected.Primaries {
			if primary == node {
				return true, duration, nil
			}
		}

		for _, standby := range expected.Standbys {
			if standby == node {
				return false, duration, nil
			}
		}

		return false, 0, errors.New("node not found")
	}

	errCollector := newErrorsCollector()
	alive := checkNodes(context.Background(), nodes, executor, Tracer{}, &errCollector)

	assert.NoError(t, errCollector.Err())
	assert.Equal(t, expected.Primaries, alive.Primaries)
	assert.Equal(t, expected.Standbys, alive.Standbys)
	assert.Equal(t, expected.Alive, alive.Alive)
}

func TestCheckNodesWithErrors(t *testing.T) {
	const count = 5
	var nodes []dbsql.Node
	for i := 0; i < count; i++ {
		db := mockdbsql.NewMockDatabase(t)
		require.NotNil(t, db)

		n := dbsql.New(fmt.Sprintf("%d", i), db, nil)
		require.NotNil(t, n)
		nodes = append(nodes, n)
	}

	executor := func(ctx context.Context, node dbsql.Node) (bool, time.Duration, error) {
		return false, 0, errors.New("node not found")
	}

	errCollector := newErrorsCollector()
	checkNodes(context.Background(), nodes, executor, Tracer{}, &errCollector)

	err := errCollector.Err()
	for i := 0; i < count; i++ {
		assert.ErrorContains(t, err, fmt.Sprintf("%q node error occurred at", nodes[i].Addr()))
	}
	assert.ErrorContains(t, err, "node not found")
}

func TestCheckNodesWithErrorsWhenNodesBecameAlive(t *testing.T) {
	const count = 5
	var nodes []dbsql.Node
	for i := 0; i < count; i++ {
		db := mockdbsql.NewMockDatabase(t)
		require.NotNil(t, db)

		n := dbsql.New(fmt.Sprintf("%d", i), db, nil)
		require.NotNil(t, n)
		nodes = append(nodes, n)
	}

	executor := func(ctx context.Context, node dbsql.Node) (bool, time.Duration, error) {
		return false, 0, errors.New("node not found")
	}

	errCollector := newErrorsCollector()
	checkNodes(context.Background(), nodes, executor, Tracer{}, &errCollector)
	require.Error(t, errCollector.Err())

	executor = func(ctx context.Context, node dbsql.Node) (bool, time.Duration, error) {
		return true, 1, nil
	}
	checkNodes(context.Background(), nodes, executor, Tracer{}, &errCollector)
	require.NoError(t, errCollector.Err())
}
