// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package offchain

// OffchainStorage is offchain DB persisted (non-fork-aware) storage.
type OffchainStorage interface {
	// Set persists a value in storage under given key and prefix.
	Set(prefix, key, value []byte)

	// Remove clears a storage entry under given key and prefix.
	Remove(prefix, key []byte)

	// Get retrieves a value from storage under given key and prefix.
	Get(prefix, key []byte) []byte

	// CompareAndSet will replace the value in storage if given oldValue matches the current one.
	//
	// Returns true if the value has been set and false otherwise.
	CompareAndSet(prefix, key, oldValue, newValue []byte) bool
}

// OffchainOverlayedChanges is a change to be applied to the offchain worker db in regards to a key.
type OffchainOverlayedChanges interface {
	OffchainOverlayedChangeRemove | OffchainOverlayedChangeSetValue
}

// OffchainOverlayedChange is a change to be applied to the offchain worker db in regards to a key.
type OffchainOverlayedChange any

// OffchainOverlayedChangeRemove removes the data associated with the key
type OffchainOverlayedChangeRemove struct{}

// OffchainOverlayedChangeSetValue overwrites the value of an associated key
type OffchainOverlayedChangeSetValue []byte
