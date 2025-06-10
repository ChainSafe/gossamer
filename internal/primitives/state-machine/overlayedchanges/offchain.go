// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package overlayedchanges

import (
	"fmt"
	"iter"
	"strings"

	"github.com/ChainSafe/gossamer/internal/primitives/core/offchain"
)

const prefixKeySeparator = "__::__"

type PrefixKey struct {
	Prefix []byte
	Key    []byte
}

type OffchainChange struct {
	PrefixKey      PrefixKey
	ValueOperation offchain.OffchainOverlayedChange
}

type OffchainChangesCollection []OffchainChange

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

func (oc OffchainOverlayedChanges) Drain() iter.Seq2[PrefixKey, offchain.OffchainOverlayedChange] {
	return func(yield func(PrefixKey, offchain.OffchainOverlayedChange) bool) {
		oc.changes.Scan(func(k string, v *GenericOverlayedEntry[offchain.OffchainOverlayedChange]) bool {
			prefix := strings.Split(k, prefixKeySeparator)
			prefixKey := PrefixKey{
				Prefix: []byte(prefix[0]),
				Key:    []byte(prefix[1]),
			}
			return yield(prefixKey, v.PopTransaction().value)
		})
	}
}

func (oc OffchainOverlayedChanges) Clone() OffchainOverlayedChanges {
	return OffchainOverlayedChanges{
		oc.OverlayedMap.Clone(),
	}
}

// Set a key and its associated value in the offchain database.
func (oc *OffchainOverlayedChanges) Set(prefix []byte, key []byte, value []byte) {
	oc.SetOffchain(prefixKey(prefix, key), offchain.OffchainOverlayedChangeSetValue(value), nil)
}

// Remove a key and its associated value from the offchain database.
func (oc *OffchainOverlayedChanges) Remove(prefix []byte, key []byte) {
	oc.SetOffchain(prefixKey(prefix, key), offchain.OffchainOverlayedChangeRemove{}, nil)
}

func prefixKey(prefix []byte, key []byte) string {
	return fmt.Sprintf("%s%s%s", prefix, prefixKeySeparator, key)
}
