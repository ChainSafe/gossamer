package runtime

import (
	"errors"
	wasm "github.com/wasmerio/go-ext-wasm/wasmer"
)

// This module implements a freeing-bump allocator
// see more details at https://github.com/paritytech/substrate/issues/1615

// The pointers need to be aligned to 8 bytes
const ALIGNMENT int32 = 8
const N = 22
const MAX_POSSIBLE_ALLOCATION = 16777216    // 2^24 bytes

type FreeingBumpHeapAllocator struct {
	bumper         int32
	heads          [N]int32
	heap           *wasm.Memory
	max_heap_size  int32
	ptr_offset     int32
	total_size     int32
}

func newAllocator(mem *wasm.Memory) FreeingBumpHeapAllocator {
	fbha := new(FreeingBumpHeapAllocator)
	current_size := mem.Length()
	heap_size := int32(current_size)
	used_size := 0  // TODO actually calculate this

	ptr_offset := int32(used_size)
	padding := ptr_offset % ALIGNMENT;
	if padding != 0 {
		ptr_offset += ALIGNMENT - padding;
	}

	fbha.bumper = 0
	fbha.heap = mem
	fbha.max_heap_size = heap_size
	fbha.ptr_offset = ptr_offset
	fbha.total_size = 0

	return *fbha
}
func (fbha FreeingBumpHeapAllocator) allocate(size int32) (int32, error) {
	// TODO: ed, implement this
	if size > MAX_POSSIBLE_ALLOCATION {
		err := errors.New("Error size to large")
		return 0,err
	}
	item_size := nextPowerOf2GT8(size);
	if (item_size + 8 + fbha.total_size) > fbha.max_heap_size {
		err := errors.New("Allocator Out of space")
		return 0, err
	}
	return 1, nil
}

func nextPowerOf2GT8(v int32) int32 {
	if v < 8 {
		return 8
	}
	v--
	v |= v >> 1
	v |= v >> 2
	v |= v >> 4
	v |= v >> 8
	v |= v >> 16
	v |= v >> 32
	v++
	return v

}