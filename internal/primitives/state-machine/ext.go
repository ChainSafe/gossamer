// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package statemachine

import (
	"github.com/ChainSafe/gossamer/pkg/scale"
)

type StorageAppend struct {
	data *StorageValue
}

func NewStorageAppend(data *StorageValue) *StorageAppend {
	return &StorageAppend{data: data}
}

func (sa *StorageAppend) ExtractLength() *uint32 {
	var length uint
	err := scale.Unmarshal(*sa.data, &length)
	if err != nil {
		return nil
	}
	decodedLength := uint32(length)
	return &decodedLength
}

func (sa *StorageAppend) ReplaceLength(oldLength *uint32, newLength uint32) {
	oldLenEncodedLen := 0
	if oldLength != nil {
		oldLenEncodedLen = compactLen(*oldLength)
	}
	newLenEncoded, _ := scale.Marshal(uint(newLength))
	data := spliceSlice(*sa.data, 0, oldLenEncodedLen, newLenEncoded)
	newStorageValue := StorageValue(data)
	*sa.data = newStorageValue
}

func (sa *StorageAppend) AppendRaw(value []byte) {
	*sa.data = append(*sa.data, value...)
}

func spliceSlice[T any](slice []T, startIdx, endIdx int, replacement []T) []T {
	if startIdx < 0 || endIdx > len(slice) || startIdx > endIdx {
		panic("invalid range")
	}

	result := make([]T, 0, len(slice)-endIdx+startIdx+len(replacement))
	result = append(result, slice[:startIdx]...)
	result = append(result, replacement...)
	result = append(result, slice[endIdx:]...)

	return result
}
