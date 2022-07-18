// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package wasmer

// #include <stdlib.h>
import "C"

import (
	"fmt"
	"math/big"

	"github.com/ChainSafe/gossamer/lib/common/types"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/wasmerio/go-ext-wasm/wasmer"
)

// Convert 64bit wasm span descriptor to Go memory slice
func asMemorySlice(context wasmer.InstanceContext, pointerSize C.int64_t) (data []byte) {
	memory := context.Memory().Data()
	ptr, size := runtime.Int64ToPointerAndSize(int64(pointerSize))
	return memory[ptr : ptr+size]
}

// Copy a byte slice to wasm memory and return the resulting 64bit span descriptor
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
	pointerSize = runtime.PointerAndSizeToInt64(int32(ptr), int32(size))
	return pointerSize, nil
}

// Copy a byte slice of a fixed size to wasm memory and return resulting pointer
func toWasmMemorySized(context wasmer.InstanceContext, data []byte, size uint32) (
	pointer uint32, err error) {
	if int(size) != len(data) {
		// Programming error
		panic(fmt.Sprintf("data is %d bytes but size specified is %d", len(data), size))
	}

	allocator := context.Data().(*runtime.Context).Allocator

	ptr, err := allocator.Allocate(size)
	if err != nil {
		return 0, fmt.Errorf("allocating: %w", err)
	}

	memory := context.Memory().Data()
	copy(memory[ptr:ptr+size], data)

	return ptr, nil
}

// Wraps slice in optional.Bytes and copies result to wasm memory. Returns resulting 64bit span descriptor
func toWasmMemoryOptional(context wasmer.InstanceContext, data []byte) (
	pointerSize int64, err error) {
	var optionalBytes *[]byte
	if data != nil {
		optionalBytes = &data
	}

	enc, err := scale.Marshal(optionalBytes)
	if err != nil {
		return 0, fmt.Errorf("scale encoding: %w", err)
	}

	return toWasmMemory(context, enc)
}

// Wraps slice in Result type and copies result to wasm memory. Returns resulting 64bit span descriptor
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

// Wraps slice in optional and copies result to wasm memory. Returns resulting 64bit span descriptor
func toWasmMemoryOptionalUint32(context wasmer.InstanceContext, data *uint32) (
	pointerSize int64, err error) {
	enc, err := scale.Marshal(data)
	if err != nil {
		return 0, fmt.Errorf("scale encoding: %w", err)
	}
	return toWasmMemory(context, enc)
}

// toKillStorageResult returns enum encoded value
func toKillStorageResultEnum(allRemoved bool, numRemoved uint32) (
	encodedEnumValue []byte, err error) {
	encodedNumRemoved, err := scale.Marshal(numRemoved)
	if err != nil {
		return nil, fmt.Errorf("scale encoding: %w", err)
	}

	if allRemoved {
		// No key remains in the child trie.
		encodedEnumValue = append(encodedEnumValue, byte(0))
	} else {
		// At least one key still resides in the child trie due to the supplied limit.
		encodedEnumValue = append(encodedEnumValue, byte(1))
	}

	encodedEnumValue = append(encodedEnumValue, encodedNumRemoved...)

	return encodedEnumValue, nil
}

// Wraps slice in optional.FixedSizeBytes and copies result to wasm memory. Returns resulting 64bit span descriptor
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

func storageAppend(storage runtime.Storage, key, valueToAppend []byte) error {
	nextLength := big.NewInt(1)
	var valueResult []byte

	// this function assumes the item in storage is a SCALE encoded array of items
	// the valueToAppend is a new item, so it appends the item and increases the length prefix by 1
	currentValue := storage.Get(key)

	if len(currentValue) == 0 {
		valueResult = valueToAppend
	} else {
		var currentLength *big.Int
		err := scale.Unmarshal(currentValue, &currentLength)
		if err != nil {
			logger.Tracef(
				"item in storage is not SCALE encoded, overwriting at key 0x%x", key)
			storage.Set(key, append([]byte{4}, valueToAppend...))
			return nil //nolint:nilerr
		}

		lengthBytes, err := scale.Marshal(currentLength)
		if err != nil {
			return fmt.Errorf("scale encoding: %w", err)
		}
		// append new item, pop off number of bytes required for length encoding,
		// since we're not using old scale.Decoder
		valueResult = append(currentValue[len(lengthBytes):], valueToAppend...)

		// increase length by 1
		nextLength = big.NewInt(0).Add(currentLength, big.NewInt(1))
	}

	encodedLength, err := scale.Marshal(nextLength)
	if err != nil {
		logger.Tracef("failed to encode new length: %s", err)
		return fmt.Errorf("scale encoding: %w", err)
	}

	// append new length prefix to start of items array
	encodedValue := append(encodedLength, valueResult...)
	logger.Debugf("resulting value: 0x%x", encodedValue)
	storage.Set(key, encodedValue)
	return nil
}
