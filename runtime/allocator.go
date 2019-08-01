package runtime

import (
	wasm "github.com/wasmerio/go-ext-wasm/wasmer"
)

// This module implements a freeing-bump allocator
// see more details at https://github.com/paritytech/substrate/issues/1615

// The pointers need to be aligned to 8 bytes
const ALIGNMENT uint32 = 8
const N = 22

type FreeingBumpHeapAllocator struct {
	bumper         int32
	heads          [N]int32
	heap           *wasm.Memory
	max_heap_size  uint32
	ptr_offset     uint32
	total_size     uint32
}

func (fbha FreeingBumpHeapAllocator) allocate(size int32) int32 {
	// TODO: ed, implement this
	return 1
}