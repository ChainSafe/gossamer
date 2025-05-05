// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package overlayedchanges

import (
	"iter"

	"github.com/tidwall/btree"
)

// Describes in which mode the node is currently executing.
type executionMode = uint8

const (
	// Executing in client mode: Removal of all transactions possible.
	executionModeClient = iota
	// Executing in runtime mode: Transactions started by the client are protected.
	executionModeRuntime
)

// Dirty keys are a set of keys that have been modified in each transaction.
type dirtyKeysSets[K comparable] []map[K]struct{}

// Inserts a key into the dirty set.
// Returns true iff we currently have at least one open transaction and if this
// is the first write to the given key in that transaction.
func (dks dirtyKeysSets[K]) insertDirty(key K) bool {
	if len(dks) == 0 {
		return false
	}

	last := dks[len(dks)-1]
	_, has := last[key]
	firstWrite := !has

	last[key] = struct{}{}
	return firstWrite
}

// Get the keys modified in the last transaction.
func (dks *dirtyKeysSets[K]) Pop() (map[K]struct{}, bool) {
	if len(*dks) == 0 {
		return make(map[K]struct{}), false
	}

	set := (*dks)[len(*dks)-1]
	*dks = (*dks)[:len(*dks)-1]

	return set, true
}

// A transaction that has been executed on a value and optional extrinsic
type transaction[V any] struct {
	// Current value. nil if value has been deleted.
	value V
	// The set of extrinsic indices where the values has been changed.
	extrinsics extrinsics
}

// History of value, with removal support.
type overlayedValue = OverlayedStorageEntry

// Change set for basic key value with extrinsics index recording and removal support.
type overlayedChangeSet struct {
	OverlayedMap[string, storageEntry, *OverlayedStorageEntry]
}

func newOverlayedChangeSetFromBtreeMap(storage btree.Map[string, []byte]) overlayedChangeSet {
	var changes btree.Map[string, *OverlayedStorageEntry]
	storage.Scan(func(k string, v []byte) bool {
		changes.Set(k, &OverlayedStorageEntry{
			GenericOverlayedEntry: GenericOverlayedEntry[storageEntry]{
				transactions: []transaction[storageEntry]{
					{
						value:      setStorageEntry{data: v},
						extrinsics: extrinsics{},
					},
				},
			},
		})
		return true
	})

	return overlayedChangeSet{
		OverlayedMap: OverlayedMap[string, storageEntry, *OverlayedStorageEntry]{
			changes:               changes,
			dirtyKeys:             dirtyKeysSets[string]{},
			numClientTransactions: 0,
			executionMode:         executionModeClient,
		},
	}
}

func newOverlayedChangeSet() overlayedChangeSet {
	return overlayedChangeSet{
		NewOverlayedMap[string, storageEntry, *OverlayedStorageEntry](),
	}
}

func (oc *overlayedChangeSet) Clone() overlayedChangeSet {
	return overlayedChangeSet{
		oc.OverlayedMap.Clone(),
	}
}

// Create a new changeset at the same transaction state but without any contents.
// This changeset might be created when there are already open transactions.
// We need to catch up here so that the child is at the same transaction depth.
func (oc *overlayedChangeSet) SpawnChild() overlayedChangeSet {
	return overlayedChangeSet{
		oc.OverlayedMap.SpawnChild(),
	}
}

// set a new value for the specified key.
// Can be rolled back or committed when called inside a transaction.
func (oc *overlayedChangeSet) set(key StorageKey, value StorageValue, atExtrinsic *uint32) {
	keyString := string(key)
	overlayed, has := oc.changes.Get(keyString)
	if !has {
		overlayed = NewOverlayedStorageEntry()
		oc.changes.Set(keyString, overlayed)
	}

	overlayed.Set(value, oc.dirtyKeys.insertDirty(keyString), atExtrinsic)
}

// Append bytes to an existing content.
func (oc *overlayedChangeSet) appendStorage(
	key StorageKey,
	value StorageValue,
	init func() StorageValue,
	atExtrinsic *uint32,
) {
	keyString := string(key)
	overlayed, has := oc.changes.Get(keyString)
	if !has {
		overlayed = NewOverlayedStorageEntry()
	}

	firstWriteInTx := oc.dirtyKeys.insertDirty(keyString)
	overlayed.Append(value, firstWriteInTx, init, atExtrinsic)
	oc.changes.Set(keyString, overlayed)
}

