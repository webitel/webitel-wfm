package shutdown

import (
	"context"
	"errors"

	"github.com/webitel/webitel-go-kit/logging/wlog"
)

// Process provides progress information about an ongoing graceful shutdown process.
//
// The process broadly consists of two phases:
//
// 1. Drain active tasks
//
// As soon as the graceful shutdown process is initiated, the service will stop accepting new
// incoming API calls and Pub/Sub messages. It will continue to process already running tasks
// until they complete (or the ForceCloseTasks deadline is reached).
//
// Additionally, all service structs that implement [Handler] will have their [Handler.Shutdown]
// function called when this phase begins. The [Handler.Shutdown] method receives a [Progress]
// struct that can be used to monitor the progress of the shutdown process, and allows the service
// to perform any necessary cleanup at the right time.
//
// This phase continues until all active tasks and handlers have completed or the ForceCloseTasks deadline
// is reached, whichever happens first. The OutstandingRequests, OutstandingPubSubMessages, and
// OutstandingTasks contexts provide insight into what tasks are still active.
//
// 2. Shut down infrastructure resources
//
// When all active tasks and [Handler.Shutdown] calls have completed, the Application begins shutting down
// infrastructure resources. The Application automatically closes all open database connections, cache connections,
// Pub/Sub connections, and other infrastructure resources.
//
// This phase continues until all infrastructure resources have been closed or the ForceShutdown deadline
// is reached, whichever happens first.
//
// 3. Exit
//
// Once phase two has completed, the process will exit.
// The exit code is 0 if the graceful shutdown is completed successfully (meaning all resources
// returned before the exit deadline), or 1 otherwise.
type Process struct {
	Log *wlog.Logger

	// OutstandingRequests is canceled when the service is no longer processing any incoming API calls.
	OutstandingRequests       context.Context
	cancelOutstandingRequests context.CancelFunc

	// OutstandingPubSubMessages is canceled when the service is no longer processing any Pub/Sub messages.
	OutstandingPubSubMessages       context.Context
	cancelOutstandingPubSubMessages context.CancelFunc

	// OutstandingTasks is canceled when the service is no longer actively processing any tasks,
	// which includes both incoming API calls and Pub/Sub messages.
	//
	// It is canceled as soon as both OutstandingRequests and OutstandingPubSubMessages have been canceled.
	OutstandingTasks       context.Context
	cancelOutstandingTasks context.CancelFunc

	// ForceCloseTasks is canceled when the graceful shutdown deadline is reached and it's time to
	// forcibly close active tasks (outstanding incoming API requests and Pub/Sub subscription messages).
	//
	// When ForceCloseTasks is closed, the contexts for all outstanding tasks are canceled.
	//
	// It is canceled early if all active tasks are done.
	ForceCloseTasks       context.Context
	cancelForceCloseTasks context.CancelFunc

	ServicesShutdownCompleted     context.Context
	markServicesShutdownCompleted context.CancelCauseFunc

	// ForceShutdown is closed when the graceful shutdown window has closed and it's time to
	// forcefully shut down.
	//
	// If the graceful shutdown window lapses before the cooperative shutdown is complete,
	// the ForceShutdown channel may be closed before RunningHandlers is canceled.
	//
	// It is canceled early if all running tasks have completed, all infrastructure resources are closed,
	// and all registered service Handler.Shutdown methods have returned.
	ForceShutdown       context.Context
	cancelForceShutdown context.CancelFunc

	handlersCompleted     context.Context
	markHandlersCompleted context.CancelCauseFunc

	// ShutdownCompleted is closed when all shutdown hooks have returned.
	ShutdownCompleted     context.Context
	markShutdownCompleted context.CancelCauseFunc
}

func (p *Process) MarkOutstandingRequestsCompleted() {
	p.cancelOutstandingRequests()
}

func (p *Process) MarkOutstandingPubSubMessagesCompleted() {
	p.cancelOutstandingPubSubMessages()
}

func (p *Process) MarkServicesShutdownCompleted(err error) {
	// TODO change error type to capture where the service came from
	if err != nil {
		p.markServicesShutdownCompleted(err)
	} else {
		p.markServicesShutdownCompleted(cleanShutdown)
	}
}

// WasCleanShutdown reports whether the shutdown was clean.
// Its return value is undefined before p.shutdownCompleted is closed.
func (p *Process) WasCleanShutdown() bool {
	return errors.Is(context.Cause(p.ShutdownCompleted), cleanShutdown)
}
