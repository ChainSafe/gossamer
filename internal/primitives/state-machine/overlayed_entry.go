// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package statemachine

import "math"

const PROOF_OVERLAY_NON_EMPTY = `
An OverlayValue is always created with at least one transaction and dropped as soon
as the last transaction is removed; qed`

// An overlay that contains all versions of a value for a specific key.
type OverlayedEntry[V any] struct {
	// The individual versions of that value.
	// One entry per transactions during that the value was actually written.
	transactions []Transaction[V]
}

func NewOverlayedEntry[V any]() *OverlayedEntry[V] {
	return &OverlayedEntry[V]{
		transactions: []Transaction[V]{},
	}
}

func (oe *OverlayedEntry[V]) ValueRef() *V {
	if len(oe.transactions) == 0 {
		panic(PROOF_OVERLAY_NON_EMPTY)
	}

	return &oe.transactions[len(oe.transactions)-1].value
}

// The value as seen by the current transaction.
func (oe *OverlayedEntry[V]) StorageValue() StorageValue {
	return any(*oe.ValueRef()).(StorageEntry).optionalValue()
}

func (oe *OverlayedEntry[V]) IntoValue() V {
	value := *oe.ValueRef()
	oe.transactions = oe.transactions[:len(oe.transactions)-1]

	return value
}

func (oe *OverlayedEntry[V]) Extrinsics() map[uint32]struct{} {
	set := make(map[uint32]struct{}, 0)

	for _, t := range oe.transactions {
		t.extrinsics.CopyExtrinsicsInto(set)
	}

	return set
}

func (oe *OverlayedEntry[V]) PopTransaction() Transaction[V] {
	if len(oe.transactions) == 0 {
		panic(PROOF_OVERLAY_NON_EMPTY)
	}

	t := oe.transactions[len(oe.transactions)-1]
	oe.transactions = oe.transactions[:len(oe.transactions)-1]

	return t
}

func (oe *OverlayedEntry[V]) TransactionExtrinsics() *Extrinsics {
	if len(oe.transactions) == 0 {
		panic(PROOF_OVERLAY_NON_EMPTY)
	}

	return &oe.transactions[len(oe.transactions)-1].extrinsics
}

func (oe *OverlayedEntry[V]) SetOffchain(value V, firstWriteInTx bool, atExtrinsic *uint32) {
	// TODO: test every branch

	if firstWriteInTx || len(oe.transactions) == 0 {
		oe.transactions = append(oe.transactions, Transaction[V]{
			value:      value,
			extrinsics: Extrinsics{},
		})
	} else {
		*oe.ValueRef() = value
	}

	if atExtrinsic != nil {
		oe.TransactionExtrinsics().Insert(*atExtrinsic)
	}
}

