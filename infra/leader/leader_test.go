package leader

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"go.uber.org/fx/fxtest"
	"go.uber.org/goleak"
)

// The runner starts goroutines for the election and for every task; none may outlive the tests.
// The consul client's idle keep-alive connections are pooled, not leaked.
func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m,
		goleak.IgnoreAnyFunction("net/http.(*persistConn).readLoop"),
		goleak.IgnoreAnyFunction("net/http.(*persistConn).writeLoop"))
}

// elector gives and takes leadership on demand, with the contract of consul.LeaderElector:
// the term is released only after onStart returns.
type elector struct {
	promote chan struct{}
	demote  chan struct{}
	stopped chan struct{}
}

func newElector() *elector {
	return &elector{promote: make(chan struct{}), demote: make(chan struct{}), stopped: make(chan struct{}, 8)}
}

func (e *elector) Run(ctx context.Context, onStart func(context.Context) error, onStop func()) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-e.promote:
		}

		term, end := context.WithCancel(ctx)
		work := make(chan error, 1)

		go func() { work <- onStart(term) }()

		select {
		case <-ctx.Done():
		case <-e.demote:
		case <-work:
			work <- errors.New("the term ended on its own")
		}

		end()
		<-work
		onStop()

		e.stopped <- struct{}{}
	}
}

type task struct {
	name    string
	run     func(ctx context.Context, tick func()) error
	started atomic.Int32
	running atomic.Int32
}

func (t *task) Name() string { return t.name }

func (t *task) Run(ctx context.Context, tick func()) error {
	t.started.Add(1)

	t.running.Add(1)
	defer t.running.Add(-1)

	if t.run != nil {
		return t.run(ctx, tick)
	}

	<-ctx.Done()

	return ctx.Err()
}

func newRunner(t *testing.T, e *elector, tasks ...Task) (*Runner, *fxtest.Lifecycle) {
	t.Helper()

	lc := fxtest.NewLifecycle(t)
	r := New(e, tasks, slog.New(slog.NewTextHandler(io.Discard, nil)), lc)

	lc.RequireStart()

	return r, lc
}

// The tests run in a synctest bubble: time is fake, so the runner keeps its real delays, and
// synctest.Wait returns once every goroutine of the runner is blocked.

func TestTasksFollowTheTerm(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		e, a, b := newElector(), &task{name: "a"}, &task{name: "b"}
		_, lc := newRunner(t, e, a, b)

		synctest.Wait()

		if a.started.Load() != 0 || b.started.Load() != 0 {
			t.Fatal("tasks started on a standby")
		}

		for term := int32(1); term <= 3; term++ {
			e.promote <- struct{}{}

			synctest.Wait()

			if a.running.Load() != 1 || b.running.Load() != 1 {
				t.Fatalf("term %d: running a=%d b=%d, want both", term, a.running.Load(), b.running.Load())
			}

			e.demote <- struct{}{}

			<-e.stopped

			if a.running.Load() != 0 || b.running.Load() != 0 {
				t.Fatalf("term %d: the key was released with tasks still running", term)
			}

			if a.started.Load() != term || b.started.Load() != term {
				t.Fatalf("term %d: started a=%d b=%d", term, a.started.Load(), b.started.Load())
			}
		}

		lc.RequireStop()
	})
}

func TestFailedTaskRestartsWithinTheTerm(t *testing.T) {
	boom := errors.New("boom")

	for name, run := range map[string]func(context.Context, func()) error{
		"error":        func(context.Context, func()) error { return boom },
		"panic":        func(context.Context, func()) error { panic("boom") },
		"early return": func(context.Context, func()) error { return nil },
	} {
		t.Run(name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				e, bad, good := newElector(), &task{name: "bad", run: run}, &task{name: "good"}
				r, lc := newRunner(t, e, bad, good)

				e.promote <- struct{}{}

				time.Sleep(2*r.restartDelay + time.Second)
				synctest.Wait()

				if got := bad.started.Load(); got != 3 {
					t.Fatalf("the failed task started %d times in two restart delays, want 3", got)
				}

				select {
				case <-e.stopped:
					t.Fatal("a failed task ended the term")
				default:
				}

				if good.started.Load() != 1 || good.running.Load() != 1 {
					t.Fatalf("the other task was disturbed: started %d, running %d", good.started.Load(), good.running.Load())
				}

				lc.RequireStop()

				if good.running.Load() != 0 {
					t.Fatal("stopped with a task still running")
				}
			})
		})
	}
}

