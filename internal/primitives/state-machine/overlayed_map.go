// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package statemachine

import (
	"errors"
	"iter"

	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/tidwall/btree"
)

var logger = log.NewFromGlobal(log.AddContext("pkg", "statemachine"))

var errorAlreadyInRuntime = errors.New("already in runtime")
var errorNotInRuntime = errors.New("not in runtime")
var errorNoOpenTransaction = errors.New("no open transaction")

type OverlayedMap[K ordered, V any] struct {
	// Stores the changes that this overlay constitutes.
	changes btree.Map[K, *OverlayedEntry[V]]
	// Stores which keys are dirty per transaction. Needed in order to determine which
	// values to merge into the parent transaction on commit. The length of this vector
	// therefore determines how many nested transactions are currently open (depth).
	dirtyKeys DirtyKeysSets[K]
	// The number of how many transactions beginning from the first transactions are started
	// by the client. Those transactions are protected against close (commit, rollback)
	// when in runtime mode.
	numClientTransactions uint
	// Determines whether the node is using the overlay from the client or the runtime.
	executionMode ExecutionMode
}

func NewOverlayedMap[K ordered, V any]() OverlayedMap[K, V] {
	return OverlayedMap[K, V]{
		dirtyKeys:             DirtyKeysSets[K]{},
		numClientTransactions: 0,
		executionMode:         ExecutionModeClient,
	}
}

func (om *OverlayedMap[K, V]) SpawnChild() OverlayedMap[K, V] {
	return OverlayedMap[K, V]{
		dirtyKeys:             make(DirtyKeysSets[K], om.TransactionDepth()),
		numClientTransactions: om.numClientTransactions,
		executionMode:         om.executionMode,
	}
}

func (om *OverlayedMap[K, V]) IsEmpty() bool {
	return om.changes.Len() == 0
}

func (om *OverlayedMap[K, V]) Get(key K) *OverlayedEntry[V] {
	value, present := om.changes.Get(key)
	if !present {
		return nil
	}

	return value
}

func (om *OverlayedMap[K, V]) SetOffchain(key K, value V, atExtrinsic *uint32) {
	overlayed := om.Get(key)

	if overlayed == nil {
		overlayed = NewOverlayedEntry[V]()
	}

	overlayed.SetOffchain(value, insertDirty(&om.dirtyKeys, key), atExtrinsic)
}

func (om *OverlayedMap[K, V]) Changes() iter.Seq2[K, *OverlayedEntry[V]] {
	return func(yield func(K, *OverlayedEntry[V]) bool) {
		om.changes.Scan(yield)
	}
}

func (om *OverlayedMap[K, V]) DrainCommited() iter.Seq2[K, V] {
	if om.TransactionDepth() != 0 {
		panic("Drain is not allowed with open transactions.")
	}

	return func(yield func(K, V) bool) {
		om.changes.Scan(func(k K, v *OverlayedEntry[V]) bool {
			return yield(k, v.PopTransaction().value)
		})
	}
}

func (om *OverlayedMap[K, V]) TransactionDepth() uint {
	return uint(len(om.dirtyKeys))
}

func (om *OverlayedMap[K, V]) EnterRuntime() error {
	if om.executionMode == ExecutionModeRuntime {
		return errorAlreadyInRuntime
	}

	om.executionMode = ExecutionModeRuntime
	om.numClientTransactions = om.TransactionDepth()
	return nil
}

func (om *OverlayedMap[K, V]) ExitRuntimeoffchain() error {
	if om.executionMode != ExecutionModeRuntime {
		return errorNotInRuntime
	}

	om.executionMode = ExecutionModeClient

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

func (om *OverlayedMap[K, V]) StartTransaction() {
	om.dirtyKeys = append(om.dirtyKeys, btree.Set[K]{})
}

func (om *OverlayedMap[K, V]) RollbackTransactionOffchain() error {
	return om.CloseTransactionOffchain(true)
}

func (om *OverlayedMap[K, V]) CommitTransactionOffchain() error {
	return om.CloseTransactionOffchain(false)
}

func (om *OverlayedMap[K, V]) CloseTransactionOffchain(rollback bool) error {
	if om.executionMode == ExecutionModeRuntime && !om.HasOpenRuntimeTransactions() {
		return errorNoOpenTransaction
	}

	lastTransaction, has := om.dirtyKeys.Pop()
	if !has {
		return errorNoOpenTransaction
	}

	lastTransaction.Scan(func(key K) bool {
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

			if len(overlayed.transactions) == 0 {
				om.changes.Delete(key)
			}
		} else {
			var hasPredecessor bool

			if len(om.dirtyKeys) > 0 {
				last := om.dirtyKeys[len(om.dirtyKeys)-1]

				hasPredecessor = last.Contains(key)
				last.Insert(key)
			} else {
				hasPredecessor = len(overlayed.transactions) > 1
			}

			if hasPredecessor {
				droppedTx := overlayed.PopTransaction()
				*overlayed.ValueRef() = droppedTx.value
				overlayed.TransactionExtrinsics().Extend(*droppedTx.extrinsics)
			}
		}

		return true
	})

	return nil
}

func (om *OverlayedMap[K, V]) HasOpenRuntimeTransactions() bool {
	return om.TransactionDepth() > om.numClientTransactions
}

func insertDirty[K ordered](set *DirtyKeysSets[K], key K) bool {
	if set == nil || len(*set) == 0 {
		return false
	}

	(*set)[len(*set)-1].Insert(key)
	return true
}
