// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"sync"
)

////// GENERIC SHARED DATA //////

type SharedDataGeneric[T any] struct {
	sync.Mutex
	inner T
}

func NewSharedDataGeneric[T any](msg T) *SharedDataGeneric[T] {
	return &SharedDataGeneric[T]{
		inner: msg,
	}
}

// Read Thread safe read of inner data
func (sd *SharedDataGeneric[T]) Read() T {
	sd.Lock()
	defer sd.Unlock()
	inner := sd.inner
	return inner
}

// Write Thread safe write of inner data
func (sd *SharedDataGeneric[T]) Write(data T) {
	sd.Lock()
	defer sd.Unlock()
	sd.inner = data
}

// Acquire lock on shared data
// Note: This MUST be released in order to allow other threads access to the data
func (sd *SharedDataGeneric[T]) Acquire() {
	sd.Lock()
}

// Release lock on shared data
// Note: This MUST be preceded by an acquire of the lock, else will result in runtime error
func (sd *SharedDataGeneric[T]) Release() {
	sd.Unlock()
}
