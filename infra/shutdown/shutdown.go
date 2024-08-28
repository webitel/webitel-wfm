package shutdown

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"reflect"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/webitel/webitel-go-kit/logging/wlog"

	"github.com/webitel/webitel-wfm/infra/health"
)

type Tracker struct {
	log *wlog.Logger

	watchSignals bool

	timings processTimings

	initiated chan struct{} // closed when graceful shutdown is initiated
	once      sync.Once     // to trigger shutdown logic only once

	mu       sync.Mutex
	handlers map[string]Handler
}

func NewTracker(log *wlog.Logger, opts ...Option) *Tracker {
	timings := processTimings{
		keepAcceptingFor:     0,
		forceCloseTasksGrace: 1 * time.Second,
		forceShutdownGrace:   1 * time.Second,
	}

	for _, opt := range opts {
		opt.apply(&timings)
	}

	if timings.forceShutdownAfter <= 0 {
		timings.forceShutdownAfter = 5 * time.Second
	}

	timings.cancelRunningTasksAfter = timings.forceShutdownAfter - timings.forceCloseTasksGrace
	if timings.cancelRunningTasksAfter < 0 {
		timings.cancelRunningTasksAfter = 0
	}

	// If we know what the grace termination is for the kubernetes pods, we want to keep accepting new traffic
	// for almost all of that duration - minus what the Application runtime needs to perform a graceful shutdown.
	//
	// We'll immediately report a health failure when SIGTERM is received, however, we'll still accept new
	// traffic as we wait for routers and load balancers to update have propagated that we're trying
	// to cleanly shutdown.
	if timings.keepAcceptingFor-timings.forceShutdownAfter < 0 {
		timings.keepAcceptingFor = 0
	}

	t := &Tracker{
		log:          log,
		watchSignals: true,
		initiated:    make(chan struct{}),
		timings:      timings,
		handlers:     make(map[string]Handler),
	}

	return t
}

// WatchForShutdownSignals watches for shutdown signals (SIGTERM, SIGINT)
// and triggers the graceful shutdown when such a signal is received.
func (t *Tracker) WatchForShutdownSignals() {
	if !t.watchSignals {
		return
	}

	gracefulSignal := make(chan os.Signal, 1)
	signal.Notify(gracefulSignal, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		signalReceived := <-gracefulSignal
		t.Shutdown(signalReceived, nil)
	}()
}

