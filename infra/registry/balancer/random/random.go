package random

import (
	"context"
	"math/rand"

	"github.com/webitel/webitel-wfm/infra/registry"
	"github.com/webitel/webitel-wfm/infra/registry/node/direct"
)

const (
	// Name is random balancer name
	Name = "random"
)

var _ registry.Balancer = (*Balancer)(nil) // Name is balancer name

// Option is random builder option.
type Option func(o *options)

// options is random builder options
type options struct{}

// Balancer is a random balancer.
type Balancer struct{}

// New a random selector.
func New(opts ...Option) registry.Selector {
	return NewBuilder(opts...).Build()
}

// Pick is pick a weighted node.
func (p *Balancer) Pick(_ context.Context, nodes []registry.WeightedNode) (registry.WeightedNode, registry.DoneFunc, error) {
	if len(nodes) == 0 {
		return nil, nil, registry.ErrNoAvailable
	}

	cur := rand.Intn(len(nodes))
	selected := nodes[cur]
	d := selected.Pick()

	return selected, d, nil
}

// NewBuilder returns a selector builder with random balancer
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

// Builder is random builder
type Builder struct{}

// Build creates Balancer
func (b *Builder) Build() registry.Balancer {
	return &Balancer{}
}
