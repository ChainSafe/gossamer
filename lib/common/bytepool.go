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

package common

import "fmt"

type BytePool struct {
	c chan byte
}

func NewBytePool256() (bp *BytePool, err error) {
	bp = NewBytePool(256)
	for i := 0; i < 256; i++ {
		err = bp.Put(byte(i))
	}
	return
}
func NewBytePool(size int) (bp *BytePool) {
	return &BytePool{
		c: make(chan byte, size),
	}
}

// Get gets a Buffer from the BytePool, or creates a new one if none are
// available in the pool.
func (bp *BytePool) Get() (b byte, err error) {
	select {
	case b = <-bp.c:
	default:
		err = fmt.Errorf("all slots used")
	}
	return
}

// Put returns the given Buffer to the BytePool.
func (bp *BytePool) Put(b byte) error {
	select {
	case bp.c <- b:
		return nil
	default: // Discard the buffer if the pool is full.
		return fmt.Errorf("pool is full")
	}
}

// NumPooled returns the number of items currently pooled.
func (bp *BytePool) NumPooled() int {
	return len(bp.c)
}
