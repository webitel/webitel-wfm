// Package ewma (Exponentially-Weighted Moving Average) maintain a moving average of each replica’s round-trip time,
// weighted by the number of outstanding requests, and distribute traffic to replicas where that cost function is smallest.
package ewma
