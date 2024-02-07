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
	tracking.DeletedGetter
	DeltaRecorder
	DeepCopy() (deepCopy *tracking.Deltas)
}

// DeltaMerger merges the given deltas into the current
// deltas.
type DeltaMerger interface {
	MergeWith(deltas tracking.DeletedGetter)
}

// DeltaRecorder records deltas done in a ongoing trie operation.
type DeltaRecorder interface {
	RecordDeleted(nodeHash common.Hash)
}
