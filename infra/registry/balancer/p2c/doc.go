// Package p2c helps you to to select two nodes randomly from all available nodes and then select a less
// loaded node based on the load of these two nodes (in other words - "Power of Two").
// Algorithm is an improved random algorithm that avoids the worst selection and load imbalances.
// The P2C approach is not as effective on a single load balancer, but it deftly avoids
// the bad‑case “herd behavior” that can occur when you scale out to a number of independent load balancers.
package p2c