func TestNoRestartAfterTheTerm(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		e := newElector()
		bad := &task{name: "bad", run: func(ctx context.Context, _ func()) error {
			<-ctx.Done()

			return errors.New("boom")
		}}
		r, lc := newRunner(t, e, bad)

		e.promote <- struct{}{}

		synctest.Wait()

		e.demote <- struct{}{}

		<-e.stopped
		time.Sleep(2 * r.restartDelay)
		synctest.Wait()

		if got := bad.started.Load(); got != 1 {
			t.Fatalf("started %d times, want 1: restarted on a standby", got)
		}

		lc.RequireStop()
	})
}

func TestRestartDelayEndsWithTheTerm(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		e := newElector()
		bad := &task{name: "bad", run: func(context.Context, func()) error { return errors.New("boom") }}

		lc := fxtest.NewLifecycle(t)
		r := New(e, []Task{bad}, slog.New(slog.NewTextHandler(io.Discard, nil)), lc)
		r.restartDelay = time.Hour

		lc.RequireStart()

		e.promote <- struct{}{}

		synctest.Wait()

		e.demote <- struct{}{}

		select {
		case <-e.stopped:
		case <-time.After(time.Minute):
			t.Fatal("the key is held for the restart delay of a failed task")
		}

		if got := bad.started.Load(); got != 1 {
			t.Fatalf("started %d times, want 1", got)
		}

		lc.RequireStop()
	})
}

func TestTermWithoutTasksIsHeld(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		e := newElector()
		_, lc := newRunner(t, e)

		e.promote <- struct{}{}

		synctest.Wait()

		select {
		case <-e.stopped:
			t.Fatal("the term ended on its own")
		default:
		}

		lc.RequireStop()
	})
}

func TestShutdownWaitsForTasks(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		e, release := newElector(), make(chan struct{})
		slow := &task{name: "slow", run: func(ctx context.Context, _ func()) error {
			<-ctx.Done()
			<-release

			return nil
		}}
		r, _ := newRunner(t, e, slow)

		e.promote <- struct{}{}

		synctest.Wait()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		if err := r.shutdown(ctx); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("shutdown: %v, want the deadline: the task has not returned", err)
		}

		select {
		case <-e.stopped:
			t.Fatal("the key was released with the task still running")
		default:
		}

		close(release)

		if err := r.shutdown(context.Background()); err != nil {
			t.Fatal(err)
		}

		if slow.running.Load() != 0 {
			t.Fatal("shutdown returned with the task still running")
		}
	})
}

func TestHealthCheck(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		e, ticks := newElector(), make(chan struct{})
		ticking := &task{name: "ticking", run: func(ctx context.Context, tick func()) error {
			for {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-ticks:
					tick()
				}
			}
		}}
		r, lc := newRunner(t, e, ticking)

		check := func(want string) {
			t.Helper()

			err := r.HealthCheck(context.Background())
			if want == "" && err != nil {
				t.Fatalf("unhealthy: %v", err)
			}

			if want != "" && (err == nil || !strings.Contains(err.Error(), want)) {
				t.Fatalf("got %v, want %q", err, want)
			}
		}

		time.Sleep(2 * r.tickStale)
		check("")

		e.promote <- struct{}{}

		synctest.Wait()
		check("")

		time.Sleep(2 * r.tickStale)
		check("leader task ticking: no tick for 2m0s")

		ticks <- struct{}{}

		synctest.Wait()
		check("")

		time.Sleep(2 * r.tickStale)
		check("no tick")

		e.demote <- struct{}{}

		<-e.stopped
		check("")

		lc.RequireStop()
		check("")
	})
}

func TestHealthCheckOfARestartingTask(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		e := newElector()
		bad := &task{name: "bad", run: func(context.Context, func()) error { return errors.New("boom") }}
		r, lc := newRunner(t, e, bad)

		e.promote <- struct{}{}

		for deadline := time.Now().Add(4 * r.tickStale); time.Now().Before(deadline); time.Sleep(r.restartDelay / 2) {
			if err := r.HealthCheck(context.Background()); err != nil {
				t.Fatalf("a task that keeps restarting is not wedged: %v", err)
			}
		}

		lc.RequireStop()
	})
}
