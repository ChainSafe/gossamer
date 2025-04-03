// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package overlayedchanges

import (
	"slices"

	"github.com/tidwall/btree"
)

const proofOverlayNonEmptyMsg = `
An OverlayValue is always created with at least one transaction and dropped as soon
as the last transaction is removed; qed`

type OverlayedEntry[V any] interface {
	ValueRef() *V
	Extrinsics() *btree.Set[uint32]
	SetOffchain(value V, firstWriteInTx bool, atExtrinsic *uint32)
	PopTransaction() transaction[V]
	Transactions() []transaction[V]
	TransactionExtrinsics() *extrinsics
	Clone() OverlayedEntry[V]
}

// An overlay that contains all versions of a value for a specific key.
type GenericOverlayedEntry[V any] struct {
	// The individual versions of that value.
	// One entry per transactions during that the value was actually written.
	transactions []transaction[V]
}

func NewGenericOverlayedEntry[V any]() *GenericOverlayedEntry[V] {
	return &GenericOverlayedEntry[V]{
		transactions: []transaction[V]{},
	}
}

func (oe GenericOverlayedEntry[V]) Clone() OverlayedEntry[V] {
	return &GenericOverlayedEntry[V]{
		transactions: slices.Clone(oe.transactions),
	}
}

func (oe *GenericOverlayedEntry[V]) Transactions() []transaction[V] {
	return oe.transactions
}

func (oe *GenericOverlayedEntry[V]) ValueRef() *V {
	if len(oe.transactions) == 0 {
		panic(proofOverlayNonEmptyMsg)
	}

	return &oe.transactions[len(oe.transactions)-1].value
}

func (oe GenericOverlayedEntry[V]) Extrinsics() *btree.Set[uint32] {
	set := btree.Set[uint32]{}

	for _, t := range oe.transactions {
		for _, ex := range t.extrinsics {
			set.Insert(ex)
		}
	}

	return &set
}

func (oe *GenericOverlayedEntry[V]) PopTransaction() transaction[V] {
	if len(oe.transactions) == 0 {
		panic(proofOverlayNonEmptyMsg)
	}

	t := oe.transactions[len(oe.transactions)-1]
	oe.transactions = oe.transactions[:len(oe.transactions)-1]

	return t
}

func (oe *GenericOverlayedEntry[V]) TransactionExtrinsics() *extrinsics {
	if len(oe.transactions) == 0 {
		panic(proofOverlayNonEmptyMsg)
	}

	return &oe.transactions[len(oe.transactions)-1].extrinsics
}

func (oe *GenericOverlayedEntry[V]) SetOffchain(value V, firstWriteInTx bool, atExtrinsic *uint32) {
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
