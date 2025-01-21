package cluster

import "github.com/webitel/webitel-wfm/infra/storage/dbsql"

// updateSubscriber represents a waiter for newly checked node event.
type updateSubscriber struct {
	ch        chan dbsql.Node
	criterion NodeStateCriterion
}

// addUpdateSubscriber adds new subscriber to notification pool.
func (cl *Cluster) addUpdateSubscriber(criterion NodeStateCriterion) <-chan dbsql.Node {
	// buffered channel is essential
	// read WaitForNode function for more information
	ch := make(chan dbsql.Node, 1)
	cl.subscribersMu.Lock()
	defer cl.subscribersMu.Unlock()
	cl.subscribers = append(cl.subscribers, updateSubscriber{ch: ch, criterion: criterion})

	return ch
}

// notifyUpdateSubscribers sends appropriate nodes to registered subscribers.
// This function uses newly checked nodes to avoid race conditions.
func (cl *Cluster) notifyUpdateSubscribers(nodes CheckedNodes) {
	cl.subscribersMu.Lock()
	defer cl.subscribersMu.Unlock()

	if len(cl.subscribers) == 0 {
		return
	}

	var nodelessWaiters []updateSubscriber
	for _, subscriber := range cl.subscribers {
		node := pickNodeByCriterion(nodes, cl.picker, subscriber.criterion)
		if node == nil {
			nodelessWaiters = append(nodelessWaiters, subscriber)

			continue
		}

		// We won't block here, read addUpdateWaiter function for more information
		subscriber.ch <- node

		// No need to close a channel since we write only once and forget it so does the 'client'
	}

	cl.subscribers = nodelessWaiters
}
