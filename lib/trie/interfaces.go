// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package trie

import (
	"github.com/ChainSafe/gossamer/internal/trie/tracking"
	"github.com/ChainSafe/gossamer/lib/common"
)

// Deltas is the interface for the trie local deltas since
// the last snapshot.
type Deltas interface {
	DeltaMerger
	DeltaGetter
}

// DeltaMerger merges the given deltas into the current
// deltas.
type DeltaMerger interface {
	MergeWith(deltas tracking.Getter, mergeDeleted bool)
}

// DeltaGetter returns the deleted node hashes recorded so far.
type DeltaGetter interface {
	Get() (insertedNodeHashes, deletedNodeHashes map[common.Hash]struct{})
}

// DeltaRecorder records deltas done in a ongoing trie operation.
type DeltaRecorder interface {
	DeltaDeletedRecorder
	RecordInserted(nodeHash common.Hash)
}

// DeltaDeletedRecorder records deleted deltas done in a ongoing trie operation.
type DeltaDeletedRecorder interface {
	RecordDeleted(nodeHash common.Hash)
}

// DeltaInsertedRecorder records inserted deltas done in a ongoing trie operation.
type DeltaInsertedRecorder interface {
	RecordInserted(nodeHash common.Hash)
}
