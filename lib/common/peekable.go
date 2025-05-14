// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package common

import (
	"iter"
)

// Peekable2 wraps a Seq2[K, V] so you can peek the next item without consuming it
type Peekable2[K, V any] struct {
	next      func() (K, V, bool)
	stop      func()
	bufK      K
	bufV      V
	hasBuffer bool
}

// NewPeekable2 converts an iter.Seq2[K, V] into a *Peekable2[K, V].
func NewPeekable2[K, V any](seq iter.Seq2[K, V]) *Peekable2[K, V] {
	next, stop := iter.Pull2(seq)
	return &Peekable2[K, V]{next: next, stop: stop}
}

// Peek returns (k, v, true) of the next pair without consuming it
func (p *Peekable2[K, V]) Peek() (K, V, bool) {
	if p.hasBuffer {
		return p.bufK, p.bufV, true
	}
	k, v, ok := p.next()
	if !ok {
		var zk K
		var zv V
		return zk, zv, false
	}
	p.bufK, p.bufV, p.hasBuffer = k, v, true
	return k, v, true
}

// Next returns (k, v, true) consuming it
func (p *Peekable2[K, V]) Next() (K, V, bool) {
	if p.hasBuffer {
		p.hasBuffer = false
		return p.bufK, p.bufV, true
	}
	return p.next()
}

// Close must be called (usually using defer) if you exit before the iterator is done
func (p *Peekable2[K, V]) Close() {
	p.stop()
}
