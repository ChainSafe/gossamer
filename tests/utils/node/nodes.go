// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	cfg "github.com/ChainSafe/gossamer/config"
)

// Nodes is a slice of nodes.
type Nodes []Node

// MakeNodes creates `num` nodes using the `tomlConfig`
// as a base config for each node. It overrides some of configuration:
// - the index of each node is incremented per node (overrides the SetIndex option, if set)
func MakeNodes(t *testing.T, num int, tomlConfig cfg.Config,
	options ...Option) (nodes Nodes) {
	nodes = make(Nodes, num)
	for i := range nodes {
		options = append(options, SetIndex(i))
		nodes[i] = New(t, tomlConfig, options...)
	}
	return nodes
}

// Init initialises all nodes and returns an error if any
// init operation failed.
func (nodes Nodes) Init() (err error) {
	initErrors := make(chan error)
	for _, node := range nodes {
		go func(node Node) {
			err := node.Init() // takes 2 seconds
			if err != nil {
				err = fmt.Errorf("node %s failed to initialise: %w", node, err)
			}
			initErrors <- err
		}(node)
	}

	for range nodes {
		initErr := <-initErrors
		if err == nil && initErr != nil {
			err = initErr
		}
	}

	return err
}

// Start starts all the nodes and returns the number of started nodes
// and an eventual start error. The started number should be used by
// the caller to wait for `started` errors coming from the wait error
// channel. All the nodes are stopped when the context is canceled,
// and `started` errors will be sent in the waitErr channel.
func (nodes Nodes) Start(ctx context.Context) (
	runtimeErrors []<-chan error, startErr error) {
	runtimeErrors = make([]<-chan error, 0, len(nodes))
	for _, node := range nodes {
		runtimeError, err := node.Start(ctx)
		if err != nil {
			return runtimeErrors, fmt.Errorf("node with index %d: %w",
				*node.index, err)
		}

		runtimeErrors = append(runtimeErrors, runtimeError)
	}

	for _, node := range nodes {
		port := node.RPCPort()
		err := waitForNode(ctx, port)
		if err != nil {
			return runtimeErrors, fmt.Errorf("node with index %d: %w", *node.index, err)
		}
	}

	return runtimeErrors, nil
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

	err := nodes.Init()
	if err != nil {
		t.Fatal(err)
	}

	nodesCtx, nodesCancel := context.WithCancel(ctx)
	runtimeErrors := newErrorsFanIn()

	for _, node := range nodes {
		runtimeError, err := node.Start(nodesCtx) // takes little time
		if err == nil {
			runtimeErrors.Add(node.String(), runtimeError)
			continue
		}

		t.Errorf("Node %s failed to start: %s", node, err)

		stopNodes(t, nodesCancel, runtimeErrors)
		t.FailNow()
	}

	// watch for runtime fatal error from any of the nodes
	watchDogCtx, watchDogCancel := context.WithCancel(ctx)
	watchDogDone := make(chan struct{})
	go func() {
		defer close(watchDogDone)
		err := runtimeErrors.watch(watchDogCtx)
		watchDogWasStopped := errors.Is(err, context.Canceled) ||
			errors.Is(err, context.DeadlineExceeded)
		if watchDogWasStopped {
			return
		}

		t.Errorf("one node has crashed: %s", err)
		// we cannot stop the test with t.FailNow() from a goroutine
		// other than the test goroutine, so we call failNow to signal
		// it to the test goroutine.
		signalTestToStop()
	}()

	t.Cleanup(func() {
		t.Helper()
		// Disengage node watchdog goroutine
		watchDogCancel()
		<-watchDogDone
		// Stop and wait for nodes to exit
		stopNodes(t, nodesCancel, runtimeErrors)
	})

	// this is run sequentially since all nodes start almost at the same time
	// so waiting for one node will also wait for all the others.
	// You can see this since the test logs out that all the nodes are ready
	// at the same time.
	for _, node := range nodes {
		err := waitForNode(ctx, node.RPCPort())
		if err == nil {
			t.Logf("Node %s is ready", node)
			continue
		}

		t.Errorf("Node %s failed to be ready: %s", node, err)
		stopNodes(t, nodesCancel, runtimeErrors)
		t.FailNow()
	}
}

func stopNodes(t *testing.T, nodesCancel context.CancelFunc,
	runtimeErrors *errorsFanIn) {
	t.Helper()

	// Stop the nodes and wait for them to exit
	nodesCancel()
	t.Logf("waiting on %d nodes to terminate...", runtimeErrors.len())
	const waitTimeout = 10 * time.Second
	err := runtimeErrors.waitForAll(waitTimeout)
	if err != nil {
		t.Logf("WARNING: %s", err)
	}
}
