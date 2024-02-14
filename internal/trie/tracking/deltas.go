// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package tracking

import (
	"github.com/ChainSafe/gossamer/lib/common"
)

type DeltaEntry struct {
	Key, Value []byte
}

// Deltas tracks the trie deltas, for example deleted node hashes.
type Deltas struct {
	deletedNodeHashes map[common.Hash]struct{}
	updatedNodes      []DeltaEntry
}

// New returns a new Deltas struct.
func New() *Deltas {
	return &Deltas{
		deletedNodeHashes: make(map[common.Hash]struct{}),
		updatedNodes:      make([]DeltaEntry, 0),
	}
}

// RecordDeleted records a node hash as deleted.
func (d *Deltas) RecordDeleted(nodeHash common.Hash) {
	d.deletedNodeHashes[nodeHash] = struct{}{}
}

func (d *Deltas) RecordUpdated(key, value []byte) {
	entryKey := make([]byte, len(key))
	copy(entryKey[:], key[:])

	entryValue := make([]byte, len(value))
	copy(entryValue[:], value[:])

	d.updatedNodes = append(d.updatedNodes, DeltaEntry{Key: entryKey, Value: entryValue})
}

// Deleted returns a set (map) of all the recorded deleted
// node hashes. Note the map returned is not deep copied for
// performance reasons and so it's not safe for mutation.
func (d *Deltas) Deleted() (nodeHashes map[common.Hash]struct{}) {
	return d.deletedNodeHashes
}

func (d *Deltas) Updated() (entries []DeltaEntry) {
	return d.updatedNodes
}

// MergeWith merges the deltas given as argument in the receiving
// deltas struct.
func (d *Deltas) MergeWith(deltas Getter) {
	for nodeHash := range deltas.Deleted() {
		d.RecordDeleted(nodeHash)
	}

	for _, entry := range deltas.Updated() {
		d.RecordUpdated(entry.Key, entry.Value)
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

	if d.updatedNodes != nil {
		deepCopy.updatedNodes = make([]DeltaEntry, 0, len(d.updatedNodes))
		for _, entry := range d.updatedNodes {
			deepCopy.RecordUpdated(entry.Key, entry.Value)
		}
	}

	return deepCopy
}
