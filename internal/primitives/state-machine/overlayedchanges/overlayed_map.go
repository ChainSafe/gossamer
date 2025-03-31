// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package overlayedchanges

import (
	"errors"
	"iter"
	"reflect"
	"slices"

	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/internal/primitives/state-machine/backend"
	"github.com/tidwall/btree"
)

var logger = log.NewFromGlobal(log.AddContext("pkg", "statemachine"))

var errorAlreadyInRuntime = errors.New("already in runtime")
var errorNotInRuntime = errors.New("not in runtime")
var errorNoOpenTransaction = errors.New("no open transaction")

// Holds a set of changes with the ability modify them using nested transactions.
type OverlayedMap[K ~string, V any, E OverlayedEntry[V]] struct {
	// Stores the changes that this overlay constitutes.
	changes btree.Map[K, E]
	// Stores which keys are dirty per transaction. Needed in order to determine which
	// values to merge into the parent transaction on commit. The length of this vector
	// therefore determines how many nested transactions are currently open (depth).
	dirtyKeys dirtyKeysSets[K]
	// The number of how many transactions beginning from the first transactions are started
	// by the client. Those transactions are protected against close (commit, rollback)
	// when in runtime mode.
	numClientTransactions uint
	// Determines whether the node is using the overlay from the client or the runtime.
	executionMode executionMode
}

func NewOverlayedMap[K ~string, V any, E OverlayedEntry[V]]() OverlayedMap[K, V, E] {
	return OverlayedMap[K, V, E]{
		dirtyKeys:             dirtyKeysSets[K]{},
		numClientTransactions: 0,
		executionMode:         executionModeClient,
	}
}

func (om OverlayedMap[K, V, E]) Clone() OverlayedMap[K, V, E] {
	// Clone changes
	var changes btree.Map[K, E]
	om.changes.Scan(func(k K, v E) bool {
		changes.Set(k, v.Clone().(E))
		return true
	})

	return OverlayedMap[K, V, E]{
		changes:               changes,
		dirtyKeys:             slices.Clone(om.dirtyKeys),
		numClientTransactions: om.numClientTransactions,
		executionMode:         om.executionMode,
	}
}

// Create a new changeset at the same transaction state but without any contents.
// This changeset might be created when there are already open transactions.
// We need to catch up here so that the child is at the same transaction depth.
func (om *OverlayedMap[K, V, E]) SpawnChild() OverlayedMap[K, V, E] {
	return OverlayedMap[K, V, E]{
		dirtyKeys:             make(dirtyKeysSets[K], om.TransactionDepth()),
		numClientTransactions: om.numClientTransactions,
		executionMode:         om.executionMode,
	}
}

// True if no changes at all are contained in the change set.
func (om *OverlayedMap[K, V, E]) IsEmpty() bool {
	return om.changes.Len() == 0
}

// Get an optional reference to the value stored for the specified key.
func (om *OverlayedMap[K, V, E]) Get(key K) (E, bool) {
	return om.changes.Get(key)
}

// Set a new value for the specified key.
// Can be rolled back or committed when called inside a transaction.
func (om *OverlayedMap[K, V, E]) SetOffchain(key K, value V, atExtrinsic *uint32) {
	overlayed, has := om.Get(key)

	if !has {
		var newEntry E
		if reflect.TypeOf(newEntry).Kind() == reflect.Ptr {
			newEntryValue := reflect.New(reflect.TypeOf(newEntry).Elem())
			newEntry = newEntryValue.Interface().(E)
		} else {
			newEntry = *new(E)
		}

		overlayed = newEntry
		om.changes.Set(key, overlayed)
	}

	overlayed.SetOffchain(value, om.dirtyKeys.insertDirty(key), atExtrinsic)
}

// Get a list of all changes as seen by current transaction.
func (om *OverlayedMap[K, V, E]) Changes() iter.Seq2[backend.StorageKey, E] {
	return func(yield func(backend.StorageKey, E) bool) {
		om.changes.Scan(func(k K, v E) bool {
			return yield(backend.StorageKey(k), v)
		})
	}
}

// Return all committed changes.
// Panics if there are open transactions: `transaction_depth() > 0`
func (om *OverlayedMap[K, V, E]) DrainCommited() iter.Seq2[backend.StorageKey, V] {
	if om.TransactionDepth() != 0 {
		panic("Drain is not allowed with open transactions.")
	}

	return func(yield func(backend.StorageKey, V) bool) {
		om.changes.Scan(func(k K, v E) bool {
			return yield(backend.StorageKey(k), v.PopTransaction().value)
		})
	}
}

