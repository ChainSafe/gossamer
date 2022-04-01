package node

import (
	"context"
	"fmt"
	"testing"
)

// Nodes is a slice of nodes.
type Nodes []Node

// MakeNodes creates `num` nodes using the `baseNode`
// as a base for each node. It sets the following fields:
// - the first node is always the BABE lead
// - the index of each node is incremented per node
// - the base path is set to a test temporary directory
// - remaining unset fields are set to their default.
func MakeNodes(t *testing.T, num int, options ...Option) (nodes Nodes) {
	nodes = make(Nodes, num)
	for i := range nodes {
		// Set fields using options given
		for _, option := range options {
			option(&nodes[i])
		}

		// Set defaults using index `i`
		if nodes[i].babeLead == nil {
			nodes[i].babeLead = boolPtr(i == 0)
		}
		if nodes[i].index == nil {
			nodes[i].index = intPtr(i)
		}

		// Set node defaults on the remaining unset fields
		nodes[i].setDefaults(t)
		nodes[i].setWriterPrefix()
	}
	return nodes
}

// Init initialises all nodes and returns an error if any
// init operation failed.
func (nodes Nodes) Init(ctx context.Context) (err error) {
	for _, node := range nodes {
		err := node.Init(ctx)
		if err != nil {
			return fmt.Errorf("failed to initialise node %s: %w",
				node, err)
		}
	}

	return nil
}

// Start starts all the nodes and returns the number of started nodes
// and an eventual start error. The started number should be used by
// the caller to wait for `started` errors coming from the wait error
// channel. All the nodes are stopped when the context is canceled,
// and `started` errors will be sent in the waitErr channel.
func (nodes Nodes) Start(ctx context.Context, waitErr chan<- error) (
	started int, startErr error) {
	for _, node := range nodes {
		err := node.Start(ctx, waitErr)
		if err != nil {
			return started, fmt.Errorf("failed to start node with index %d: %w",
				node.index, err)
		}

		started++
	}

	return started, nil
}

// InitAndStartTest is a test helper method to initialise and start nodes,
// as well as registering appriopriate test handlers.
// It monitors each node for failure, and terminates all of them if any fails.
// It also shuts down all the nodes on test cleanup.
// Finally it calls the passed stop signalling functional argument when the
// test should be failed because the nodes got terminated.
func (nodes Nodes) InitAndStartTest(ctx context.Context, t *testing.T,
	signalTestToStop context.CancelFunc) {
	t.Helper()

	for _, node := range nodes {
		t.Logf("Node %s initialising", node)
		err := node.Init(ctx)
		if err != nil {
			t.Errorf("node %s failed to initialise: %s", node, err)
			signalTestToStop()
			return
		}
	}

	var started int
	nodesCtx, nodesCancel := context.WithCancel(ctx)
	waitErr := make(chan error)

	for _, node := range nodes {
		t.Logf("Node %s starting", node)
		err := node.Start(nodesCtx, waitErr)
		if err == nil {
			t.Logf("Node %s started", node)
			started++
			continue
		}

		t.Errorf("Node %s failed to start: %s", node, err)

		stopNodes(t, started, nodesCancel, waitErr)
		close(waitErr)

		// Signal calling test to stop.
		signalTestToStop()
		return
	}

	// watch for runtime fatal error from any of the nodes
	watchDogCtx, watchDogCancel := context.WithCancel(ctx)
	watchDogDone := make(chan struct{})
	go func() {
		defer close(watchDogDone)
		select {
		case <-watchDogCtx.Done():
			return
		case err := <-waitErr: // one node crashed
			if watchDogCtx.Err() != nil {
				// make sure the runtime watchdog is not meant
				// to be disengaged, in case of signal racing.
				return
			}

			t.Errorf("one node has crashed: %s", err)
			started--

			stopNodes(t, started, nodesCancel, waitErr)
			close(waitErr)

			// we cannot stop the test with t.FailNow() from a goroutine
			// other than the test goroutine, so we call failNow to signal
			// it to the test goroutine.
			signalTestToStop()
		}
	}()

	t.Cleanup(func() {
		t.Helper()
		// Disengage node watchdog goroutine
		watchDogCancel()
		<-watchDogDone
		// Stop and wait for nodes to exit
		stopNodes(t, started, nodesCancel, waitErr)
		close(waitErr)
	})
}

func stopNodes(t *testing.T, started int,
	nodesCancel context.CancelFunc, waitErr <-chan error) {
	t.Helper()

	// Stop the nodes and wait for them to exit
	nodesCancel()
	t.Logf("waiting on %d nodes to terminate...", started)
	for i := 0; i < started; i++ {
		err := <-waitErr
		t.Logf("Node has terminated: %s", err)
	}
}