// Returns an iterator over all changes that follow the supplied `key`.
func (oc *overlayedChangeSet) changesAfter(key StorageKey) iter.Seq2[StorageKey, *overlayedValue] {
	return func(yield func(StorageKey, *overlayedValue) bool) {
		oc.changes.Ascend(string(key), func(k string, v *overlayedValue) bool {
			// the pivot is included so we have to skip it in the resulting iterator
			return k <= string(key) || yield([]byte(k), v)
		})
	}
}

// Set all values to deleted which are matched by the predicate.
// Can be rolled back or committed when called inside a transaction.
func (oc *overlayedChangeSet) clearWhere(predicate func([]byte, *overlayedValue) bool, atExtrinsic *uint32) uint32 {
	count := uint32(0)
	for k, v := range oc.Changes() {
		if v == nil {
			continue
		}

		if predicate([]byte(k), v) {
			switch any(*v.ValueRef()).(type) {
			case appendStorageEntry, setStorageEntry:
				count++
			}
			v.Set(nil, oc.dirtyKeys.insertDirty(string(k)), atExtrinsic)
		}
	}

	return count
}

// Call this when control returns from the runtime.
// This rollbacks all dangling transaction left open by the runtime.
// Calling this while already outside the runtime will return an error.
func (oc *overlayedChangeSet) exitRuntime() error {
	if oc.executionMode != executionModeRuntime {
		return errorNotInRuntime
	}

	oc.executionMode = executionModeClient
	if oc.HasOpenRuntimeTransactions() {
		logger.Warnf("%d storage transactions are left open by the runtime. Those will be rolled back.",
			oc.TransactionDepth()-oc.numClientTransactions)
	}

	for oc.HasOpenRuntimeTransactions() {
		if oc.rollbackTransaction() != nil {
			panic("The loop condition checks that the transaction depth is > 0; qed")
		}
	}

	return nil
}

// Rollback the last transaction started by `start_transaction`.
// Any changes made during that transaction are discarded. Returns an error if
// there is no open transaction that can be rolled back.
func (oc *overlayedChangeSet) rollbackTransaction() error {
	return oc.closeTransaction(true)
}

// Commit the last transaction started by `start_transaction`.
// Any changes made during that transaction are committed. Returns an error if
// there is no open transaction that can be committed.
func (oc *overlayedChangeSet) commitTransaction() error {
	return oc.closeTransaction(false)
}

// Internal method to close the transaction and either commit or roll back the changes.
func (oc *overlayedChangeSet) closeTransaction(rollback bool) error {
	if oc.executionMode == executionModeRuntime && !oc.HasOpenRuntimeTransactions() {
		return errorNoOpenTransaction
	}

	lastTransactions, has := oc.dirtyKeys.Pop()
	if !has {
		return errorNoOpenTransaction
	}

	for key := range lastTransactions {
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
			case *appendStorageEntry:
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
				last := oc.dirtyKeys[len(oc.dirtyKeys)-1]
				_, hasPredecessor = last[key]
				last[key] = struct{}{}
			} else {
				hasPredecessor = len(overlayed.transactions) > 1
			}

			if hasPredecessor {
				commitedTx := overlayed.PopTransaction()
				mergeAppends := false

				if entry, ok := commitedTx.value.(*appendStorageEntry); ok && entry.parentSize != nil {
					parent := *overlayed.ValueRef()
					if parentEntry, ok := any(parent).(*appendStorageEntry); ok {
						mergeAppends = true
						*entry.parentSize = *parentEntry.parentSize
					}
				}

				if mergeAppends {
					*overlayed.ValueRef() = commitedTx.value
				} else {
					removed := *overlayed.ValueRef()
					*overlayed.ValueRef() = commitedTx.value

					if entry, ok := removed.(*appendStorageEntry); ok {
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

				overlayed.TransactionExtrinsics().extend(commitedTx.extrinsics)
			}
		}
	}

	return nil
}
