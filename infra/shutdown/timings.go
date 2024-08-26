package shutdown

import (
	"time"
)

type processTimings struct {
	// keepAcceptingFor is the duration from the moment we receive a SIGTERM
	// after which we stop accepting new requests. However, we will
	// report being unhealthy to the load balancer immediately.
	//
	// This is necessary as in a Kubernetes environment, the pod sent a SIGTERM
	// once its replacement is ready, however, it will take some time for that
	// to propagate to the load balancer. If we stop accepting requests immediately,
	// we will have a period of time when the load balancer still sends
	// requests to the pod, which will be rejected. This will cause the load
	// balancer to report 502 errors.
	//
	// See: https://cloud.google.com/kubernetes-engine/docs/how-to/container-native-load-balancing#traffic_does_not_reach_endpoints
	keepAcceptingFor time.Duration

	// cancelRunningTasksAfter is the duration (measured from shutdown initiation)
	// after which running tasks (outstanding API calls & PubSub messages) have
	// their contexts canceled.
	cancelRunningTasksAfter time.Duration

	// forceCloseTasksGrace is the duration (measured from when canceling running tasks)
	// after which the tasks are considered done, even if they're still running.
	forceCloseTasksGrace time.Duration

	// forceShutdownAfter is the duration (measured from shutdown initiation)
	// after which the shutdown process enters the "force shutdown" phase,
	// tearing down infrastructure resources.
	forceShutdownAfter time.Duration

	// forceShutdownGrace is the grace period after beginning the force shutdown
	// before the shutdown is marked as completed, causing the process to exit.
	forceShutdownGrace time.Duration
}