// RegisterShutdownHandler registers a shutdown handler that will be called when the app
// is gracefully shutting down.
//
// The given context is closed when the graceful shutdown window is closed, and it's
// time to forcefully shut down. force.Deadline() can be inspected to learn when this
// will happen in advance.
//
// The shutdown is cooperative: the process will not exit until all shutdown hooks
// have returned, unless the process is forcefully killed by a signal (which may happen
// in certain cloud environments if the graceful shutdown takes longer than its timeout).
//
// If t is nil this function is a no-op.
func (t *Tracker) RegisterShutdownHandler(name string, h Handler) error {
	if t == nil {
		return fmt.Errorf("shutdown tracker doesn't configured")
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	if _, ok := t.handlers[name]; ok {
		return fmt.Errorf("shutdown tracker already registered: %s", name)
	}

	t.handlers[name] = h

	return nil
}

func (t *Tracker) RegisterShutdownHandlerFunc(name string, f HandlerFunc) error {
	return t.RegisterShutdownHandler(name, f)
}

// ShutdownInitiated reports whether graceful shutdown has been initiated.
func (t *Tracker) ShutdownInitiated() bool {
	select {
	case <-t.initiated:
		return true
	default:
		return false
	}
}

// HealthCheck returns a health check failure once a SIGTERM has been received.
//
// This is to allow load balancers to detect this instance is shutting down
// and should not be routed to for new traffic.
func (t *Tracker) HealthCheck(_ context.Context) []health.CheckResult {
	var reportError error
	if t.ShutdownInitiated() {
		reportError = errors.New("SIGTERM has been received, graceful shutdown started")
	}

	return []health.CheckResult{{
		Name: "shutdown-signal-monitoring",
		Err:  reportError,
	}}
}

// Shutdown triggers the shutdown logic.
// If it has already been triggered, it does nothing and returns immediately.
func (t *Tracker) Shutdown(reasonSignal os.Signal, reasonError error) {
	t.once.Do(func() {
		close(t.initiated)

		if reasonError != nil {
			t.log.Error("a fatal error occurred, initiating graceful shutdown", wlog.Err(reasonError))
		}

		if reasonSignal != nil {
			t.log.Info("got shutdown signal, initiating graceful shutdown", wlog.String("signal", reasonSignal.String()))
		}

		// If we received a SIGTERM and have a configured keepAcceptingFor duration,
		// then log the fact we're going to continue accepting new requests and then
		// sleep for that time before begining to graceful shutdown.
		if reasonSignal == syscall.SIGTERM && t.timings.keepAcceptingFor > 0 {
			t.log.Info("continuing to accept requests for a short period of time to allow the load balancer to update", wlog.String("duration", t.timings.keepAcceptingFor.String()))

			time.Sleep(t.timings.keepAcceptingFor)
			t.log.Info("stopping to accept new requests and continuing graceful shutdown")
		}

		p := t.beginShutdownProcess()
		go t.runShutdownHandlers(p)
		t.exitOnCompletion(p)
	})
}

func (t *Tracker) beginShutdownProcess() *Process {
	start := time.Now()

	tt := t.timings
	outstandingTasks, cancelOutstandingTasks := context.WithDeadline(context.Background(), start.Add(tt.cancelRunningTasksAfter+tt.forceCloseTasksGrace))
	outstandingRequests, cancelOutstandingRequests := context.WithCancel(outstandingTasks)
	outstandingPubSubMessages, cancelOutstandingPubSubMessages := context.WithCancel(outstandingTasks)

	forceCloseTasks, cancelForceCloseTasks := context.WithDeadline(outstandingTasks, start.Add(tt.cancelRunningTasksAfter))

	forceShutdown, cancelForceShutdown := context.WithDeadline(context.Background(), start.Add(tt.forceShutdownAfter))

	serviceShutdownCompleted, cancelServiceShutdownCompleted := context.WithCancelCause(context.Background())
	handlersCompleted, cancelHandlersCompleted := context.WithCancelCause(context.Background())

	shutdownCompleted, cancelShutdownCompleted := context.WithCancelCause(context.Background())

	// Close the runningHandlers context when both
	// outstandingRequests and outstandingPubSubMessages are done.
	go func() {
		<-outstandingRequests.Done()
		<-outstandingPubSubMessages.Done()
		cancelOutstandingTasks()

		// This is redundant (the context is derived from runningTasks),
		// but it makes the linter happy.
		cancelForceCloseTasks()
	}()

	// Cancel forceShutdown early if running tasks and handlers complete.
	go func() {
		<-outstandingTasks.Done()
		<-handlersCompleted.Done()
		cancelForceShutdown()
	}()

	// Mark the shutdown completed.
	go func() {
		<-forceShutdown.Done()

		// When forceShutdown is done, see if it was due to reaching the deadline (unclean shutdown)
		// or if we canceled the context early (clean shutdown).
		if errors.Is(forceShutdown.Err(), context.Canceled) {
			cancelShutdownCompleted(cleanShutdown)

			return
		} else {
			// We reached the deadline. The ForceShutdown context was canceled just now,
			// so give it another second to let the shutdown handlers finish.
			timeout, cancel := context.WithTimeout(handlersCompleted, tt.forceShutdownGrace)
			defer cancel()
			<-timeout.Done()

			if errors.Is(timeout.Err(), context.Canceled) {
				// The handlers did eventually complete, so this is a clean shutdown.
				cancelShutdownCompleted(cleanShutdown)
			} else {
				cancelShutdownCompleted(timeout.Err())
			}
		}
	}()

	return &Process{
		Log:                       t.log,
		OutstandingRequests:       outstandingRequests,
		cancelOutstandingRequests: cancelOutstandingRequests,

		OutstandingPubSubMessages:       outstandingPubSubMessages,
		cancelOutstandingPubSubMessages: cancelOutstandingPubSubMessages,

		OutstandingTasks:       outstandingTasks,
		cancelOutstandingTasks: cancelOutstandingTasks,

		ForceCloseTasks:       forceCloseTasks,
		cancelForceCloseTasks: cancelForceCloseTasks,

		ForceShutdown:       forceShutdown,
		cancelForceShutdown: cancelForceShutdown,

		ServicesShutdownCompleted:     serviceShutdownCompleted,
		markServicesShutdownCompleted: cancelServiceShutdownCompleted,

		handlersCompleted:     handlersCompleted,
		markHandlersCompleted: cancelHandlersCompleted,

		ShutdownCompleted:     shutdownCompleted,
		markShutdownCompleted: cancelShutdownCompleted,
	}
}

// runShutdownHandlers runs the registered shutdown handlers.
func (t *Tracker) runShutdownHandlers(p *Process) {
	var (
		shutdownErrorMu sync.Mutex
		shutdownErrs    []error
	)

	addShutdownErr := func(err shutdownError) {
		shutdownErrorMu.Lock()
		defer shutdownErrorMu.Unlock()
		shutdownErrs = append(shutdownErrs, err)
	}

	// Mark the handlers as completed when we're done.
	defer func() {
		shutdownErrorMu.Lock()
		errList := shutdownErrs
		shutdownErrorMu.Unlock()

		// Determine the error to use.
		var shutdownErr error
		if len(errList) > 0 {
			shutdownErr = shutdownErrors{errors: errList}
		}

		t.log.Debug("all shutdown hooks completed", wlog.Err(shutdownErr))
		if shutdownErr != nil {
			p.markHandlersCompleted(shutdownErr)
		} else {
			p.markHandlersCompleted(cleanShutdown)
		}
	}()

	t.mu.Lock()
	handlers := t.handlers
	t.mu.Unlock()

	var wg sync.WaitGroup
	wg.Add(len(handlers))

	for name, fn := range handlers {
		fn := fn
		go func() {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					err := shutdownError{
						handlerName: name,
						err:         fmt.Errorf("panic: %s", r),
					}

					addShutdownErr(err)
					t.log.Error(fmt.Sprintf("panic encountered during shutdown hook: %s", debug.Stack()), wlog.Err(err))
				}
			}()

			defer t.log.Debug("shutdown hook completed", wlog.String("hook", name))
			t.log.Debug("running shutdown hook...", wlog.String("hook", name))
			if err := fn.Shutdown(p); err != nil {
				shutdownErr := shutdownError{handlerName: name, err: err}
				t.log.Error("shutdown handler returned an error", wlog.Err(shutdownErr), wlog.String("hook", name))
				addShutdownErr(shutdownErr)
			}
		}()
	}

	wg.Wait()
}

// exitOnCompletion exits the process when the shutdown is completed.
func (t *Tracker) exitOnCompletion(p *Process) {
	<-p.ShutdownCompleted.Done()

	if p.WasCleanShutdown() {
		t.log.Debug("graceful shutdown completed")
		os.Exit(0)
	} else {
		t.log.Debug("graceful shutdown window closed, forcing shutdown")
		os.Exit(1)
	}
}

func functionName(fn any) (rtn string) {
	defer func() {
		if r := recover(); r != nil && rtn == "" {
			rtn = fmt.Sprintf("<panic getting function name: %v>", r)
		}
	}()

	return strings.TrimSuffix(runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name(), "-fm")
}
