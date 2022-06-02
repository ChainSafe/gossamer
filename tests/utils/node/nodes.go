// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/config/toml"
	"github.com/ChainSafe/gossamer/tests/utils/config"
	"github.com/ChainSafe/gossamer/tests/utils/runtime"
)

// Nodes is a slice of nodes.
type Nodes []Node

// MakeNodes creates `num` nodes using the `baseNode`
// as a base for each node. It sets the following fields:
// - the first node is always the BABE lead
// - the index of each node is incremented per node
// - the base path is set to a test temporary directory
// - remaining unset fields are set to their default.
func MakeNodes(t *testing.T, num int, tomlConfig toml.Config,
	options ...Option) (nodes Nodes) {
	nodes = make(Nodes, num)
	for i := range nodes {
		nodes[i].tomlConfig = tomlConfig
		// Set fields using options given
		for _, option := range options {
			option(&nodes[i])
		}

		// Set defaults using index `i`
		nodes[i].tomlConfig.Core.BABELead = i == 0

		if nodes[i].index == nil {
			nodes[i].index = intPtr(i)
		}

		// Set node defaults on the remaining unset fields
		nodes[i].setDefaults(t)
		nodes[i].setWriterPrefix()

		nodes[i].configPath = config.Write(t, nodes[i].tomlConfig)
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

	nodesCtx, nodesCancel := context.WithCancel(ctx)
	runtimeErrors := runtime.NewErrorsFanIn()

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

	// watch for runtime fatal error from any of the nodes
	watchDogCtx, watchDogCancel := context.WithCancel(ctx)
	watchDogDone := make(chan struct{})
	go func() {
		defer close(watchDogDone)
		err := runtimeErrors.Watch(watchDogCtx)
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
}

func stopNodes(t *testing.T, nodesCancel context.CancelFunc,
	runtimeErrors *runtime.ErrorsFanIn) {
	t.Helper()

	// Stop the nodes and wait for them to exit
	nodesCancel()
	t.Logf("waiting on %d nodes to terminate...", runtimeErrors.Len())
	const waitTimeout = 10 * time.Second
	err := runtimeErrors.WaitForAll(waitTimeout)
	if err != nil {
		t.Logf("WARNING: %s", err)
	}
}
