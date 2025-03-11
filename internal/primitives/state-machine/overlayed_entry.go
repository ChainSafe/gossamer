// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package statemachine

import "github.com/tidwall/btree"

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

func (oe *OverlayedEntry[V]) Value() V {
	value := *oe.ValueRef()
	oe.transactions = oe.transactions[:len(oe.transactions)-1]

	return value
}

func (oe *OverlayedEntry[V]) Extrinsics() btree.Set[uint32] {
	set := btree.Set[uint32]{}

	for _, t := range oe.transactions {
		t.extrinsics.CopyExtrinsicsInto(&set)
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

	return &oe.transactions[len(oe.transactions)-1].extrinsics
}

func (oe *OverlayedEntry[V]) SetOffchain(value V, firstWriteInTx bool, atExtrinsic *uint32) {
	// TODO: test every branch

	if firstWriteInTx || len(oe.transactions) == 0 {
		oe.transactions = append(oe.transactions, InnerValue[V]{
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
func (oe *OverlayedEntry[V]) Set(value *StorageValue, firstWriteInTx bool, atExtrinsic *uint32) {
	var action StorageEntry

	if value == nil {
		action = RemoveStorageEntry{}
	} else {
		action = SetStorageEntry{*value}
	}

	if firstWriteInTx || len(oe.transactions) == 0 {
		oe.transactions = append(oe.transactions, InnerValue[V]{
			value:      action.(V), //TODO: check this
			extrinsics: Extrinsics{},
		})
	} else {
	}

	panic("TODO")
}
