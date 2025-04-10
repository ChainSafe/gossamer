// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package overlayedchanges

import "github.com/ChainSafe/gossamer/internal/primitives/core/offchain"

type OffchainChangesCollection []struct {
	PrefixKey struct {
		Prefix []byte
		Key    []byte
	}
	ValueOperation offchain.OffchainOverlayedChange
}

// In-memory storage for offchain workers recording changes for the actual offchain storage
// implementation.
type OffchainOverlayedChanges struct {
	OverlayedMap[string, offchain.OffchainOverlayedChange, *GenericOverlayedEntry[offchain.OffchainOverlayedChange]]
}

func NewOffchainOverlayedChanges() OffchainOverlayedChanges {
	return OffchainOverlayedChanges{
		NewOverlayedMap[string, offchain.OffchainOverlayedChange, *GenericOverlayedEntry[offchain.OffchainOverlayedChange]](),
	}
}

func (oc OffchainOverlayedChanges) Clone() OffchainOverlayedChanges {
	return OffchainOverlayedChanges{
		oc.OverlayedMap.Clone(),
	}
}

// Remove a key and its associated value from the offchain database.
func (oc *OffchainOverlayedChanges) Set(prefix []byte, key []byte, value []byte) {
	prefixedKey := string(append(prefix, key...))
	oc.SetOffchain(prefixedKey, offchain.OffchainOverlayedChangeSetValue(value), nil)
}

// Remove a key and its associated value from the offchain database.
func (oc *OffchainOverlayedChanges) Remove(prefix []byte, key []byte) {
	prefixedKey := string(append(prefix, key...))
	oc.SetOffchain(prefixedKey, offchain.OffchainOverlayedChangeRemove{}, nil)
}
