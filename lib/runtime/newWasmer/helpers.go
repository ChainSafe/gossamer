package newWasmer

import "C"
import (
	"fmt"
)

// splitPointerSize converts an int64 pointer size to an
// uint32 pointer and an uint32 size.
func splitPointerSize(pointerSize int64) (ptr, size uint32) {
	return uint32(pointerSize), uint32(pointerSize >> 32)
}

// asMemorySlice converts a 64 bit pointer size to a Go byte slice.
func asMemorySlice(context *Context, pointerSize int64) (data []byte) {
	memory := context.Memory.Data()
	ptr, size := splitPointerSize(pointerSize)
	return memory[ptr : ptr+size]
}

// toWasmMemorySized copies a Go byte slice to wasm memory and returns the corresponding
// 32 bit pointer. Note the data must have a well known fixed length in the runtime.
func toWasmMemorySized(context *Context, data []byte) (
	pointer uint32, err error) {
	allocator := context.Allocator

	size := uint32(len(data))
	pointer, err = allocator.Allocate(size)
	if err != nil {
		return 0, fmt.Errorf("allocating: %w", err)
	}

	memory := context.Memory.Data()
	copy(memory[pointer:pointer+size], data)

	return pointer, nil
}
