// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// errorsFanIn takes care of fanning runtime errors from
// different error channels to a single error channel.
// It also handles removal of specific runtime error channels
// from the fan in, which can be useful if one node crashes
// or is stopped on purpose.
type errorsFanIn struct {
	nodeToRuntimeError map[string]<-chan error
	nodeToFaninCancel  map[string]context.CancelFunc
	nodeToFaninDone    map[string]<-chan struct{}
	fifo               chan nodeError
	mutex              sync.RWMutex
}

type nodeError struct {
	node string
	err  error
}

// newErrorsFanIn returns a new errors fan in object.
func newErrorsFanIn() *errorsFanIn {
	return &errorsFanIn{
		nodeToRuntimeError: make(map[string]<-chan error),
		nodeToFaninCancel:  make(map[string]context.CancelFunc),
		nodeToFaninDone:    make(map[string]<-chan struct{}),
		fifo:               make(chan nodeError),
	}
}

// Add adds a runtime error receiving channel to the fan in mechanism
// for the particular node string given. Note each node string must be
// unique or the code will panic.
func (e *errorsFanIn) Add(node string, runtimeError <-chan error) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	// check for duplicate node string
	_, exists := e.nodeToRuntimeError[node]
	if exists {
		panic(fmt.Sprintf("node %q was already added", node))
	}

	e.nodeToRuntimeError[node] = runtimeError
	ctx, cancel := context.WithCancel(context.Background())
	e.nodeToFaninCancel[node] = cancel
	fanInDone := make(chan struct{})
	e.nodeToFaninDone[node] = fanInDone

	go fanIn(ctx, node, runtimeError, e.fifo, fanInDone)
}

func fanIn(ctx context.Context, node string,
	runtimeError <-chan error, fifo chan<- nodeError,
	fanInDone chan<- struct{}) {
	defer close(fanInDone)

	select {
	case <-ctx.Done():
		return
	case err := <-runtimeError:
		fifo <- nodeError{
			node: node,
			err:  err,
		}
	}
}

// len returns how many nodes are being monitored
// for runtime errors.
func (e *errorsFanIn) len() (length int) {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	return len(e.nodeToRuntimeError)
}

// remove removes a node from the fan in mechanism
// and clears it from the internal maps.
func (e *errorsFanIn) remove(node string) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	e.removeWithoutLock(node)
}

func (e *errorsFanIn) removeWithoutLock(node string) {
	// Stop fanning in
	cancelFanIn := e.nodeToFaninCancel[node]
	fanInDone := e.nodeToFaninDone[node]
	cancelFanIn()
	<-fanInDone

	// Clear from maps
	delete(e.nodeToRuntimeError, node)
	delete(e.nodeToFaninCancel, node)
	delete(e.nodeToFaninDone, node)
}

var (
	ErrWaitTimedOut = errors.New("waiting for all nodes timed out")
)

// waitForAll waits to collect all the runtime errors from all the
// nodes added and which did not crash previously.
// If the timeout duration specified is reached, all internal
// fan in operations are stopped and all the nodes are cleared from
// the internal maps, and an error is returned.
func (e *errorsFanIn) waitForAll(timeout time.Duration) (err error) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	timer := time.NewTimer(timeout)

	length := len(e.nodeToRuntimeError)
	for i := 0; i < length; i++ {
		select {
		case <-timer.C:
			for node := range e.nodeToRuntimeError {
				e.removeWithoutLock(node)
			}
			return fmt.Errorf("%w: for %d nodes after %s",
				ErrWaitTimedOut, len(e.nodeToRuntimeError), timeout)
		case identifiedError := <-e.fifo: // one error per node max
			node := identifiedError.node
			e.removeWithoutLock(node)
		}
	}

	_ = timer.Stop()

	return nil
}

// watch returns the next runtime error from the N runtime
// error channels, in a first in first out mechanism.
func (e *errorsFanIn) watch(ctx context.Context) (err error) {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case identifiedErr := <-e.fifo: // single fatal error
		e.remove(identifiedErr.node)
		return identifiedErr.err
	}
}
