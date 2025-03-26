// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package statemachine

// Content in an overlay for a given transactional depth.
type storageEntry interface {
	value() StorageValue
}

type (
	// The storage entry should be set to the stored value.
	setStorageEntry struct {
		data StorageValue
	}

	// The storage entry should be removed.
	removeStorageEntry struct{}

	// The storage entry was appended to.
	// This assumes that the storage entry is encoded as a SCALE list. This means that it is
	// prefixed with a compact uint that reprensents the length, followed by all the encoded
	// elements.
	appendStorageEntry struct {
		// The value of the storage entry.
		// This may or may not be prefixed by the length, depending on the materialised length.
		data StorageValue
		// Current number of elements stored in data.
		currentLength uint
		// The number of elements as stored in the prefixed length in `data`.
		// If `nil`, than `data` is not yet prefixed with the length.
		materializedLength *uint
		// The size of `data` in the parent transactional layer.
		// Only set when the parent layer is in  `Append` state.
		parentSize *uint
	}
)

func (se setStorageEntry) value() StorageValue    { return se.data }
func (se removeStorageEntry) value() StorageValue { return nil }
func (se *appendStorageEntry) value() StorageValue {
	se.materializedInPlace()
	return se.data
}

// Materialise the internal state and cache the resulting materialised value.
func (se *appendStorageEntry) materializedInPlace() {
	currentLength := se.currentLength
	if se.materializedLength != nil && *se.materializedLength == currentLength {
		return
	}
	NewStorageAppend(&se.data).ReplaceLength(se.materializedLength, currentLength)
	se.materializedLength = &currentLength
}
