package cluster

import "time"

// Option is a functional option type for Cluster constructor.
type Option func(*Cluster)

// WithUpdateInterval sets interval between Cluster node updates.
func WithUpdateInterval(d time.Duration) Option {
	return func(cl *Cluster) {
		cl.updateInterval = d
	}
}

// WithUpdateTimeout sets ping timeout for update of each node in Cluster.
func WithUpdateTimeout(d time.Duration) Option {
	return func(cl *Cluster) {
		cl.updateTimeout = d
	}
}

// WithNodePicker sets algorithm for node selection (e.g., random, round-robin etc.).
func WithNodePicker(picker Picker) Option {
	return func(cl *Cluster) {
		cl.picker = picker
	}
}

// WithTracer sets tracer for actions happening in the background.
func WithTracer(tracer Tracer) Option {
	return func(cl *Cluster) {
		cl.tracer = tracer
	}
}

// WithUpdate decides whether to update node states.
// Useful for tests with mocked sql.DB.
func WithUpdate() Option {
	return func(cl *Cluster) {
		cl.update = true
	}
}
