// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package overlayedchanges

import (
	"github.com/ChainSafe/gossamer/internal/primitives/state-machine/backend"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// storageAppend is a helper struct that appends a StorageValue to a slice.
type storageAppend struct {
	data *backend.StorageValue
}

// newStorageAppend creates a new storageAppend instance.
func newStorageAppend(data *backend.StorageValue) *storageAppend {
	return &storageAppend{data: data}
}

// extractLength extracts the scale encoded length of the StorageValue.
func (sa *storageAppend) extractLength() *uint {
	var length uint
	err := scale.Unmarshal(*sa.data, &length)
	if err != nil {
		return nil
	}
	return &length
}

// replaceLength replaces the length of the StorageValue using scale encoding.
func (sa *storageAppend) replaceLength(oldLength *uint, newLength uint) {
	oldLenEncodedLen := 0
	if oldLength != nil {
		oldLenEncodedLen = compactLen(*oldLength)
	}
	newLenEncoded, _ := scale.Marshal(newLength)
	data := spliceSlice(*sa.data, 0, oldLenEncodedLen, newLenEncoded)
	newStorageValue := backend.StorageValue(data)
	*sa.data = newStorageValue
}

// appendRaw appends a raw byte slice to the current StorageValue.
func (sa *storageAppend) appendRaw(value []byte) {
	*sa.data = append(*sa.data, value...)
}

// spliceSlice is a helper function that replaces a slice of elements with a new slice in the given positions.
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

func compactLen(val uint) int {
	switch {
	case val <= 0b0011_1111:
		return 1
	case val <= 0b0011_1111_1111_1111:
		return 2
	case val <= 0b0011_1111_1111_1111_1111_1111_1111_1111:
		return 4
	default:
		return 5
	}
}
