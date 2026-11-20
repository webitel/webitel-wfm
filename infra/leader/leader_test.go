package leader

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/fx/fxtest"
)

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
	r.restartDelay = 10 * time.Millisecond

	lc.RequireStart()

	return r, lc
}

func eventually(t *testing.T, what string, ok func() bool) {
	t.Helper()

	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(time.Millisecond) {
		if ok() {
			return
		}
	}

	t.Fatalf("never happened: %s", what)
}

func TestTasksFollowTheTerm(t *testing.T) {
	e, a, b := newElector(), &task{name: "a"}, &task{name: "b"}
	_, lc := newRunner(t, e, a, b)

	time.Sleep(20 * time.Millisecond)

	if a.started.Load() != 0 || b.started.Load() != 0 {
		t.Fatal("tasks started on a standby")
	}

	for term := int32(1); term <= 3; term++ {
		e.promote <- struct{}{}

		eventually(t, "both tasks run", func() bool { return a.running.Load() == 1 && b.running.Load() == 1 })

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
}

func TestFailedTaskRestartsWithinTheTerm(t *testing.T) {
	boom := errors.New("boom")

	for name, run := range map[string]func(context.Context, func()) error{
		"error":        func(context.Context, func()) error { return boom },
		"panic":        func(context.Context, func()) error { panic("boom") },
		"early return": func(context.Context, func()) error { return nil },
	} {
		t.Run(name, func(t *testing.T) {
			e, bad, good := newElector(), &task{name: "bad", run: run}, &task{name: "good"}
			_, lc := newRunner(t, e, bad, good)

			e.promote <- struct{}{}

			eventually(t, "the failed task is restarted", func() bool { return bad.started.Load() >= 3 })

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
	}
}

func TestNoRestartAfterTheTerm(t *testing.T) {
	e := newElector()
	bad := &task{name: "bad", run: func(ctx context.Context, _ func()) error {
		<-ctx.Done()

		return errors.New("boom")
	}}
	_, lc := newRunner(t, e, bad)

	e.promote <- struct{}{}

	eventually(t, "the task runs", func() bool { return bad.running.Load() == 1 })

	e.demote <- struct{}{}

	<-e.stopped
	time.Sleep(30 * time.Millisecond)

	if got := bad.started.Load(); got != 1 {
		t.Fatalf("started %d times, want 1: restarted on a standby", got)
	}

	lc.RequireStop()
}

func TestRestartDelayEndsWithTheTerm(t *testing.T) {
	e := newElector()
	bad := &task{name: "bad", run: func(context.Context, func()) error { return errors.New("boom") }}

	lc := fxtest.NewLifecycle(t)
	r := New(e, []Task{bad}, slog.New(slog.NewTextHandler(io.Discard, nil)), lc)
	r.restartDelay = time.Hour

	lc.RequireStart()

	e.promote <- struct{}{}

	eventually(t, "the task fails", func() bool { return bad.started.Load() == 1 && bad.running.Load() == 0 })

	e.demote <- struct{}{}

	select {
	case <-e.stopped:
	case <-time.After(5 * time.Second):
		t.Fatal("the key is held for the restart delay of a failed task")
	}

	if got := bad.started.Load(); got != 1 {
		t.Fatalf("started %d times, want 1", got)
	}

	lc.RequireStop()
}

func TestTermWithoutTasksIsHeld(t *testing.T) {
	e := newElector()
	_, lc := newRunner(t, e)

	e.promote <- struct{}{}

	select {
	case <-e.stopped:
		t.Fatal("the term ended on its own")
	case <-time.After(30 * time.Millisecond):
	}

	lc.RequireStop()
}

func TestShutdownWaitsForTasks(t *testing.T) {
	e, release := newElector(), make(chan struct{})
	slow := &task{name: "slow", run: func(ctx context.Context, _ func()) error {
		<-ctx.Done()
		<-release

		return nil
	}}
	r, _ := newRunner(t, e, slow)

	e.promote <- struct{}{}

	eventually(t, "the task runs", func() bool { return slow.running.Load() == 1 })

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
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
}

func TestHealthCheck(t *testing.T) {
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
	r.tickStale = 50 * time.Millisecond

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

	eventually(t, "the task runs", func() bool { return ticking.running.Load() == 1 })
	check("")

	time.Sleep(2 * r.tickStale)
	check("leader task ticking: no tick")

	ticks <- struct{}{}

	eventually(t, "the tick is seen", func() bool { return r.HealthCheck(context.Background()) == nil })

	time.Sleep(2 * r.tickStale)
	check("no tick")

	e.demote <- struct{}{}

	<-e.stopped
	check("")

	lc.RequireStop()
	check("")
}

func TestHealthCheckOfARestartingTask(t *testing.T) {
	e := newElector()
	bad := &task{name: "bad", run: func(context.Context, func()) error { return errors.New("boom") }}
	r, lc := newRunner(t, e, bad)
	r.tickStale = 50 * time.Millisecond

	e.promote <- struct{}{}

	for deadline := time.Now().Add(4 * r.tickStale); time.Now().Before(deadline); time.Sleep(5 * time.Millisecond) {
		if err := r.HealthCheck(context.Background()); err != nil {
			t.Fatalf("a task that keeps restarting is not wedged: %v", err)
		}
	}

	lc.RequireStop()
}
