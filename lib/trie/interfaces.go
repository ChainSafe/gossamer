// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package trie

import (
	"github.com/ChainSafe/gossamer/internal/trie/tracking"
	"github.com/ChainSafe/gossamer/lib/common"
)

// Database is an interface to interact with a key value database.
type Database interface {
	Getter
	Putter
}

// Getter is an interface to get values from a
// key value database.
type Getter interface {
	Get(key []byte) (value []byte, err error)
}

// Putter is an interface to put key value pairs in a
// key value database.
type Putter interface {
	Put(key, value []byte) (err error)
}

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
	// RecordDeleted records a node hash or storage value hash as deleted.
	RecordDeleted(hash common.Hash)
}
