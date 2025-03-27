// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package overlayedchanges

import "github.com/ChainSafe/gossamer/internal/primitives/core/offchain"

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
