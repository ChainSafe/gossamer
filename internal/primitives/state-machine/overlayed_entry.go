// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package statemachine

import (
	"math"

	"github.com/tidwall/btree"
)

const proofOverlayNonEmpty = `
An OverlayValue is always created with at least one transaction and dropped as soon
as the last transaction is removed; qed`

// An overlay that contains all versions of a value for a specific key.
type OverlayedEntry[V any] struct {
	// The individual versions of that value.
	// One entry per transactions during that the value was actually written.
	transactions []transaction[V]
}

func NewOverlayedEntry[V any]() *OverlayedEntry[V] {
	return &OverlayedEntry[V]{
		transactions: []transaction[V]{},
	}
}

func (oe *OverlayedEntry[V]) ValueRef() *V {
	if len(oe.transactions) == 0 {
		panic(proofOverlayNonEmpty)
	}

	return &oe.transactions[len(oe.transactions)-1].value
}

// The value as seen by the current transaction.
func (oe *OverlayedEntry[V]) StorageValue() StorageValue {
	return any(*oe.ValueRef()).(storageEntry).value()
}

func (oe *OverlayedEntry[V]) Extrinsics() *btree.Set[uint32] {
	set := btree.Set[uint32]{}

	for _, t := range oe.transactions {
		for _, ex := range t.extrinsics {
			set.Insert(ex)
		}
	}

	return &set
}

func (oe *OverlayedEntry[V]) PopTransaction() transaction[V] {
	if len(oe.transactions) == 0 {
		panic(proofOverlayNonEmpty)
	}

	t := oe.transactions[len(oe.transactions)-1]
	oe.transactions = oe.transactions[:len(oe.transactions)-1]

	return t
}

func (oe *OverlayedEntry[V]) TransactionExtrinsics() *extrinsics {
	if len(oe.transactions) == 0 {
		panic(proofOverlayNonEmpty)
	}

	return &oe.transactions[len(oe.transactions)-1].extrinsics
}

func (oe *OverlayedEntry[V]) SetOffchain(value V, firstWriteInTx bool, atExtrinsic *uint32) {
	if firstWriteInTx || len(oe.transactions) == 0 {
		oe.transactions = append(oe.transactions, transaction[V]{
			value:      value,
			extrinsics: extrinsics{},
		})
	} else {
		*oe.ValueRef() = value
	}

	if atExtrinsic != nil {
		oe.TransactionExtrinsics().insert(*atExtrinsic)
	}
}

// Writes a new version of a value.
// This makes sure that the old version is not overwritten and can be properly
// rolled back when required.
func (oe *OverlayedEntry[V]) Set(value StorageValue, firstWriteInTx bool, atExtrinsic *uint32) {
	var action storageEntry

	if value == nil {
		action = removeStorageEntry{}
	} else {
		action = setStorageEntry{value}
	}

	if firstWriteInTx || len(oe.transactions) == 0 {
		oe.transactions = append(oe.transactions, transaction[V]{
			value:      action.(V),
			extrinsics: extrinsics{},
		})
	} else {
		oldValue := oe.ValueRef()

		var setPrev *struct {
			data                []byte
			currentMaterialized *uint
			parentSize          uint
		}

		switch oldVal := any(oldValue).(type) {
		case appendStorageEntry:
			*oldValue = any(oldVal).(V)

			setPrev = &struct {
				data                []byte
				currentMaterialized *uint
				parentSize          uint
			}{
				data:                []byte{},
				currentMaterialized: oldVal.materializedLength,
				parentSize:          *oldVal.parentSize,
			}
		}

		*oldValue = action.(V)
		if setPrev != nil {
			transactions := len(oe.transactions)
			if transactions < 2 {
				panic("`set_prev` is not nil if the value came from parent; qed")
			}

			parent := &oe.transactions[transactions-2]

			parentValue, ok := any(parent.value).(storageEntry)
			if !ok {
				panic("Parent value is not a storage entry")
			}

			restoreAppendToParent(parentValue, setPrev.data, setPrev.currentMaterialized, setPrev.parentSize)
		}
	}

	if atExtrinsic != nil {
		oe.TransactionExtrinsics().insert(*atExtrinsic)
	}
}

