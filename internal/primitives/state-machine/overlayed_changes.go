// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package statemachine

import (
	"github.com/ChainSafe/gossamer/internal/primitives/core/offchain"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
)

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

type ChildInfo interface {
	isChildInfo()
}

type ChildInfoParentKeyId struct {
	ParentKeyId []byte
}

func (ChildInfoParentKeyId) isChildInfo() {}

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

type OverlayedChanges[H runtime.Hash, Hasher runtime.Hasher[H]] struct {
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
