// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package trie

import (
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/trie/tracking"
)

// Deltas is the interface for the trie local deltas since
// the last snapshot.
type Deltas interface {
	DeltaMerger
	DeltaDeletedGetter
}

// DeltaMerger merges the given deltas into the current
// deltas.
type DeltaMerger interface {
	MergeWith(deltas tracking.DeletedGetter)
}

// DeltaDeletedGetter returns the deleted node hashes recorded so far.
type DeltaDeletedGetter interface {
	Deleted() (nodeHashes map[common.Hash]struct{})
}

// DeltaRecorder records deltas done in a ongoing trie operation.
type DeltaRecorder interface {
	RecordDeleted(nodeHash common.Hash)
}
