package runtime

import (
	wasm "github.com/wasmerio/go-ext-wasm/wasmer"
)

// This module implements a freeing-bump allocator
// see more details at https://github.com/paritytech/substrate/issues/1615

// The pointers need to be aligned to 8 bytes
const ALIGNMENT int32 = 8
const N = 22

type FreeingBumpHeapAllocator struct {
	bumper         int32
	heads          [N]int32
	heap           *wasm.Memory
	max_heap_size  uint32
	ptr_offset     int32
	total_size     uint32
}

func newAllocator(mem *wasm.Memory) FreeingBumpHeapAllocator {
	fbha := new(FreeingBumpHeapAllocator)
	current_size := mem.Length()
	heap_size := current_size
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
func (fbha FreeingBumpHeapAllocator) allocate(size int32) int32 {
	// TODO: ed, implement this
	return 1
}