// Writes a new version of a value.
// This makes sure that the old version is not overwritten and can be properly
// rolled back when required.
func (oe *OverlayedEntry[V]) Set(value StorageValue, firstWriteInTx bool, atExtrinsic *uint32) {
	var action StorageEntry

	if value == nil {
		action = RemoveStorageEntry{}
	} else {
		action = SetStorageEntry{value}
	}

	if firstWriteInTx || len(oe.transactions) == 0 {
		oe.transactions = append(oe.transactions, Transaction[V]{
			value:      action.(V), //TODO: check this
			extrinsics: Extrinsics{},
		})
	} else {
		oldValue := oe.ValueRef()

		var setPrev *struct {
			data                []byte
			currentMaterialized *uint32
			parentSize          uint
		}

		switch oldVal := any(oldValue).(type) {
		case AppendStorageEntry:
			*oldValue = any(oldVal).(V)

			setPrev = &struct {
				data                []byte
				currentMaterialized *uint32
				parentSize          uint
			}{
				data:                []byte{},
				currentMaterialized: oldVal.materializedLength,
				parentSize:          *oldVal.parentSize,
			}
		}

		*oldValue = action.(V) //TODO: check this
		if setPrev != nil {
			transactions := len(oe.transactions)
			if transactions < 2 {
				panic("`set_prev` is not nil if the value came from parent; qed")
			}

			parent := &oe.transactions[transactions-2]

			parentValue, ok := any(parent.value).(StorageEntry)
			if !ok {
				panic("Parent value is not a storage entry")
			}

			restoreAppendToParent(parentValue, setPrev.data, setPrev.currentMaterialized, setPrev.parentSize)
		}
	}

	if atExtrinsic != nil {
		oe.TransactionExtrinsics().Insert(*atExtrinsic)
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
	var currentLength uint32
	var materializedLength *uint32
	var parentSize *uint

	replace := true

	if len(oe.transactions) == 0 {
		initValue := init()
		storageAppend := NewStorageAppend(&initValue)

		// Either the init value is a SCALE list like value to that the `element` gets appended
		// or the value is reset to `element`.
		length := storageAppend.ExtractLength()
		if length != nil {
			storageAppend.AppendRaw(element)
			data = initValue
			currentLength = *length + 1
			materializedLength = length
		} else {
			data = element
			currentLength = 1
			materializedLength = nil
		}

		oe.transactions = append(oe.transactions, Transaction[V]{
			value: any(&AppendStorageEntry{
				data:               data,
				currentLength:      currentLength,
				materializedLength: materializedLength,
				parentSize:         nil,
			}).(V),
			extrinsics: Extrinsics{},
		})
	} else if firstWriteInTx {
		parent := *oe.ValueRef()

		switch entry := any(parent).(type) {
		case RemoveStorageEntry:
			data = element
			currentLength = 1
			materializedLength = nil
			parentSize = nil
		case *AppendStorageEntry:
			parentLen := uint(len(entry.data))
			NewStorageAppend(&entry.data).AppendRaw(element)
			data = entry.data
			currentLength = entry.currentLength + 1
			materializedLength = entry.materializedLength
			parentSize = &parentLen
		case SetStorageEntry:
			// For compatibility: append if there is a encoded length, overwrite
			// with value otherwhise.
			length := NewStorageAppend(&entry.data).ExtractLength()
			if length != nil {
				NewStorageAppend(&entry.data).AppendRaw(element)
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

		oe.transactions = append(oe.transactions, Transaction[V]{
			value: any(&AppendStorageEntry{
				data:               data,
				currentLength:      currentLength,
				materializedLength: materializedLength,
				parentSize:         parentSize,
			}).(V),
			extrinsics: Extrinsics{},
		})
	} else {
		// not first transaction write
		oldValue := oe.ValueRef()

		switch oldVal := any(*oldValue).(type) {
		case RemoveStorageEntry:
			data = element
			currentLength = 1
			materializedLength = nil
		case SetStorageEntry:
			// Note that when the data here is not initialized with append,
			// and still starts with a valid compact u32 we can have totally broken
			// encoding.
			append := NewStorageAppend(&oldVal.data)

			len := append.ExtractLength()

			// For compatibility: append if there is a encoded length, overwrite
			// with value otherwhise.
			if len != nil {
				append.AppendRaw(element)
				data = oldVal.data
				currentLength = *len + 1
				materializedLength = len
			} else {
				data = element
				currentLength = 1
				materializedLength = nil
			}
		case *AppendStorageEntry:
			NewStorageAppend(&oldVal.data).AppendRaw(element)
			oldVal.currentLength += 1
			replace = false
			*oldValue = any(oldVal).(V)
		}

		if replace {
			*oldValue = any(AppendStorageEntry{
				data:               data,
				currentLength:      currentLength,
				materializedLength: materializedLength,
				parentSize:         nil,
			}).(V)
		}
	}
	if atExtrinsic != nil {
		oe.TransactionExtrinsics().Insert(*atExtrinsic)
	}
}

func restoreAppendToParent(
	parent StorageEntry,
	currentData []byte,
	currentMaterialized *uint32,
	targetParentSize uint,
) {
	switch parent := parent.(type) {
	case *AppendStorageEntry:
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

func compactLen(val uint32) int {
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
