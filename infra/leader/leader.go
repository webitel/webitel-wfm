// Package leader runs leader-only tasks on the instance elected through consul.
package leader

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/fx"

	"github.com/webitel/webitel-go-kit/infra/discovery/consul"
)

// TaskGroup is the fx group the runner takes its tasks from.
const TaskGroup = `group:"leader_tasks"`

// Task serves one leadership term and returns once ctx is canceled. It calls tick on every turn
// of its loop, and at least once a minute while a turn runs long: without ticks the leader is unhealthy.
type Task interface {
	Name() string
	Run(ctx context.Context, tick func()) error
}

type Runner struct {
	elector consul.LeadershipElector
	tasks   []Task
	log     *slog.Logger

	restartDelay time.Duration
	tickStale    time.Duration

	// Unix nanoseconds of each task's last tick; zero outside a term.
	ticks []atomic.Int64

	stop context.CancelFunc
	done chan struct{}
}

func New(elector consul.LeadershipElector, tasks []Task, log *slog.Logger, lc fx.Lifecycle) *Runner {
	r := &Runner{
		elector:      elector,
		tasks:        tasks,
		log:          log.With("component", "leader"),
		restartDelay: 5 * time.Second,
		tickStale:    time.Minute,
		ticks:        make([]atomic.Int64, len(tasks)),
		done:         make(chan struct{}),
	}

	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			ctx, stop := context.WithCancel(context.Background())
			r.stop = stop

			go func() {
				defer close(r.done)

				r.elector.Run(ctx, r.lead, func() { r.log.Warn("leader tasks stopped") })
			}()

			return nil
		},
		OnStop: r.shutdown,
	})

	return r
}

// shutdown ends the election; the elector stops the tasks before releasing the key.
func (r *Runner) shutdown(ctx context.Context) error {
	r.stop()

	select {
	case <-r.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// lead never fails: a task error must not end the term, only the loss of the consul session does.
func (r *Runner) lead(ctx context.Context) error {
	r.log.Info("leader tasks started", "tasks", len(r.tasks))

	var wg sync.WaitGroup
	for i, task := range r.tasks {
		wg.Go(func() { r.serve(ctx, task, &r.ticks[i]) })
	}

	<-ctx.Done()
	wg.Wait()

	return nil
}

func (r *Runner) serve(ctx context.Context, task Task, last *atomic.Int64) {
	defer last.Store(0)

	tick := func() { last.Store(time.Now().UnixNano()) }

	for {
		tick()

		err := run(ctx, task, tick)
		if ctx.Err() != nil {
			return
		}

		r.log.Error("leader task failed, restarting", "task", task.Name(), "delay", r.restartDelay, "err", err)

		select {
		case <-time.After(r.restartDelay):
		case <-ctx.Done():
			return
		}
	}
}

func run(ctx context.Context, task Task, tick func()) (err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("panic: %v\n%s", p, debug.Stack())
		}
	}()

	if err := task.Run(ctx, tick); err != nil {
		return err
	}

	return errors.New("returned before the term ended")
}

// HealthCheck fails when a task of this term stopped ticking. A standby has no ticks, so it
// stays healthy; otherwise every instance but the leader would be restarted.
func (r *Runner) HealthCheck(context.Context) error {
	for i, task := range r.tasks {
		last := r.ticks[i].Load()
		if last == 0 {
			continue
		}

		if idle := time.Since(time.Unix(0, last)); idle > r.tickStale {
			return fmt.Errorf("leader task %s: no tick for %s", task.Name(), idle.Round(time.Second))
		}
	}

	return nil
}
