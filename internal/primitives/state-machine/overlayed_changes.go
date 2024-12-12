// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package statemachine

import "github.com/ChainSafe/gossamer/internal/primitives/core/offchain"

// Storage key.
type StorageKey []byte

// Storage value. Value can be nil
type StorageValue []byte

// Storage key and value.
type StorageKeyValue struct {
	StorageKey
	StorageValue
}

// In memory array of storage values.
type StorageCollection []StorageKeyValue

// In memory arrays of storage values for multiple child tries.
type ChildStorageCollection []struct {
	StorageKey
	StorageCollection
}

// In memory array of storage values.
type OffchainChangesCollection []struct {
	PrefixKey struct {
		Prefix []byte
		Key    []byte
	}
	ValueOperation offchain.OffchainOverlayedChange
}

// Transaction index operation.
type IndexOperations interface {
	IndexOperationInsert | IndexOperationRenew
}

// Transaction index operation.
type IndexOperation interface {
	isIndexOperation()
}

// Insert transaction into index.
type IndexOperationInsert struct {
	// Extrinsic index in the current block.
	Extrinsic uint32
	// Data content hash.
	Hash []byte
	// Indexed data size.
	Size uint32
}

// Renew existing transaction storage.
type IndexOperationRenew struct {
	// Extrinsic index in the current block.
	Extrinsic uint32
	// Referenced index hash.
	Hash []byte
}

func (IndexOperationInsert) isIndexOperation() {}
func (IndexOperationRenew) isIndexOperation()  {}
