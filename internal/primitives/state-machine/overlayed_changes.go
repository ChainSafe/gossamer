// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package statemachine

import (
	"github.com/ChainSafe/gossamer/internal/primitives/core/offchain"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
)

var NoExtrinsicIndex uint32 = 0xffffffff

// StorageKey is a storage key.
type StorageKey []byte

// StorageValue is a storage value. Value can be nil
type StorageValue []byte

// StorageKeyValue is storage key and value.
type StorageKeyValue struct {
	StorageKey
	StorageValue
}

// StorageCollection is a slice of storage values.
type StorageCollection []StorageKeyValue

// ChildStorageCollection is a slice of storage values for multiple child tries.
type ChildStorageCollection []struct {
	StorageKey
	StorageCollection
}

// OffchainChangesCollection is slice of storage values.
type OffchainChangesCollection []struct {
	PrefixKey struct {
		Prefix []byte
		Key    []byte
	}
	ValueOperation offchain.OffchainOverlayedChange
}

// IndexOperations is interface constraint of [IndexOperation].
type IndexOperations interface {
	IndexOperationInsert | IndexOperationRenew
}

// IndexOperation is a transaction index operation.
type IndexOperation interface {
	isIndexOperation()
}

// IndexOperationInsert is an insert transaction into index.
type IndexOperationInsert struct {
	// Extrinsic index in the current block.
	Extrinsic uint32
	// Data content hash.
	Hash []byte
	// Indexed data size.
	Size uint32
}

// IndexOperationRenew renews existing transaction storage.
type IndexOperationRenew struct {
	// Extrinsic index in the current block.
	Extrinsic uint32
	// Referenced index hash.
	Hash []byte
}

func (IndexOperationInsert) isIndexOperation() {}
func (IndexOperationRenew) isIndexOperation()  {}

type childStorageValue struct {
	OverlayedChangeSet
	ChildInfo
}

type OffchainOverlayedChange interface {
	isOffchainOverlayedChange()
}

type (
	OffchainOverlayedChangeRemove   struct{}
	OffchainOverlayedChangeSetValue []byte
)

func (OffchainOverlayedChangeRemove) isOffchainOverlayedChange()   {}
func (OffchainOverlayedChangeSetValue) isOffchainOverlayedChange() {}

type OffchainOverlayedChanges struct {
	OverlayedMap[string, []byte]
	OffchainOverlayedChange
}

// Storage transactions are calculated as part of the `storage_root`.
// These transactions can be reused for importing the block into the
// storage. So, we cache them to not require a recomputation of those transactions.
type StorageTransactionCache[H runtime.Hash, Hasher runtime.Hasher[H]] struct {
	// Contains the changes for the main and the child storages as one transaction.
	transaction BackendTransaction[H, Hasher]
	// The storage root after applying the transaction.
	transactionStorageRoot H
}

/*type OverlayedChanges[H runtime.Hash, Hasher runtime.Hasher[H]] struct {
	// Top level storage changes.
	top OverlayedChangeSet
	// Child storage changes. The map key is the child storage key without the common prefix.
	children map[string]childStorageValue
	// Offchain related changes.
	offchain OffchainOverlayedChanges
	// Transaction index changes,
	transactionIndexOps []IndexOperation
	// True if extrinsics stats must be collected.
	collectExtrinsics bool
	// Collect statistic on this execution.
	stats StateMachineStats
	// Caches the "storage transaction" that is created while calling `storage_root`.
	// This transaction can be applied to the backend to persist the state changes.
	storageTransactionCache *StorageTransactionCache[H, Hasher]
}

// Whether no changes are contained in the top nor in any of the child changes.
func (oc *OverlayedChanges[H, Hasher]) IsEmpty() bool {
	return oc.top.IsEmpty() && len(oc.children) == 0
}

// Ask to collect/not to collect extrinsics indices where key(s) has been changed.
func (oc *OverlayedChanges[H, Hasher]) SetCollectExtrinsic(collectExtrinsic bool) {
	oc.collectExtrinsics = collectExtrinsic
}

// Returns (nil, false) if the key is unknown (i.e. and the query should be referred
// to the backend); (nil, true) if the key has been deleted. or a (value, true) for a key whose
// value has been set.
func (oc *OverlayedChanges[H, Hasher]) Storage(key string) ([]byte, bool) {
	entry := oc.top.Get(key)
	if entry == nil {
		return nil, false
	}

	value := entry.ValueRef()
	if value == nil {
		oc.stats.TallyReadModified(0)
		return nil, true
	}

	//oc.stats.TallyReadModified(uint64(len(*value)))
	//return *value, true
	panic("TODO")
}

// Should be called when there are changes that require to reset the
func (oc *OverlayedChanges[H, Hasher]) markDirty() {
	oc.storageTransactionCache = nil
}

// Returns (nil, false) if the key is unknown (i.e. and the query should be referred
// to the backend); (nil, true) if the key has been deleted. or a (value, true) for a key whose
// value has been set.
func (oc *OverlayedChanges[H, Hasher]) ChildStorage(childInfo ChildInfo, key *string) ([]byte, bool) {
	entry, has := oc.children[string(childInfo.StorageKey())]

	if !has {
		return nil, false
	}

	value := entry.OverlayedChangeSet.Get(*key).ValueRef()
	if value == nil {
		oc.stats.TallyReadModified(0)
		return nil, true
	}

	//oc.stats.TallyReadModified(uint64(len(*value)))
	//return *value, true

	panic("TODO")

}

func (oc *OverlayedChanges[H, Hasher]) SetStorage(key StorageKey, value StorageValue) {
	oc.markDirty()

	var sizeWrite uint64
	if value == nil {
		sizeWrite = 0
	} else {
		sizeWrite = uint64(len(value))
	}

	oc.stats.TallyWriteOverlay(sizeWrite)
	extrinsicIndex := oc.extrinsicIndex()
	oc.top.Set(string(key), value, extrinsicIndex)
}

// Returns current extrinsic index to use in changes trie construction.
// nil is returned if it is not set or changes trie config is not set.
// Persistent value (from the backend) can be ignored because runtime must
// set this index before first and unset after last extrinsic is executed.
// Changes that are made outside of extrinsics, are marked with
// `NO_EXTRINSIC_INDEX` index.
func (oc *OverlayedChanges[H, Hasher]) extrinsicIndex() *uint32 {
	if !oc.collectExtrinsics {
		return nil
	}

	val, has := oc.Storage(string(keys.ExtrinsicIndexKey))
	if !has {
		return nil
	}

	var result uint32
	err := scale.Unmarshal(val, &result)
	if err != nil {
		return &NoExtrinsicIndex
	}

	return &result
}*/
