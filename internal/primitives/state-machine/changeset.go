// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package statemachine

import "github.com/tidwall/btree"

type ordered interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr |
		~float32 | ~float64 | ~string
}

type ExecutionMode = uint8

const (
	// Executing in client mode: Removal of all transactions possible.
	ExecutionModeClient = iota
	// Executing in runtime mode: Transactions started by the client are protected.
	ExecutionModeRuntime
)

type InnerValue[V any] struct {
	// Current value. None if value has been deleted.
	value V
	// The set of extrinsic indices where the values has been changed.
	extrinsics *Extrinsics
}

type DirtyKeysSets[K ordered] []btree.Set[K]

func (dks *DirtyKeysSets[K]) Pop() (btree.Set[K], bool) {
	if len(*dks) == 0 {
		return btree.Set[K]{}, false
	}

	set := (*dks)[len(*dks)-1]
	*dks = (*dks)[:len(*dks)-1]

	return set, true
}

type Transactions[V any] []InnerValue[V]

type OverlayedChangeSet struct {
	OverlayedMap[string, StorageEntry]
}

func NewOverlayedChangeSet() OverlayedChangeSet {
	return OverlayedChangeSet{
		NewOverlayedMap[string, StorageEntry](),
	}
}

func (oc *OverlayedChangeSet) Set(key StorageKey, value StorageValue, atExtrinsic *uint32) {
	keyString := string(key)
	overlayed, has := oc.changes[keyString]
	if !has {
		overlayed = NewOverlayedEntry[StorageEntry]()
	}

	overlayed.Set(value, insertDirty(&oc.dirtyKeys, keyString), atExtrinsic)
	oc.changes[keyString] = overlayed
}

func (oc *OverlayedChangeSet) RollbackTransaction() error {
	return oc.closeTransaction(true)
}

func (oc *OverlayedChangeSet) CommitTransaction() error {
	return oc.closeTransaction(false)
}

func (oc *OverlayedChangeSet) closeTransaction(rollback bool) error {
	if oc.executionMode == ExecutionModeRuntime && !oc.HasOpenRuntimeTransactions() {
		return errorNoOpenTransaction
	}

	lastTransaction, has := oc.dirtyKeys.Pop()
	if !has {
		return errorNoOpenTransaction
	}

	lastTransaction.Scan(func(key string) bool {
		overlayed, has := oc.changes[key]
		if !has {
			panic(`
				A write to an OverlayedValue is recorded in the dirty key set. Before an
				OverlayedValue is removed, its containing dirty set is removed. This
				function is only called for keys that are in the dirty set. qed\
			`)
		}

		if rollback {
			lastTx := overlayed.PopTransaction().value
			switch entry := lastTx.(type) {
			case AppendStorageEntry:
				if len(overlayed.transactions) == 0 {
					panic("AppendStorageEntry should have transactions")
				}
				restoreAppendToParent(
					*overlayed.ValueRef(),
					entry.data,
					entry.materializedLength,
					*entry.parentSize,
				)
			default: // do nothing
			}

			if len(overlayed.transactions) == 0 {
				delete(oc.changes, key)
			}
		} else {
			var hasPredecessor bool

			if len(oc.dirtyKeys) > 0 {
				last := oc.dirtyKeys[len(oc.dirtyKeys)-1]

				hasPredecessor = last.Contains(key)
				last.Insert(key)
			} else {
				hasPredecessor = len(overlayed.transactions) > 1
			}

			if hasPredecessor {
				commitedTx := overlayed.PopTransaction()
				mergeAppends := false

				if entry, ok := commitedTx.value.(AppendStorageEntry); ok && entry.parentSize != nil {
					if parentEntry, ok := any(overlayed.ValueRef()).(AppendStorageEntry); ok {
						mergeAppends = true
						*entry.parentSize = *parentEntry.parentSize
					}
				}

				if mergeAppends {
					*overlayed.ValueRef() = commitedTx.value
				} else {
					// TODO: check this
					removed := *overlayed.ValueRef()
					*overlayed.ValueRef() = commitedTx.value

					if entry, ok := removed.(AppendStorageEntry); ok {
						if entry.parentSize != nil {
							transactions := len(overlayed.transactions)

							if transactions < 2 {
								panic("transactions should be at least 2")
							}

							parent := overlayed.transactions[transactions-2]
							restoreAppendToParent(
								parent.value,
								entry.data,
								entry.materializedLength,
								*entry.parentSize,
							)
						}
					}
				}

				overlayed.TransactionExtrinsics().Extend(*commitedTx.extrinsics)
			}
		}

		return true
	})

	return nil
}

func (oc *OverlayedChangeSet) AppendStorage(
	key StorageKey,
	value StorageValue,
	init func() StorageValue,
	atExtrinsic *uint32,
) {
	keyString := string(key)
	overlayed, has := oc.changes[keyString]
	if !has {
		overlayed = NewOverlayedEntry[StorageEntry]()
	}

	firstWriteInTx := insertDirty(&oc.dirtyKeys, keyString)
	overlayed.Append(value, firstWriteInTx, init, atExtrinsic)
	oc.changes[keyString] = overlayed
}
