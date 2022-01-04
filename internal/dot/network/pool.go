// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package network

// sizedBufferPool is a pool of buffers used for reading from streams
type sizedBufferPool struct {
	c chan []byte
}

func newSizedBufferPool(preAllocate, size int) (bp *sizedBufferPool) {
	bufferCh := make(chan []byte, size)

	for i := 0; i < preAllocate; i++ {
		buf := make([]byte, maxMessageSize)
		bufferCh <- buf
	}

	return &sizedBufferPool{
		c: bufferCh,
	}
}

// get gets a buffer from the sizedBufferPool, or creates a new one if none are
// available in the pool. Buffers have a pre-allocated capacity.
func (bp *sizedBufferPool) get() (b []byte) {
	select {
	case b = <-bp.c:
		// reuse existing buffer
		return b
	default:
		// create new buffer
		return make([]byte, maxMessageSize)
	}
}

// put returns the given buffer to the sizedBufferPool.
func (bp *sizedBufferPool) put(b []byte) {
	select {
	case bp.c <- b:
	default: // Discard the buffer if the pool is full.
	}
}
