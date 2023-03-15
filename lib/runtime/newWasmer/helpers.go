package newWasmer

import "C"
import (
	"fmt"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// toPointerSize converts an uint32 pointer and uint32 size
// to an int64 pointer size.
func toPointerSize(ptr, size uint32) (pointerSize int64) {
	return int64(ptr) | (int64(size) << 32)
}

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

// toWasmMemory copies a Go byte slice to wasm memory and returns the corresponding
// 64 bit pointer size.
func toWasmMemory(context *Context, data []byte) (
	pointerSize int64, err error) {
	allocator := context.Allocator
	size := uint32(len(data))

	ptr, err := allocator.Allocate(size)
	if err != nil {
		return 0, fmt.Errorf("allocating: %w", err)
	}

	memory := context.Memory.Data()

	if uint32(len(memory)) < ptr+size {
		panic(fmt.Sprintf("length of memory is less than expected, want %d have %d", ptr+size, len(memory)))
	}

	copy(memory[ptr:ptr+size], data)
	pointerSize = toPointerSize(ptr, size)
	return pointerSize, nil
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

// toWasmMemoryOptional scale encodes the byte slice `data`, writes it to wasm memory
// and returns the corresponding 64 bit pointer size.
func toWasmMemoryOptional(context *Context, data []byte) (
	pointerSize int64, err error) {
	var optionalSlice *[]byte
	if data != nil {
		optionalSlice = &data
	}

	encoded, err := scale.Marshal(optionalSlice)
	if err != nil {
		return 0, err
	}

	return toWasmMemory(context, encoded)
}
func toWasmMemoryOptionalNil(context *Context) (
	cPointerSize C.int64_t, err error) {
	pointerSize, err := toWasmMemoryOptional(context, nil)
	if err != nil {
		return 0, err
	}

	return C.int64_t(pointerSize), nil
}

func mustToWasmMemoryOptionalNil(context *Context) (
	cPointerSize C.int64_t) {
	cPointerSize, err := toWasmMemoryOptionalNil(context)
	if err != nil {
		panic(err)
	}
	return cPointerSize
}

// toWasmMemoryFixedSizeOptional copies the `data` byte slice to a 64B array,
// scale encodes the pointer to the resulting array, writes it to wasm memory
// and returns the corresponding 64 bit pointer size.
func toWasmMemoryFixedSizeOptional(context *Context, data []byte) (
	pointerSize int64, err error) {
	var optionalFixedSize [64]byte
	copy(optionalFixedSize[:], data)
	encodedOptionalFixedSize, err := scale.Marshal(&optionalFixedSize)
	if err != nil {
		return 0, fmt.Errorf("scale encoding: %w", err)
	}
	return toWasmMemory(context, encodedOptionalFixedSize)
}
