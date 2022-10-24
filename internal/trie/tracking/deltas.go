// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package tracking

import (
	"github.com/ChainSafe/gossamer/lib/common"
)

// Deltas tracks the trie deltas, for example deleted node hashes.
type Deltas struct {
	deletedNodeHashes  map[common.Hash]struct{}
	insertedNodeHashes map[common.Hash]struct{}
}

// New returns a new Deltas struct.
func New() *Deltas {
	return &Deltas{
		deletedNodeHashes:  make(map[common.Hash]struct{}),
		insertedNodeHashes: make(map[common.Hash]struct{}),
	}
}

// RecordDeleted records a node hash as deleted.
func (d *Deltas) RecordDeleted(nodeHash common.Hash) {
	_, insertionPending := d.insertedNodeHashes[nodeHash]
	if insertionPending {
		// Inserted node got re-deleted in the same operation,
		// so no need to track it as deleted, just remove it from
		// the inserted node hashes set.
		delete(d.insertedNodeHashes, nodeHash)
	} else {
		d.deletedNodeHashes[nodeHash] = struct{}{}
	}
}

// RecordInserted records a node hash as inserted.
func (d *Deltas) RecordInserted(nodeHash common.Hash) {
	_, deletionPending := d.deletedNodeHashes[nodeHash]
	if deletionPending {
		// Deleted node got re-inserted in the same operation,
		// so no need to track it as inserted, just remove it from
		// the deleted node hashes set.
		delete(d.deletedNodeHashes, nodeHash)
	} else {
		d.insertedNodeHashes[nodeHash] = struct{}{}
	}
}

// Get returns the sets (maps) of all the recorded inserted
// and deleted node hashes. Note the map returned is not deep
// copied for performance reasons and so it's not safe for mutation.
func (d *Deltas) Get() (insertedNodeHashes, deletedNodeHashes map[common.Hash]struct{}) {
	return d.insertedNodeHashes, d.deletedNodeHashes
}

// MergeWith merges the deltas given as argument in the receiving
// deltas struct.
func (d *Deltas) MergeWith(pendingDeltas Getter, mergeDeleted bool) {
	insertedNodeHashes, deletedNodeHashes := pendingDeltas.Get()

	for nodeHash := range insertedNodeHashes {
		_, insertedSinceSnapshot := d.insertedNodeHashes[nodeHash]
		if insertedSinceSnapshot {
			// Node has already been inserted since the last snapshot
			continue
		}

		_, deletedSinceSnapshot := d.deletedNodeHashes[nodeHash]
		if deletedSinceSnapshot {
			// Node has been re-inserted, so just delete the deleted node hash
			// from the tracking of deleted node hashes.
			delete(d.deletedNodeHashes, nodeHash)
			continue
		}

		d.insertedNodeHashes[nodeHash] = struct{}{}
	}

	for nodeHash := range deletedNodeHashes {
		_, deletedSinceSnapshot := d.deletedNodeHashes[nodeHash]
		if deletedSinceSnapshot {
			// Node has already been deleted since the last snapshot
			continue
		}

		_, insertedSinceSnapshot := d.insertedNodeHashes[nodeHash]
		if insertedSinceSnapshot {
			// Node has been re-deleted, so just delete the inserted node hash
			// from the tracking of inserted node hashes.
			delete(d.insertedNodeHashes, nodeHash)
			continue
		}

		if mergeDeleted {
		d.deletedNodeHashes[nodeHash] = struct{}{}
		}
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

	if d.insertedNodeHashes != nil {
		deepCopy.insertedNodeHashes = make(map[common.Hash]struct{}, len(d.insertedNodeHashes))
		for nodeHash := range d.insertedNodeHashes {
			deepCopy.insertedNodeHashes[nodeHash] = struct{}{}
		}
	}

	return deepCopy
}
