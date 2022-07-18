// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package wasmer

// #include <stdlib.h>
import "C"

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/ChainSafe/gossamer/lib/common/types"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/wasmerio/go-ext-wasm/wasmer"
)

// Convert 64bit wasm span descriptor to Go memory slice
func asMemorySlice(context wasmer.InstanceContext, span C.int64_t) []byte {
	memory := context.Memory().Data()
	ptr, size := runtime.Int64ToPointerAndSize(int64(span))
	return memory[ptr : ptr+size]
}

// Copy a byte slice to wasm memory and return the resulting 64bit span descriptor
func toWasmMemory(context wasmer.InstanceContext, data []byte) (int64, error) {
	allocator := context.Data().(*runtime.Context).Allocator
	size := uint32(len(data))

	out, err := allocator.Allocate(size)
	if err != nil {
		return 0, err
	}

	memory := context.Memory().Data()

	if uint32(len(memory)) < out+size {
		panic(fmt.Sprintf("length of memory is less than expected, want %d have %d", out+size, len(memory)))
	}

	copy(memory[out:out+size], data)
	return runtime.PointerAndSizeToInt64(int32(out), int32(size)), nil
}

// Copy a byte slice of a fixed size to wasm memory and return resulting pointer
func toWasmMemorySized(context wasmer.InstanceContext, data []byte, size uint32) (uint32, error) {
	if int(size) != len(data) {
		return 0, errors.New("internal byte array size missmatch")
	}

	allocator := context.Data().(*runtime.Context).Allocator

	out, err := allocator.Allocate(size)
	if err != nil {
		return 0, err
	}

	memory := context.Memory().Data()
	copy(memory[out:out+size], data)

	return out, nil
}

// Wraps slice in optional.Bytes and copies result to wasm memory. Returns resulting 64bit span descriptor
func toWasmMemoryOptional(context wasmer.InstanceContext, data []byte) (int64, error) {
	var opt *[]byte
	if data != nil {
		temp := data
		opt = &temp
	}

	enc, err := scale.Marshal(opt)
	if err != nil {
		return 0, err
	}

	return toWasmMemory(context, enc)
}

// Wraps slice in Result type and copies result to wasm memory. Returns resulting 64bit span descriptor
func toWasmMemoryResult(context wasmer.InstanceContext, data []byte) (int64, error) {
	var res *types.Result
	if len(data) == 0 {
		res = types.NewResult(byte(1), nil)
	} else {
		res = types.NewResult(byte(0), data)
	}

	enc, err := res.Encode()
	if err != nil {
		return 0, err
	}

	return toWasmMemory(context, enc)
}

// Wraps slice in optional and copies result to wasm memory. Returns resulting 64bit span descriptor
func toWasmMemoryOptionalUint32(context wasmer.InstanceContext, data *uint32) (int64, error) {
	var opt *uint32
	if data != nil {
		temp := *data
		opt = &temp
	}

	enc, err := scale.Marshal(opt)
	if err != nil {
		return int64(0), err
	}
	return toWasmMemory(context, enc)
}

// toKillStorageResult returns enum encoded value
func toKillStorageResultEnum(allRemoved bool, numRemoved uint32) ([]byte, error) {
	var b, sbytes []byte
	sbytes, err := scale.Marshal(numRemoved)
	if err != nil {
		return nil, err
	}

	if allRemoved {
		// No key remains in the child trie.
		b = append(b, byte(0))
	} else {
		// At least one key still resides in the child trie due to the supplied limit.
		b = append(b, byte(1))
	}

	b = append(b, sbytes...)

	return b, err
}

// Wraps slice in optional.FixedSizeBytes and copies result to wasm memory. Returns resulting 64bit span descriptor
func toWasmMemoryFixedSizeOptional(context wasmer.InstanceContext, data []byte) (int64, error) {
	var opt [64]byte
	copy(opt[:], data)
	enc, err := scale.Marshal(&opt)
	if err != nil {
		return 0, err
	}
	return toWasmMemory(context, enc)
}

func storageAppend(storage runtime.Storage, key, valueToAppend []byte) error {
	nextLength := big.NewInt(1)
	var valueRes []byte

	// this function assumes the item in storage is a SCALE encoded array of items
	// the valueToAppend is a new item, so it appends the item and increases the length prefix by 1
	valueCurr := storage.Get(key)

	if len(valueCurr) == 0 {
		valueRes = valueToAppend
	} else {
		var currLength *big.Int
		err := scale.Unmarshal(valueCurr, &currLength)
		if err != nil {
			logger.Tracef(
				"item in storage is not SCALE encoded, overwriting at key 0x%x", key)
			storage.Set(key, append([]byte{4}, valueToAppend...))
			return nil //nolint:nilerr
		}

		lengthBytes, err := scale.Marshal(currLength)
		if err != nil {
			return err
		}
		// append new item, pop off number of bytes required for length encoding,
		// since we're not using old scale.Decoder
		valueRes = append(valueCurr[len(lengthBytes):], valueToAppend...)

		// increase length by 1
		nextLength = big.NewInt(0).Add(currLength, big.NewInt(1))
	}

	lengthEnc, err := scale.Marshal(nextLength)
	if err != nil {
		logger.Tracef("failed to encode new length: %s", err)
		return err
	}

	// append new length prefix to start of items array
	lengthEnc = append(lengthEnc, valueRes...)
	logger.Debugf("resulting value: 0x%x", lengthEnc)
	storage.Set(key, lengthEnc)
	return nil
}
