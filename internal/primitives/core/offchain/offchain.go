// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package offchain

// Offchain DB persistent (non-fork-aware) storage.
type OffchainStorage interface {
	// Persist a value in storage under given key and prefix.
	Set(prefix, key, value []byte)

	// Clear a storage entry under given key and prefix.
	Remove(prefix, key []byte)

	// Retrieve a value from storage under given key and prefix.
	Get(prefix, key []byte) []byte

	// Replace the value in storage if given old_value matches the current one.
	//
	// Returns true if the value has been set and false otherwise.
	CompareAndSet(prefix, key, oldValue, newValue []byte) bool
}

// Change to be applied to the offchain worker db in regards to a key.
type OffchainOverlayedChanges interface {
	OffchainOverlayedChangeRemove | OffchainOverlayedChangeSetValue
}

// Change to be applied to the offchain worker db in regards to a key.
type OffchainOverlayedChange any

// Remove the data associated with the key
type OffchainOverlayedChangeRemove struct{}

// Overwrite the value of an associated key
type OffchainOverlayedChangeSetValue []byte
