package shutdown

// Handler is the interface for resources that participate in the graceful shutdown process.
type Handler interface {
	// Shutdown is called by Application when the graceful shutdown process is initiated.
	//
	// The provided Progress struct provides information about the graceful shutdown progress,
	// which can be used to determine at what point in time it's appropriate to close certain resources.
	//
	// For example, a service struct may want to wait for all incoming requests to complete
	// before it closes its client to a third-party service:
	//
	// 			func (s *MyService) Shutdown(p *shutdown.Process) error {
	//				<-p.OutstandingRequests.Done()
	//				return s.client.Close()
	//			}
	//
	// The shutdown process is cooperative (to the extent it is possible),
	// and Application will wait for all Handlers to return before closing
	// infrastructure resources and exiting the process,
	// until the ForceShutdown deadline is reached.
	//
	// The return value of Shutdown is used to report shutdown errors only,
	// and has no effect on the shutdown process.
	Shutdown(p *Process) error
}

// HandlerFunc is a type that implements the Handler interface.
type HandlerFunc func(p *Process) error

func (h HandlerFunc) Shutdown(p *Process) error {
	return h(p)
}
