package leader

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/fx/fxtest"

	"github.com/webitel/webitel-go-kit/infra/discovery/consul"
)

// The consul tests skip unless WFM_TEST_CONSUL_ADDR is set (host:port of a throwaway agent).
type cluster struct {
	t       *testing.T
	addr    string
	service string

	leading atomic.Int32
	overlap atomic.Bool
}

func newCluster(t *testing.T) *cluster {
	t.Helper()

	addr := os.Getenv("WFM_TEST_CONSUL_ADDR")
	if addr == "" {
		t.Skip("WFM_TEST_CONSUL_ADDR is not set")
	}

	return &cluster{t: t, addr: addr, service: fmt.Sprintf("wfm-test-%d", time.Now().UnixNano())}
}

type node struct {
	id   string
	lc   *fxtest.Lifecycle
	task *task
}

func (c *cluster) start(id string) *node {
	c.t.Helper()

	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	e, err := consul.NewLeaderElector(c.addr, c.service, id, log,
		consul.WithLockDelay(time.Second),
		consul.WithRetryInterval(100*time.Millisecond),
		consul.WithErrorCooldown(100*time.Millisecond),
		consul.WithMonitorInterval(100*time.Millisecond),
	)
	if err != nil {
		c.t.Fatal(err)
	}

	n := &node{id: id, lc: fxtest.NewLifecycle(c.t)}
	n.task = &task{name: "job", run: func(ctx context.Context, _ func()) error {
		if c.leading.Add(1) > 1 {
			c.overlap.Store(true)
		}
		defer c.leading.Add(-1)

		<-ctx.Done()

		return ctx.Err()
	}}

	New(e, []Task{n.task}, log, n.lc)
	n.lc.RequireStart()

	return n
}

// key returns the node id and the session that hold the leader key.
func (c *cluster) key() (string, string) {
	c.t.Helper()

	resp, err := http.Get("http://" + c.addr + "/v1/kv/" + consul.LeaderKey(c.service))
	if err != nil {
		c.t.Fatal(err)
	}
	defer resp.Body.Close()

	var pairs []struct{ Value, Session string }
	if resp.StatusCode != http.StatusOK || json.NewDecoder(resp.Body).Decode(&pairs) != nil || len(pairs) == 0 {
		return "", ""
	}

	id, _ := base64.StdEncoding.DecodeString(pairs[0].Value)

	return string(id), pairs[0].Session
}

func (c *cluster) leader(nodes ...*node) *node {
	c.t.Helper()

	var found *node

	eventually(c.t, "one node leads and holds the key", func() bool {
		found = nil

		for _, n := range nodes {
			if n.task.running.Load() == 1 {
				if found != nil {
					return false
				}

				found = n
			}
		}

		id, session := c.key()

		return found != nil && id == found.id && session != ""
	})

	return found
}

func (c *cluster) others(nodes []*node, leader *node) []*node {
	var rest []*node

	for _, n := range nodes {
		if n != leader {
			rest = append(rest, n)
		}
	}

	return rest
}

func TestConsulElectsOneLeader(t *testing.T) {
	c := newCluster(t)
	nodes := []*node{c.start("wfm@a"), c.start("wfm@b"), c.start("wfm@c")}

	first := c.leader(nodes...)

	time.Sleep(500 * time.Millisecond)

	if got := c.leader(nodes...); got != first {
		t.Fatalf("leadership moved from %s to %s with nothing wrong", first.id, got.id)
	}

	for _, n := range nodes {
		n.lc.RequireStop()
	}

	if _, session := c.key(); session != "" {
		t.Fatal("the key is still held after every node stopped")
	}

	if c.overlap.Load() {
		t.Fatal("two nodes ran the task at once")
	}
}

func TestConsulStandbyTakesOverAStoppedLeader(t *testing.T) {
	c := newCluster(t)
	nodes := []*node{c.start("wfm@a"), c.start("wfm@b")}

	first := c.leader(nodes...)
	first.lc.RequireStop()

	if first.task.running.Load() != 0 {
		t.Fatal("stopped with the task still running")
	}

	rest := c.others(nodes, first)
	if second := c.leader(rest...); second.task.started.Load() != 1 {
		t.Fatalf("the standby started its task %d times", second.task.started.Load())
	}

	// The same id leads again after a restart: it owns the leases it left behind.
	rest[0].lc.RequireStop()

	again := c.start(first.id)
	if got := c.leader(again); got.id != first.id {
		t.Fatalf("key holds %s, want %s", got.id, first.id)
	}

	again.lc.RequireStop()

	if c.overlap.Load() {
		t.Fatal("two nodes ran the task at once")
	}
}

func TestConsulLostSessionEndsTheTerm(t *testing.T) {
	c := newCluster(t)
	nodes := []*node{c.start("wfm@a"), c.start("wfm@b")}

	first := c.leader(nodes...)
	_, session := c.key()

	req, _ := http.NewRequest(http.MethodPut, "http://"+c.addr+"/v1/session/destroy/"+session, nil)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	resp.Body.Close()

	eventually(t, "the leader stops its task", func() bool { return first.task.running.Load() == 0 })

	next := c.leader(nodes...)
	if _, now := c.key(); now == session {
		t.Fatal("the key still names the destroyed session")
	}

	if total := nodes[0].task.started.Load() + nodes[1].task.started.Load(); total != 2 {
		t.Fatalf("tasks started %d times over two terms (now leads %s)", total, next.id)
	}

	for _, n := range nodes {
		n.lc.RequireStop()
	}

	if c.overlap.Load() {
		t.Fatal("two nodes ran the task at once")
	}
}