// Returns the current nesting depth of the transaction stack.
// A value of zero means that no transaction is open and changes are committed on write.
func (om *OverlayedMap[K, V, E]) TransactionDepth() uint {
	return uint(len(om.dirtyKeys))
}

// Call this before transferring control to the runtime.
// This protects all existing transactions from being removed by the runtime.
// Calling this while already inside the runtime will return an error.
func (om *OverlayedMap[K, V, E]) enterRuntime() error {
	if om.executionMode == executionModeRuntime {
		return errorAlreadyInRuntime
	}

	om.executionMode = executionModeRuntime
	om.numClientTransactions = om.TransactionDepth()
	return nil
}

// Call this when control returns from the runtime.
// This rollbacks all dangling transaction left open by the runtime.
// Calling this while already outside the runtime will return an error.
func (om *OverlayedMap[K, V, E]) exitRuntimeoffchain() error {
	if om.executionMode != executionModeRuntime {
		return errorNotInRuntime
	}

	om.executionMode = executionModeClient

	if om.HasOpenRuntimeTransactions() {
		logger.Warnf("%d storage transactions are left open by the runtime. Those will be rolled back.",
			om.TransactionDepth()-om.numClientTransactions)
	}

	for om.HasOpenRuntimeTransactions() {
		err := om.RollbackTransactionOffchain()
		if err != nil {
			panic("The loop condition checks that the transaction depth is > 0; qed")
		}
	}

	return nil
}

// Start a new nested transaction.
// This allows to either commit or roll back all changes that were made while this
// transaction was open. Any transaction must be closed by either `commit_transaction`
// or `rollback_transaction` before this overlay can be converted into storage changes.
// Changes made without any open transaction are committed immediately.
func (om *OverlayedMap[K, V, E]) StartTransaction() {
	om.dirtyKeys = append(om.dirtyKeys, map[K]struct{}{})
}

// Rollback the last transaction started by `start_transaction`.
// Any changes made during that transaction are discarded. Returns an error if
// there is no open transaction that can be rolled back.
func (om *OverlayedMap[K, V, E]) RollbackTransactionOffchain() error {
	return om.closeTransactionOffchain(true)
}

// Commit the last transaction started by `start_transaction`.
// Any changes made during that transaction are committed. Returns an error if
// there is no open transaction that can be committed.
func (om *OverlayedMap[K, V, E]) CommitTransactionOffchain() error {
	return om.closeTransactionOffchain(false)
}

// Internal method to close the transaction and either commit or roll back the changes.
func (om *OverlayedMap[K, V, E]) closeTransactionOffchain(rollback bool) error {
	// runtime is not allowed to close transactions started by the client
	if om.executionMode == executionModeRuntime && !om.HasOpenRuntimeTransactions() {
		return errorNoOpenTransaction
	}

	lastTransaction, has := om.dirtyKeys.Pop()
	if !has {
		return errorNoOpenTransaction
	}

	for key := range lastTransaction {
		overlayed, has := om.changes.Get(key)
		if !has {
			panic(`
				A write to an OverlayedValue is recorded in the dirty key set. Before an
				OverlayedValue is removed, its containing dirty set is removed. This
				function is only called for keys that are in the dirty set. qed\
			`)
		}

		if rollback {
			overlayed.PopTransaction()

			// We need to remove the key as an `OverlayValue` with no transactions
			// violates its invariant of always having at least one transaction.
			if len(overlayed.Transactions()) == 0 {
				om.changes.Delete(key)
			}
		} else {
			var hasPredecessor bool

			if len(om.dirtyKeys) > 0 {
				last := om.dirtyKeys[len(om.dirtyKeys)-1]

				// Check if the previous tx wrote this key
				_, hasPredecessor = last[key]
				last[key] = struct{}{}
			} else {
				// Last tx: Is there already a value in the committed set?
				// Check against one rather than empty because the current tx is still
				// in the list as it is popped later in this function.
				hasPredecessor = len(overlayed.Transactions()) > 1
			}

			// We only need to merge if there is an pre-existing value. It may be a value from
			// the previous transaction or a value committed without any open transaction.
			if hasPredecessor {
				droppedTx := overlayed.PopTransaction()
				*overlayed.ValueRef() = droppedTx.value
				overlayed.TransactionExtrinsics().extend(droppedTx.extrinsics)
			}
		}
	}

	return nil
}

// True if there are open transactions that were started by the runtime.
func (om *OverlayedMap[K, V, E]) HasOpenRuntimeTransactions() bool {
	return om.TransactionDepth() > om.numClientTransactions
}
