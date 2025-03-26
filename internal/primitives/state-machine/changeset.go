// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package statemachine

import (
	"iter"

	"github.com/tidwall/btree"
)

// From btree.Set constraints
type ordered interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr |
		~float32 | ~float64 | ~string
}

// Describes in which mode the node is currently executing.
type ExecutionMode = uint8

const (
	// Executing in client mode: Removal of all transactions possible.
	ExecutionModeClient = iota
	// Executing in runtime mode: Transactions started by the client are protected.
	ExecutionModeRuntime
)

// Dirty keys are a set of keys that have been modified in each transaction.
type DirtyKeysSets[K ordered] []btree.Set[K]

// Inserts a key into the dirty set.
// Returns true iff we currently have at least one open transaction and if this
// is the first write to the given key in that transaction.
func (dks DirtyKeysSets[K]) insertDirty(key K) bool {
	if len(dks) == 0 {
		return false
	}

	last := &dks[len(dks)-1]
	firstWrite := !last.Contains(key)

	last.Insert(key)
	return firstWrite
}

// Get the keys modified in the last transaction.
func (dks *DirtyKeysSets[K]) Pop() (btree.Set[K], bool) {
	if len(*dks) == 0 {
		return btree.Set[K]{}, false
	}

	set := (*dks)[len(*dks)-1]
	*dks = (*dks)[:len(*dks)-1]

	return set, true
}

// A transaction that has been executed on a value and optional extrinsic
type Transaction[V any] struct {
	// Current value. nil if value has been deleted.
	value V
	// The set of extrinsic indices where the values has been changed.
	extrinsics Extrinsics
}

// History of value, with removal support.
type OverlayedValue = OverlayedEntry[StorageEntry]

// Change set for basic key value with extrinsics index recording and removal support.
type OverlayedChangeSet struct {
	OverlayedMap[string, StorageEntry]
}

func NewOverlayedChangeSet() OverlayedChangeSet {
	return OverlayedChangeSet{
		NewOverlayedMap[string, StorageEntry](),
	}
}

func (oc *OverlayedChangeSet) Clone() OverlayedChangeSet {
	return OverlayedChangeSet{
		oc.OverlayedMap.Clone(),
	}
}

// Set a new value for the specified key.
// Can be rolled back or committed when called inside a transaction.
func (oc *OverlayedChangeSet) Set(key StorageKey, value StorageValue, atExtrinsic *uint32) {
	keyString := string(key)
	overlayed, has := oc.changes.Get(keyString)
	if !has {
		overlayed = NewOverlayedEntry[StorageEntry]()
	}

	overlayed.Set(value, oc.dirtyKeys.insertDirty(keyString), atExtrinsic)
	oc.changes.Set(keyString, overlayed)
}

// Append bytes to an existing content.
func (oc *OverlayedChangeSet) AppendStorage(
	key StorageKey,
	value StorageValue,
	init func() StorageValue,
	atExtrinsic *uint32,
) {
	keyString := string(key)
	overlayed, has := oc.changes.Get(keyString)
	if !has {
		overlayed = NewOverlayedEntry[StorageEntry]()
	}

	firstWriteInTx := oc.dirtyKeys.insertDirty(keyString)
	overlayed.Append(value, firstWriteInTx, init, atExtrinsic)
	oc.changes.Set(keyString, overlayed)
}

// Returns an iterator over all changes that follow the supplied `key`.
func (oc *OverlayedChangeSet) ChangesAfter(key StorageKey) iter.Seq2[StorageKey, *OverlayedValue] {
	return func(yield func(StorageKey, *OverlayedValue) bool) {
		oc.changes.Scan(func(k string, v *OverlayedValue) bool {
			if k > string(key) && !yield([]byte(k), v) {
				return false
			}
			return true
		})
	}
}

// Set all values to deleted which are matched by the predicate.
// Can be rolled back or committed when called inside a transaction.
func (oc *OverlayedChangeSet) ClearWhere(predicate func([]byte, *OverlayedValue) bool, atExtrinsic *uint32) {
	count := 0
	for k, v := range oc.Changes() {
		if predicate([]byte(k), v) {
			v.Set(nil, oc.dirtyKeys.insertDirty(k), atExtrinsic)
			if v != nil {
				switch any(*v).(type) {
				case AppendStorageEntry, SetStorageEntry:
					count++
				}
			}
		}
	}
}

// Call this when control returns from the runtime.
// This rollbacks all dangling transaction left open by the runtime.
// Calling this while already outside the runtime will return an error.
func (oc *OverlayedChangeSet) ExitRuntime() error {
	if oc.executionMode != ExecutionModeRuntime {
		return errorNotInRuntime
	}

	oc.executionMode = ExecutionModeClient
	if oc.HasOpenRuntimeTransactions() {
		logger.Warnf("%d storage transactions are left open by the runtime. Those will be rolled back.",
			oc.TransactionDepth()-oc.numClientTransactions)
	}

	for oc.HasOpenRuntimeTransactions() {
		if oc.RollbackTransaction() != nil {
			panic("The loop confidtion checks that the transaction depth is > 0; qed")
		}
	}

	return nil
}

// Rollback the last transaction started by `start_transaction`.
// Any changes made during that transaction are discarded. Returns an error if
// there is no open transaction that can be rolled back.
func (oc *OverlayedChangeSet) RollbackTransaction() error {
	return oc.closeTransaction(true)
}

// Commit the last transaction started by `start_transaction`.
// Any changes made during that transaction are committed. Returns an error if
// there is no open transaction that can be committed.
func (oc *OverlayedChangeSet) CommitTransaction() error {
	return oc.closeTransaction(false)
}

// Internal method to close the transaction and either commit or roll back the changes.
func (oc *OverlayedChangeSet) closeTransaction(rollback bool) error {
	if oc.executionMode == ExecutionModeRuntime && !oc.HasOpenRuntimeTransactions() {
		return errorNoOpenTransaction
	}

	lastTransactions, has := oc.dirtyKeys.Pop()
	if !has {
		return errorNoOpenTransaction
	}

	lastTransactions.Scan(func(key string) bool {
		overlayed, has := oc.changes.GetMut(key)
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
			case *AppendStorageEntry:
				if entry.parentSize != nil {
					if len(overlayed.transactions) == 0 {
						panic("AppendStorageEntry should have transactions")
					}
					restoreAppendToParent(
						*overlayed.ValueRef(),
						entry.data,
						entry.materializedLength,
						*entry.parentSize,
					)
				}
			default: // do nothing
			}

			if len(overlayed.transactions) == 0 {
				oc.changes.Delete(key)
			}
		} else {
			var hasPredecessor bool

			if len(oc.dirtyKeys) > 0 {
				last := &oc.dirtyKeys[len(oc.dirtyKeys)-1]
				hasPredecessor = last.Contains(key)
				last.Insert(key)
			} else {
				hasPredecessor = len(overlayed.transactions) > 1
			}

			if hasPredecessor {
				commitedTx := overlayed.PopTransaction()
				mergeAppends := false

				if entry, ok := commitedTx.value.(*AppendStorageEntry); ok && entry.parentSize != nil {
					parent := *overlayed.ValueRef()
					if parentEntry, ok := any(parent).(*AppendStorageEntry); ok {
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

					if entry, ok := removed.(*AppendStorageEntry); ok {
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

				overlayed.TransactionExtrinsics().Extend(commitedTx.extrinsics)
			}
		}

		return true
	})

	return nil
}
