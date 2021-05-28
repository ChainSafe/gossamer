// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

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
