// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

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
			return started, fmt.Errorf("node with index %d: %w",
				*node.index, err)
		}

		started++
	}

	for _, node := range nodes {
		port := node.GetRPCPort()
		err := waitForNode(ctx, port)
		if err != nil {
			return started, fmt.Errorf("node with index %d: %w", *node.index, err)
		}
	}

	return started, nil
}

// InitAndStartTest is a test helper method to initialise and start nodes,
// as well as registering appriopriate test handlers.
// If any node fails to initialise or start, cleanup is done and the test
// is instantly failed.
// If any node crashes at runtime, all other nodes are shutdown,
// cleanup is done and the passed argument `signalTestToStop`
// is called to signal to the main test goroutine to stop.
func (nodes Nodes) InitAndStartTest(ctx context.Context, t *testing.T,
	signalTestToStop context.CancelFunc) {
	t.Helper()

	initErrors := make(chan error)
	for _, node := range nodes {
		go func(node Node) {
			err := node.Init(ctx) // takes 2 seconds
			if err != nil {
				err = fmt.Errorf("node %s failed to initialise: %w", node, err)
			}
			initErrors <- err
		}(node)
	}

	for range nodes {
		err := <-initErrors
		if err != nil {
			t.Error(err)
		}
	}
	if t.Failed() {
		t.FailNow()
	}

	var started int
	nodesCtx, nodesCancel := context.WithCancel(ctx)
	waitErr := make(chan error)

	for _, node := range nodes {
		err := node.Start(nodesCtx, waitErr) // takes little time
		if err == nil {
			started++
			continue
		}

		t.Errorf("Node %s failed to start: %s", node, err)

		stopNodes(t, started, nodesCancel, waitErr)
		close(waitErr)
		t.FailNow()
	}

	// this is run sequentially since all nodes start almost at the same time
	// so waiting for one node will also wait for all the others.
	// You can see this since the test logs out that all the nodes are ready
	// at the same time.
	for _, node := range nodes {
		err := waitForNode(ctx, node.GetRPCPort())
		if err == nil {
			t.Logf("Node %s is ready", node)
			continue
		}

		t.Errorf("Node %s failed to be ready: %s", node, err)
		stopNodes(t, started, nodesCancel, waitErr)
		close(waitErr)
		t.FailNow()
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
		<-waitErr
	}
}
