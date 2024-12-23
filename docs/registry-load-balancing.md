# Routing and Load Balancing

## Interface Implementation

The main interface for routing and load balancing is Selector, and a default Selector implementation is also provided in the same directory. 
This implementation can implement node weight calculation algorithm, service routing filtering strategy, and load balancing algorithm by 
replacing **NodeBuilder**, **Filter**, **Balancer**, and Pluggable.

```go
type Selector interface {
    // The list of service nodes maintained internally by the Selector is updated through the Rebalancer interface.
    Rebalancer

    // Select nodes.
    // if err == nil, selected and done must not be empty.
    Select(ctx context.Context, opts ...SelectOption) (selected Node, done DoneFunc, err error)
}

// Realize service node change awareness through Rebalancer.
type Rebalancer interface {
    Apply(nodes []Node)
}
```

Supported implementations:

- [wrr](https://github.com/webitel/webitel-wfm/tree/main/infra/registry/wrr) : Weighted round robin
- [p2c](https://github.com/webitel/webitel-wfm/tree/main/infra/registry/p2c) : Power of two choices
- [random](https://github.com/webitel/webitel-wfm/tree/main/infra/registry/random) : Random

## How to use

### gRPC Client

```go
package main

import (
	"google.golang.org/grpc"

	"github.com/webitel/webitel-wfm/infra/registry"
	"github.com/webitel/webitel-wfm/infra/registry/balancer/wrr"
	"github.com/webitel/webitel-wfm/infra/registry/resolver"
)

func main() {
	// Due to the limitations of the gRPC framework, 
	// only the global balancer name can be used to inject Selector.
	registry.SetGlobalSelector(wrr.NewBuilder())

	_, err := grpc.NewClient("discovery:///webitel-wfm", grpc.WithResolvers(resolver.NewBuilder(log, discovery, resolver.WithInsecure(true))))
	if err != nil {
		panic(err)
	}
}
```