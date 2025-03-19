// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package statemachine

const PROOF_OVERLAY_NON_EMPTY = `
An OverlayValue is always created with at least one transaction and dropped as soon
as the last transaction is removed; qed`

type OverlayedEntry[V any] struct {
	// The individual versions of that value.
	// One entry per transactions during that the value was actually written.
	transactions Transactions[V]
}

func NewOverlayedEntry[V any]() *OverlayedEntry[V] {
	return &OverlayedEntry[V]{
		transactions: Transactions[V]{},
	}
}

func (oe *OverlayedEntry[V]) ValueRef() *V {
	if len(oe.transactions) == 0 {
		panic(PROOF_OVERLAY_NON_EMPTY)
	}

	return &oe.transactions[len(oe.transactions)-1].value
}

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

func (oe *OverlayedEntry[V]) PopTransaction() InnerValue[V] {
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

	return oe.transactions[len(oe.transactions)-1].extrinsics
}

func (oe *OverlayedEntry[V]) SetOffchain(value V, firstWriteInTx bool, atExtrinsic *uint32) {
	// TODO: test every branch

	if firstWriteInTx || len(oe.transactions) == 0 {
		oe.transactions = append(oe.transactions, InnerValue[V]{
			value:      value,
			extrinsics: &Extrinsics{},
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
		oe.transactions = append(oe.transactions, InnerValue[V]{
			value:      action.(V), //TODO: check this
			extrinsics: &Extrinsics{},
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

		oe.transactions = append(oe.transactions, InnerValue[V]{
			value: any(AppendStorageEntry{
				data:               data,
				currentLength:      currentLength,
				materializedLength: materializedLength,
				parentSize:         nil,
			}).(V),
			extrinsics: &Extrinsics{},
		})
	} else if firstWriteInTx {
		parent := *oe.ValueRef()

		switch entry := any(parent).(type) {
		case RemoveStorageEntry:
			data = element
			currentLength = 1
			materializedLength = nil
			parentSize = nil
		case AppendStorageEntry:
			parentLen := uint(len(entry.data))
			NewStorageAppend(&entry.data).AppendRaw(element)
			data = entry.data
			currentLength = entry.currentLength + 1
			materializedLength = entry.materializedLength
			parentSize = &parentLen
		case SetStorageEntry:
			length := NewStorageAppend(&entry.data).ExtractLength()
			if length != nil {
				NewStorageAppend(&entry.data).AppendRaw(element)
				data = entry.data
				currentLength = *length + 1
				materializedLength = length
				parentSize = nil
			} else {
				data = element
				currentLength = 1
				materializedLength = nil
				parentSize = nil
			}
		}

		oe.transactions = append(oe.transactions, InnerValue[V]{
			value: any(AppendStorageEntry{
				data:               data,
				currentLength:      currentLength,
				materializedLength: materializedLength,
				parentSize:         parentSize,
			}).(V),
			extrinsics: &Extrinsics{},
		})
	} else {
		oldValue := oe.ValueRef()

		switch oldVal := any(*oldValue).(type) {
		case RemoveStorageEntry:
			data = element
			currentLength = 1
			materializedLength = nil
		case SetStorageEntry:
			append := NewStorageAppend(&oldVal.data)

			len := append.ExtractLength()
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
		case AppendStorageEntry:
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

		if prev >= new {
			targetParentSize += uint(prev - new)
		} else {
			targetParentSize -= uint(new - prev)
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
