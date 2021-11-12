// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package network

// sizedBufferPool is a pool of buffers used for reading from streams
type sizedBufferPool struct {
	c chan *[maxMessageSize]byte
}

func newSizedBufferPool(min, max int) (bp *sizedBufferPool) {
	bufferCh := make(chan *[maxMessageSize]byte, max)

	for i := 0; i < min; i++ {
		buf := [maxMessageSize]byte{}
		bufferCh <- &buf
	}

	return &sizedBufferPool{
		c: bufferCh,
	}
}

// get gets a buffer from the sizedBufferPool, or creates a new one if none are
// available in the pool. Buffers have a pre-allocated capacity.
func (bp *sizedBufferPool) get() [maxMessageSize]byte {
	var buff *[maxMessageSize]byte
	select {
	case buff = <-bp.c:
	// reuse existing buffer
	default:
		// create new buffer
		buff = &[maxMessageSize]byte{}
	}
	return *buff
}

// put returns the given buffer to the sizedBufferPool.
func (bp *sizedBufferPool) put(b *[maxMessageSize]byte) {
	select {
	case bp.c <- b:
	default: // Discard the buffer if the pool is full.
	}
}
