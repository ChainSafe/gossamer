// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package wasmer

// #include <stdlib.h>
import "C" //skipcq: SCC-compile

import (
	"fmt"
	"math/big"

	"github.com/ChainSafe/gossamer/lib/common/types"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/wasmerio/go-ext-wasm/wasmer"
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
func asMemorySlice(context wasmer.InstanceContext, pointerSize C.int64_t) (data []byte) {
	memory := context.Memory().Data()
	ptr, size := splitPointerSize(int64(pointerSize))
	return memory[ptr : ptr+size]
}

// toWasmMemory copies a Go byte slice to wasm memory and returns the corresponding
// 64 bit pointer size.
func toWasmMemory(context wasmer.InstanceContext, data []byte) (
	pointerSize int64, err error) {
	allocator := context.Data().(*runtime.Context).Allocator
	size := uint32(len(data))

	ptr, err := allocator.Allocate(size)
	if err != nil {
		return 0, fmt.Errorf("allocating: %w", err)
	}

	memory := context.Memory().Data()

	if uint32(len(memory)) < ptr+size {
		panic(fmt.Sprintf("length of memory is less than expected, want %d have %d", ptr+size, len(memory)))
	}

	copy(memory[ptr:ptr+size], data)
	pointerSize = toPointerSize(ptr, size)
	return pointerSize, nil
}

// toWasmMemorySized copies a Go byte slice to wasm memory and returns the corresponding
// 32 bit pointer. Note the data must have a well known fixed length in the runtime.
func toWasmMemorySized(context wasmer.InstanceContext, data []byte) (
	pointer uint32, err error) {
	allocator := context.Data().(*runtime.Context).Allocator

	size := uint32(len(data))
	pointer, err = allocator.Allocate(size)
	if err != nil {
		return 0, fmt.Errorf("allocating: %w", err)
	}

	memory := context.Memory().Data()
	copy(memory[pointer:pointer+size], data)

	return pointer, nil
}

// toWasmMemoryOptional scale encodes the byte slice `data`, writes it to wasm memory
// and returns the corresponding 64 bit pointer size.
func toWasmMemoryOptional(context wasmer.InstanceContext, data []byte) (
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

// toWasmMemoryResult wraps the data byte slice in a Result type, scale encodes it,
// copies it to wasm memory and returns the corresponding 64 bit pointer size.
func toWasmMemoryResult(context wasmer.InstanceContext, data []byte) (
	pointerSize int64, err error) {
	var result *types.Result
	if len(data) == 0 {
		result = types.NewResult(byte(1), nil)
	} else {
		result = types.NewResult(byte(0), data)
	}

	encodedResult, err := result.Encode()
	if err != nil {
		return 0, fmt.Errorf("encoding result: %w", err)
	}

	return toWasmMemory(context, encodedResult)
}

// toWasmMemoryOptional scale encodes the uint32 pointer `data`, writes it to wasm memory
// and returns the corresponding 64 bit pointer size.
func toWasmMemoryOptionalUint32(context wasmer.InstanceContext, data *uint32) (
	pointerSize int64, err error) {
	enc, err := scale.Marshal(data)
	if err != nil {
		return 0, fmt.Errorf("scale encoding: %w", err)
	}
	return toWasmMemory(context, enc)
}

func mustToWasmMemoryNil(context wasmer.InstanceContext) (
	cPointerSize C.int64_t) {
	allocator := context.Data().(*runtime.Context).Allocator
	ptr, err := allocator.Allocate(0)
	if err != nil {
		// we allocate 0 byte, this should never fail
		panic(err)
	}
	pointerSize := toPointerSize(ptr, 0)
	return C.int64_t(pointerSize)
}

func toWasmMemoryOptionalNil(context wasmer.InstanceContext) (
	cPointerSize C.int64_t, err error) {
	pointerSize, err := toWasmMemoryOptional(context, nil)
	if err != nil {
		return 0, err
	}

	return C.int64_t(pointerSize), nil
}

func mustToWasmMemoryOptionalNil(context wasmer.InstanceContext) (
	cPointerSize C.int64_t) {
	cPointerSize, err := toWasmMemoryOptionalNil(context)
	if err != nil {
		panic(err)
	}
	return cPointerSize
}

func toWasmMemoryResultEmpty(context wasmer.InstanceContext) (
	cPointerSize C.int64_t, err error) {
	pointerSize, err := toWasmMemoryResult(context, nil)
	if err != nil {
		return 0, err
	}
	return C.int64_t(pointerSize), nil
}

func mustToWasmMemoryResultEmpty(context wasmer.InstanceContext) (
	cPointerSize C.int64_t) {
	cPointerSize, err := toWasmMemoryResultEmpty(context)
	if err != nil {
		panic(err)
	}
	return cPointerSize
}

// toKillStorageResultEnum encodes the `allRemoved` flag and
// the `numRemoved` uint32 to a byte slice and returns it.
// The format used is:
// Byte 0: 1 if allRemoved is false, 0 otherwise
// Byte 1-5: scale encoding of numRemoved (up to 4 bytes)
func toKillStorageResultEnum(allRemoved bool, numRemoved uint32) (
	encodedEnumValue []byte, err error) {
	encodedNumRemoved, err := scale.Marshal(numRemoved)
	if err != nil {
		return nil, fmt.Errorf("scale encoding: %w", err)
	}

	encodedEnumValue = make([]byte, len(encodedNumRemoved)+1)
	if !allRemoved {
		// At least one key resides in the child trie due to the supplied limit.
		encodedEnumValue[0] = 1
	}
	copy(encodedEnumValue[1:], encodedNumRemoved)

	return encodedEnumValue, nil
}

// toWasmMemoryFixedSizeOptional copies the `data` byte slice to a 64B array,
// scale encodes the pointer to the resulting array, writes it to wasm memory
// and returns the corresponding 64 bit pointer size.
func toWasmMemoryFixedSizeOptional(context wasmer.InstanceContext, data []byte) (
	pointerSize int64, err error) {
	var optionalFixedSize [64]byte
	copy(optionalFixedSize[:], data)
	encodedOptionalFixedSize, err := scale.Marshal(&optionalFixedSize)
	if err != nil {
		return 0, fmt.Errorf("scale encoding: %w", err)
	}
	return toWasmMemory(context, encodedOptionalFixedSize)
}

func storageAppend(storage runtime.Storage, key, valueToAppend []byte, version trie.Version) (err error) {
	// this function assumes the item in storage is a SCALE encoded array of items
	// the valueToAppend is a new item, so it appends the item and increases the length prefix by 1
	currentValue := storage.Get(key)

	var value []byte
	if len(currentValue) == 0 {
		nextLength := big.NewInt(1)
		encodedLength, err := scale.Marshal(nextLength)
		if err != nil {
			return fmt.Errorf("scale encoding: %w", err)
		}
		value = make([]byte, len(encodedLength)+len(valueToAppend))
		// append new length prefix to start of items array
		copy(value, encodedLength)
		copy(value[len(encodedLength):], valueToAppend)
	} else {
		var currentLength *big.Int
		err := scale.Unmarshal(currentValue, &currentLength)
		if err != nil {
			logger.Tracef(
				"item in storage is not SCALE encoded, overwriting at key 0x%x", key)
			value = make([]byte, 1+len(valueToAppend))
			value[0] = 4
			copy(value[1:], valueToAppend)
		} else {
			lengthBytes, err := scale.Marshal(currentLength)
			if err != nil {
				return fmt.Errorf("scale encoding: %w", err)
			}

			// increase length by 1
			nextLength := big.NewInt(0).Add(currentLength, big.NewInt(1))
			nextLengthBytes, err := scale.Marshal(nextLength)
			if err != nil {
				return fmt.Errorf("scale encoding next length bytes: %w", err)
			}

			// append new item, pop off number of bytes required for length encoding,
			// since we're not using old scale.Decoder
			value = make([]byte, len(nextLengthBytes)+len(currentValue)-len(lengthBytes)+len(valueToAppend))
			// append new length prefix to start of items array
			i := 0
			copy(value[i:], nextLengthBytes)
			i += len(nextLengthBytes)
			copy(value[i:], currentValue[len(lengthBytes):])
			i += len(currentValue) - len(lengthBytes)
			copy(value[i:], valueToAppend)
		}
	}

	err = storage.Put(key, value, version)
	if err != nil {
		return fmt.Errorf("putting key and value in storage: %w", err)
	}

	return nil
}

func panicOnError(err error) {
	if err != nil {
		panic(err)
	}
}
