// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package tracking

import (
	"github.com/ChainSafe/gossamer/lib/common"
)

type DeltaEntry struct {
	Key   []byte
	Value []byte
}

// Deltas tracks the trie deltas, for example deleted node hashes.
type Deltas struct {
	entries           []DeltaEntry
	deletedNodeHashes map[common.Hash]struct{}
}

// New returns a new Deltas struct.
func New() *Deltas {
	return &Deltas{
		entries:           make([]DeltaEntry, 0),
		deletedNodeHashes: make(map[common.Hash]struct{}),
	}
}

// RecordDeleted records a node hash as deleted.
func (d *Deltas) RecordDeleted(nodeHash common.Hash) {
	d.deletedNodeHashes[nodeHash] = struct{}{}
}

// RecordUpdated records a node hash that was created or updated.
func (d *Deltas) RecordUpdated(key, value []byte) {
	newEntry := DeltaEntry{
		Key:   make([]byte, len(key)),
		Value: make([]byte, len(value)),
	}

	copy(newEntry.Key[:], key[:])
	copy(newEntry.Value[:], value[:])
	d.entries = append(d.entries, newEntry)
}

func (d *Deltas) HasUpdated(partialKeyHash common.Hash) bool {
	return false
}

// Deleted returns a set (map) of all the recorded deleted
// node hashes. Note the map returned is not deep copied for
// performance reasons and so it's not safe for mutation.
func (d *Deltas) Deleted() (nodeHashes map[common.Hash]struct{}) {
	return d.deletedNodeHashes
}

func (d *Deltas) Updated() []DeltaEntry {
	return d.entries
}

// MergeWith merges the deltas given as argument in the receiving
// deltas struct.
func (d *Deltas) MergeWith(deltas Getter) {
	for nodeHash := range deltas.Deleted() {
		d.RecordDeleted(nodeHash)
	}

	for _, deltaEntry := range deltas.Updated() {
		d.RecordUpdated(deltaEntry.Key, deltaEntry.Value)
	}
}

// DeepCopy returns a deep copy of the deltas.
func (d *Deltas) DeepCopy() (deepCopy *Deltas) {
	if d == nil {
		return nil
	}

	deepCopy = &Deltas{}

	if d.deletedNodeHashes != nil {
		deepCopy.deletedNodeHashes = make(map[common.Hash]struct{}, len(d.deletedNodeHashes))
		for nodeHash := range d.deletedNodeHashes {
			deepCopy.deletedNodeHashes[nodeHash] = struct{}{}
		}
	}

	if len(d.entries) != 0 {
		deepCopy.entries = make([]DeltaEntry, len(d.entries))
		for idx, deltaEntry := range d.entries {
			newEntry := DeltaEntry{
				Key:   make([]byte, len(deltaEntry.Key)),
				Value: make([]byte, len(deltaEntry.Value)),
			}

			copy(newEntry.Key[:], deltaEntry.Key[:])
			copy(newEntry.Value[:], deltaEntry.Value[:])
			deepCopy.entries[idx] = newEntry
		}
	}
	return deepCopy
}
