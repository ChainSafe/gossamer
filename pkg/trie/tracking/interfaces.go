// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package tracking

import "github.com/ChainSafe/gossamer/lib/common"

// Getter gets deleted node hashes.
type Getter interface {
	Deleted() (nodeHashes map[common.Hash]struct{})
}

// Deltas is the interface for the trie local deltas since
// the last snapshot.
type Delta interface {
	DeltaMerger
	Getter
	DeltaRecorder
	DeepCopy() (deepCopy *Deltas)
}

// DeltaMerger merges the given deltas into the current
// deltas.
type DeltaMerger interface {
	MergeWith(deltas Getter)
}

// DeltaRecorder records deltas done in a ongoing trie operation.
type DeltaRecorder interface {
	RecordDeleted(nodeHash common.Hash)
}
