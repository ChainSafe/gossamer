// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package statedb

// State database maintenance. Handles canonicalization and pruning in the database.
//
// # Canonicalization.
// Canonicalization window tracks a tree of blocks identified by header hash. The in-memory
// overlay allows to get any trie node that was inserted in any of the blocks within the window.
// The overlay is journaled to the backing database and rebuilt on startup.
// There's a limit of 32 blocks that may have the same block number in the canonicalization window.
//
// Canonicalization function selects one root from the top of the tree and discards all other roots
// and their subtrees. Upon canonicalization all trie nodes that were inserted in the block are
// added to the backing DB and block tracking is moved to the pruning window, where no forks are
// allowed.
//
// # Canonicalization vs Finality
// Database engine uses a notion of canonicality, rather then finality. A canonical block may not
// be yet finalized from the perspective of the consensus engine, but it still can't be reverted in
// the database. Most of the time during normal operation last canonical block is the same as last
// finalized. However if finality stall for a long duration for some reason, there's only a certain
// number of blocks that can fit in the non-canonical overlay, so canonicalization of an
// unfinalized block may be forced.
//
// # Pruning.
// See `RefWindow` for pruning algorithm details. `StateDb` prunes on each canonicalization until
// pruning constraints are satisfied.

import (
	"errors"

	"github.com/ChainSafe/gossamer/pkg/scale"
)

// Database value type.
type DBValue []byte

// Basic set of requirements for the Block hash and node key types.
type Hash interface {
	comparable
}

// HashDBValue is a helper struct which contains Hash and DBValue.
type HashDBValue[H any] struct {
	Hash H
	DBValue
}

// Backend database interface for metadata. Read-only.
type MetaDB interface {
	// Get meta value, such as the journal.
	GetMeta(key []byte) (*DBValue, error)
}

// Backend database trait. Read-only.
type NodeDB[Key comparable] interface {
	// Get state trie node.
	Get(key Key) (*DBValue, error)
}

var (
	// Trying to canonicalize invalid block.
	ErrInvalidBlock = errors.New("trying to canonicalize invalid block")
	// Trying to insert block with invalid number.
	ErrInvalidBlockNumber = errors.New("trying to insert block with invalid number")
	// Trying to insert block with unknown parent.
	ErrInvalidParent = errors.New("trying to insert block with unknown parent")
	// Invalid pruning mode specified. Contains expected mode.
	ErrIncompatiblePruningModes = errors.New("incompatible pruning modes")
	// Trying to insert existing block.
	ErrBlockAlreadyExists = errors.New("block already exists")
	// Trying to get a block record from db while it is not commit to db yet
	ErrBlockUnavailable = errors.New("trying to get a block record from db while it is not commit to db yet")
	// Invalid metadata
	ErrMetadata = errors.New("Invalid metadata:")
)

// A set of state node changes.
type ChangeSet[H any] struct {
	// Inserted nodes.
	Inserted []HashDBValue[H]
	// Deleted nodes.
	Deleted []H
}

// A set of changes to the backing database.
type CommitSet[H Hash] struct {
	// State node changes.
	Data ChangeSet[H]
	// Metadata changes.
	Meta ChangeSet[[]byte]
}

func toMetaKey(suffix []byte, data any) []byte {
	key := scale.MustMarshal(data)
	key = append(key, suffix...)
	return key
}
