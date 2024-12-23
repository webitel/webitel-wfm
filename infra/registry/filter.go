package registry

import "context"

// NodeFilter is select filter.
type NodeFilter func(context.Context, []Node) []Node

// Version is version filter.
func Version(version string) NodeFilter {
	return func(_ context.Context, nodes []Node) []Node {
		newNodes := make([]Node, 0, len(nodes))
		for _, n := range nodes {
			if n.Version() == version {
				newNodes = append(newNodes, n)
			}
		}

		return newNodes
	}
}
