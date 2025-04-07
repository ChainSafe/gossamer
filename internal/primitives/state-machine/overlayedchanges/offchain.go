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

type OffchainOverlayedChange interface {
	isOffchainOverlayedChange()
}

type (
	OffchainOverlayedChangeRemove   struct{}
	OffchainOverlayedChangeSetValue []byte
)

func (OffchainOverlayedChangeRemove) isOffchainOverlayedChange()   {}
func (OffchainOverlayedChangeSetValue) isOffchainOverlayedChange() {}

// In-memory storage for offchain workers recording changes for the actual offchain storage
// implementation.
type OffchainOverlayedChanges struct {
	OverlayedMap[string, OffchainOverlayedChange, *GenericOverlayedEntry[OffchainOverlayedChange]]
}

func NewOffchainOverlayedChanges() OffchainOverlayedChanges {
	return OffchainOverlayedChanges{
		NewOverlayedMap[string, OffchainOverlayedChange, *GenericOverlayedEntry[OffchainOverlayedChange]](),
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
	oc.SetOffchain(prefixedKey, OffchainOverlayedChangeSetValue(value), nil)
}

// Remove a key and its associated value from the offchain database.
func (oc *OffchainOverlayedChanges) Remove(prefix []byte, key []byte) {
	prefixedKey := string(append(prefix, key...))
	oc.SetOffchain(prefixedKey, OffchainOverlayedChangeRemove{}, nil)
}
