# Graceful shutdown

Graceful shutdown is initiated when the ```webitel-wfm``` receive ```SIGTERM``` or ```SIGINT``` signals. The process 
broadly consists of three phases:

1. **Drain active tasks.**

   Services impacted: Consul, gRPC server, pubsub fetching.

   As soon as the graceful shutdown process is initiated, the service will deregister in Consul and stop accepting 
   new incoming API calls (from gRPC server) and Pub/Sub messages. It will continue to process already running tasks 
   until they complete (or the ForceCloseTasks deadline is reached).

   Additionally, all services that implement graceful shutdown handler will have their shutdown
   function called when this phase begins. The shutdown method receives a progress that can be used to monitor the 
   progress of the shutdown process, and allows the service to perform any necessary cleanup at the right time.

   This phase continues until all active tasks and handlers have completed or the ```ForceCloseTasks``` deadline
   is reached, whichever happens first. The ```OutstandingRequests```, ```OutstandingPubSubMessages```, and
   ```OutstandingTasks``` contexts provide insight into what tasks are still active.

2. **Shut down infrastructure resources.**
   
   Services impacted: Webitel services, database, pubsub connection.

   When all active tasks and external service shutdown calls have completed, ```webitel-wfm``` begins shutting down 
   infrastructure resources. ```webitel-wfm``` automatically closes all open database, cache and Pub/Sub connections, 
   other infrastructure resources.
    
   This phase continues until all infrastructure resources have been closed or the ForceShutdown deadline
   is reached, whichever happens first.

3. **Exit.** 

   Once phase two has completed, the process will exit. 

   The exit code is 0 if the graceful shutdown is completed successfully (meaning all resources returned before the 
   exit deadline), or 1 otherwise.

Graceful shutdown configured using options:

| Option                        | Env | Flag                          | Default                                                            | Description                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                               |  
|-------------------------------|-----|-------------------------------|--------------------------------------------------------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|  
| ```keepAcceptingFor```        |     | ```keep-accepting```          | 0s                                                                 | Duration from the moment we receive a SIGTERM after which we stop accepting new requests. However, we will report being unhealthy to the load balancer immediately. This is necessary as in a Kubernetes environment, the pod sent a SIGTERM once its replacement is ready, however, it will take some time for that to propagate to the load balancer. If we stop accepting requests immediately, we will have a period of time when the load balancer still sends requests to the pod, which will be rejected. This will cause the load balancer to report 502 errors. See: [Traffic does not reach endpoints](https://cloud.google.com/kubernetes-engine/docs/how-to/container-native-load-balancing#traffic_does_not_reach_endpoints) |
| ```cancelRunningTasksAfter``` |     |                               | greater(```forceShutdownAfter``` - ```forceCloseTasksGrace```, 0s) | Duration (measured from shutdown initiation) after which running tasks (outstanding API calls & PubSub messages) have their contexts canceled.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                            |
| ```forceCloseTasksGrace```    |     | ```force-close-tasks-grace``` | 1s                                                                 | Duration (measured from when canceling running tasks) after which the tasks are considered done, even if they're still running.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                           |
| ```forceShutdownAfter```      |     |                               | 5s                                                                 | Duration (measured from shutdown initiation) after which the shutdown process enters the "force shutdown" phase, tearing down infrastructure resources.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                   |
| ```forceShutdownGrace```      |     | ```force-shutdown-grace```    | 1s                                                                 | Grace period after beginning the force shutdown before the shutdown is marked as completed, causing the process to exit.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  |
