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

func (oc *OverlayedChangeSet) Set(key string, value StorageValue, atExtrinsic *uint32) {
	overlayed, has := oc.changes[key]
	if !has {
		overlayed = NewOverlayedEntry[StorageEntry]()
	}

	overlayed.Set(value, insertDirty(&oc.dirtyKeys, key), atExtrinsic)
	oc.changes[key] = overlayed
}

func (oc *OverlayedChangeSet) RollbackTransaction() error {
	return oc.closeTransaction(true)
}

func (oc *OverlayedChangeSet) CommitTransaction() error {
	return oc.closeTransaction(false)
}

func (oc *OverlayedChangeSet) closeTransaction(rollback bool) error {
	panic("TODO OverlayedChangeSet::closeTransaction")
}