// Append content to a value, updating a prefixed compact encoded length.
// This makes sure that the old version is not overwritten and can be properly
// rolled back when required.
// This avoid copying value from previous transaction.
func (oe *OverlayedEntry[V]) Append(
	element StorageValue,
	firstWriteInTx bool,
	init func() StorageValue,
	atExtrinsic *uint32,
) {
	var data []byte
	var currentLength uint
	var materializedLength *uint
	var parentSize *uint

	replace := true

	if len(oe.transactions) == 0 {
		initValue := init()
		storageAppend := newStorageAppend(&initValue)

		// Either the init value is a SCALE list like value to that the `element` gets appended
		// or the value is reset to `element`.
		length := storageAppend.extractLength()
		if length != nil {
			storageAppend.appendRaw(element)
			data = initValue
			currentLength = *length + 1
			materializedLength = length
		} else {
			data = element
			currentLength = 1
			materializedLength = nil
		}

		oe.transactions = append(oe.transactions, transaction[V]{
			value: any(&appendStorageEntry{
				data:               data,
				currentLength:      currentLength,
				materializedLength: materializedLength,
				parentSize:         nil,
			}).(V),
			extrinsics: extrinsics{},
		})
	} else if firstWriteInTx {
		parent := *oe.ValueRef()

		switch entry := any(parent).(type) {
		case removeStorageEntry:
			data = element
			currentLength = 1
			materializedLength = nil
			parentSize = nil
		case *appendStorageEntry:
			parentLen := uint(len(entry.data))
			newStorageAppend(&entry.data).appendRaw(element)
			data = entry.data
			currentLength = entry.currentLength + 1
			materializedLength = entry.materializedLength
			parentSize = &parentLen
		case setStorageEntry:
			// For compatibility: append if there is a encoded length, overwrite
			// with value otherwhise.
			length := newStorageAppend(&entry.data).extractLength()
			if length != nil {
				newStorageAppend(&entry.data).appendRaw(element)
				data = entry.data
				currentLength = *length + 1
				materializedLength = length
				parentSize = nil
			} else {
				// overwrite, same as empty case.
				data = element
				currentLength = 1
				materializedLength = nil
				parentSize = nil
			}
		}

		oe.transactions = append(oe.transactions, transaction[V]{
			value: any(&appendStorageEntry{
				data:               data,
				currentLength:      currentLength,
				materializedLength: materializedLength,
				parentSize:         parentSize,
			}).(V),
			extrinsics: extrinsics{},
		})
	} else {
		// not first transaction write
		oldValue := oe.ValueRef()

		switch oldVal := any(*oldValue).(type) {
		case removeStorageEntry:
			data = element
			currentLength = 1
			materializedLength = nil
		case setStorageEntry:
			// Note that when the data here is not initialised with append,
			// and still starts with a valid compact u32 we can have totally broken
			// encoding.
			append := newStorageAppend(&oldVal.data)

			len := append.extractLength()

			// For compatibility: append if there is a encoded length, overwrite
			// with value otherwhise.
			if len != nil {
				append.appendRaw(element)
				data = oldVal.data
				currentLength = *len + 1
				materializedLength = len
			} else {
				data = element
				currentLength = 1
				materializedLength = nil
			}
		case *appendStorageEntry:
			newStorageAppend(&oldVal.data).appendRaw(element)
			oldVal.currentLength += 1
			replace = false
			*oldValue = any(oldVal).(V)
		}

		if replace {
			*oldValue = any(appendStorageEntry{
				data:               data,
				currentLength:      currentLength,
				materializedLength: materializedLength,
				parentSize:         nil,
			}).(V)
		}
	}
	if atExtrinsic != nil {
		oe.TransactionExtrinsics().insert(*atExtrinsic)
	}
}

func restoreAppendToParent(
	parent storageEntry,
	currentData []byte,
	currentMaterialized *uint,
	targetParentSize uint,
) {
	switch parent := parent.(type) {
	case *appendStorageEntry:
		prev := 0
		new := 0

		if parent.materializedLength != nil {
			prev = compactLen(*parent.materializedLength)
		}

		if currentMaterialized != nil {
			new = compactLen(*currentMaterialized)
		}

		diff := math.Abs(float64(prev - new))
		if prev >= new {
			targetParentSize -= uint(diff)
		} else {
			targetParentSize += uint(diff)
		}

		*parent.materializedLength = *currentMaterialized

		// Truncate the data to remove any extra elements
		if len(currentData) > int(targetParentSize) {
			parent.data = currentData[:targetParentSize]
		}

	default: // No value or a simple value, no need to restore
	}
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
