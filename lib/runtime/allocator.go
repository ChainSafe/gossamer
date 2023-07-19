// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package runtime

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math/bits"
)

// This module implements a freeing-bump allocator
// see more details at https://github.com/paritytech/substrate/issues/1615

// DefaultHeapBase is the default heap base value (offset) used when the runtime does not provide one
const DefaultHeapBase = uint32(1469576)

// The pointers need to be aligned to 8 bytes
const alignment uint32 = 8

// HeadsQty 23
const HeadsQty = 23

// MaxPossibleAllocation 2^25 bytes, 32 MiB
const MaxPossibleAllocation = (1 << 25)

// FreeingBumpHeapAllocator struct
type FreeingBumpHeapAllocator struct {
	bumper      uint32
	heads       [HeadsQty]uint32
	heap        Memory
	maxHeapSize uint32
	ptrOffset   uint32
	totalSize   uint32
}

// NewAllocator Creates a new allocation heap which follows a freeing-bump strategy.
// The maximum size which can be allocated at once is 16 MiB.
//
// # Arguments
//
//   - `mem` - A runtime.Memory to the available memory which is
//     used as the heap.
//
//   - `ptrOffset` - The pointers returned by `Allocate()` start from this
//     offset on. The pointer offset needs to be aligned to a multiple of 8,
//     hence a padding might be added to align `ptrOffset` properly.
//
//   - returns a pointer to an initilized FreeingBumpHeapAllocator
func NewAllocator(mem Memory, ptrOffset uint32) *FreeingBumpHeapAllocator {
	fbha := new(FreeingBumpHeapAllocator)

	padding := ptrOffset % alignment
	if padding != 0 {
		ptrOffset += alignment - padding
	}

	if mem.Size() <= ptrOffset {
		_, ok := mem.Grow(((ptrOffset - mem.Size()) / PageSize) + 1)
		if !ok {
			panic("exceeds max memory definition")
		}
	}

	fbha.bumper = 0
	fbha.heap = mem
	fbha.maxHeapSize = mem.Size() - alignment
	fbha.ptrOffset = ptrOffset
	fbha.totalSize = 0

	return fbha
}

func (fbha *FreeingBumpHeapAllocator) growHeap(numPages uint32) error {
	_, ok := fbha.heap.Grow(numPages)
	if !ok {
		return fmt.Errorf("heap.Grow ignored")
	}

	fbha.maxHeapSize = fbha.heap.Size() - alignment
	return nil
}

// Allocate determines if there is space available in WASM heap to grow the heap by 'size'.  If there is space
// available it grows the heap to fit give 'size'.  The heap grows is chunks of Powers of 2, so the growth becomes
// the next highest power of 2 of the requested size.
func (fbha *FreeingBumpHeapAllocator) Allocate(size uint32) (uint32, error) {
	// test for space allocation
	if size > MaxPossibleAllocation {
		err := errors.New("size too large")
		return 0, err
	}
	itemSize := nextPowerOf2GT8(size)

	if (itemSize + fbha.totalSize + fbha.ptrOffset) > fbha.maxHeapSize {
		pagesNeeded := ((itemSize + fbha.totalSize + fbha.ptrOffset) - fbha.maxHeapSize) / PageSize
		err := fbha.growHeap(pagesNeeded + 1)
		if err != nil {
			return 0, fmt.Errorf("allocator out of space; failed to grow heap; %w", err)
		}
	}

	// get pointer based on list_index
	listIndex := bits.TrailingZeros32(itemSize) - 3

	var ptr uint32
	if item := fbha.heads[listIndex]; item != 0 {
		// Something from the free list
		fourBytes := fbha.getHeap4bytes(item)
		fbha.heads[listIndex] = binary.LittleEndian.Uint32(fourBytes)
		ptr = item + 8
	} else {
		// Nothing te be freed. Bump.
		ptr = fbha.bump(itemSize+8) + 8
	}

	if (ptr + itemSize + fbha.ptrOffset) > fbha.maxHeapSize {
		pagesNeeded := (ptr + itemSize + fbha.ptrOffset - fbha.maxHeapSize) / PageSize
		err := fbha.growHeap(pagesNeeded + 1)
		if err != nil {
			return 0, fmt.Errorf("allocator out of space; failed to grow heap; %w", err)
		}

		if fbha.maxHeapSize < (ptr + itemSize + fbha.ptrOffset) {
			panic(fmt.Sprintf("failed to grow heap, want %d have %d", (ptr + itemSize + fbha.ptrOffset), fbha.maxHeapSize))
		}
	}

	// write "header" for allocated memory to heap
	for i := uint32(1); i <= 8; i++ {
		fbha.setHeap(ptr-i, 255)
	}
	fbha.setHeap(ptr-8, uint8(listIndex))
	fbha.totalSize = fbha.totalSize + itemSize + 8
	return fbha.ptrOffset + ptr, nil
}

// Deallocate deallocates the memory located at pointer address
func (fbha *FreeingBumpHeapAllocator) Deallocate(pointer uint32) error {
	ptr := pointer - fbha.ptrOffset
	if ptr < 8 {
		return errors.New("invalid pointer for deallocation")
	}
	listIndex := fbha.getHeapByte(ptr - 8)

	// update heads array, and heap "header"
	tail := fbha.heads[listIndex]
	fbha.heads[listIndex] = ptr - 8

	bTail := make([]byte, 4)
	binary.LittleEndian.PutUint32(bTail, tail)
	fbha.setHeap4bytes(ptr-8, bTail)

	// update heap total size
	itemSize := getItemSizeFromIndex(uint(listIndex))
	fbha.totalSize = fbha.totalSize - uint32(itemSize+8)

	return nil
}

// Clear resets the allocator, effectively freeing all allocated memory
func (fbha *FreeingBumpHeapAllocator) Clear() {
	fbha.bumper = 0
	fbha.totalSize = 0

	for i := range fbha.heads {
		fbha.heads[i] = 0
	}
}

func (fbha *FreeingBumpHeapAllocator) bump(qty uint32) uint32 {
	res := fbha.bumper
	fbha.bumper += qty
	return res
}

func (fbha *FreeingBumpHeapAllocator) setHeap(ptr uint32, value uint8) {
	if !fbha.heap.WriteByte(fbha.ptrOffset+ptr, value) {
		panic("write: out of range")
	}
}

func (fbha *FreeingBumpHeapAllocator) setHeap4bytes(ptr uint32, value []byte) {
	if !fbha.heap.Write(fbha.ptrOffset+ptr, value) {
		panic("write: out of range")
	}
}

func (fbha *FreeingBumpHeapAllocator) getHeap4bytes(ptr uint32) []byte {
	bytes, ok := fbha.heap.Read(fbha.ptrOffset+ptr, 4)
	if !ok {
		panic("read: out of range")
	}
	return bytes
}

func (fbha *FreeingBumpHeapAllocator) getHeapByte(ptr uint32) byte {
	b, ok := fbha.heap.ReadByte(fbha.ptrOffset + ptr)
	if !ok {
		panic("read: out of range")
	}
	return b
}

func getItemSizeFromIndex(index uint) uint {
	// we shift 1 by three places since the first possible item size is 8
	return 1 << 3 << index
}

func nextPowerOf2GT8(v uint32) uint32 {
	if v < 8 {
		return 8
	}
	v--
	v |= v >> 1
	v |= v >> 2
	v |= v >> 4
	v |= v >> 8
	v |= v >> 16
	v++
	return v
}
