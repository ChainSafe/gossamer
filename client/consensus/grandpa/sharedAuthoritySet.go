// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"sync"
)

// SharedData Generic shared data structure
type SharedData[T any] struct {
	sync.Mutex
	inner T
}

// NewSharedData Creates new SharedData
func NewSharedData[T any](msg T) *SharedData[T] {
	return &SharedData[T]{
		inner: msg,
	}
}

// Read Thread safe read of inner data
func (sd *SharedData[T]) Read() T {
	sd.Lock()
	defer sd.Unlock()
	inner := sd.inner
	return inner
}

// Write Thread safe write of inner data
func (sd *SharedData[T]) Write(data T) {
	sd.Lock()
	defer sd.Unlock()
	sd.inner = data
}

// Acquire lock on shared data
// Note: This MUST be released in order to allow other threads access to the data
func (sd *SharedData[T]) Acquire() {
	sd.Lock()
}

// Release lock on shared data
// Note: This MUST be preceded by an acquire of the lock, else will result in runtime error
func (sd *SharedData[T]) Release() {
	sd.Unlock()
}
