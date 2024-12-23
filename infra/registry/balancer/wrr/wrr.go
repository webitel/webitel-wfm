package wrr

import (
	"context"
	"sync"

	"github.com/webitel/webitel-wfm/infra/registry"
	"github.com/webitel/webitel-wfm/infra/registry/node/direct"
)

const (
	// Name is wrr(Weighted Round Robin) balancer name
	Name = "wrr"
)

var _ registry.Balancer = (*Balancer)(nil) // Name is balancer name

// Option is wrr builder option.
type Option func(o *options)

// options is wrr builder options
type options struct{}

// Balancer is a wrr balancer.
type Balancer struct {
	mu            sync.Mutex
	currentWeight map[string]float64
}

// New random a selector.
func New(opts ...Option) registry.Selector {
	return NewBuilder(opts...).Build()
}

// Pick is pick a weighted node.
func (p *Balancer) Pick(_ context.Context, nodes []registry.WeightedNode) (registry.WeightedNode, registry.DoneFunc, error) {
	if len(nodes) == 0 {
		return nil, nil, registry.ErrNoAvailable
	}

	var totalWeight float64
	var selected registry.WeightedNode
	var selectWeight float64

	// nginx wrr load balancing algorithm: http://blog.csdn.net/zhangskd/article/details/50194069
	p.mu.Lock()
	for _, node := range nodes {
		totalWeight += node.Weight()
		cwt := p.currentWeight[node.Address()]
		// current += effectiveWeight
		cwt += node.Weight()
		p.currentWeight[node.Address()] = cwt
		if selected == nil || selectWeight < cwt {
			selectWeight = cwt
			selected = node
		}
	}

	p.currentWeight[selected.Address()] = selectWeight - totalWeight
	p.mu.Unlock()

	d := selected.Pick()

	return selected, d, nil
}

// NewBuilder returns a selector builder with wrr balancer
func NewBuilder(opts ...Option) registry.Builder {
	var option options
	for _, opt := range opts {
		opt(&option)
	}

	return &registry.DefaultBuilder{
		Balancer: &Builder{},
		Node:     &direct.Builder{},
	}
}

// Builder is wrr builder
type Builder struct{}

// Build creates Balancer
func (b *Builder) Build() registry.Balancer {
	return &Balancer{currentWeight: make(map[string]float64)}
}
