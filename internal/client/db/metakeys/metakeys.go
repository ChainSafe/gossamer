// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package metakeys

// Keys of entries in COLUMN_META.

// Type of storage (full or light).
var Type = []byte("type")

// BestBlock key.
var BestBlock = []byte("best")

// FinalizedBlock is last finalized block key.
var FinalizedBlock = []byte("final")

// FinalizedStgate is last finalized state key.
var FinalizedState = []byte("fstate")

// BlockGap key.
var BlockGap = []byte("gap")

// GenesisHash is genesis block hash key.
var GenesisHash = []byte("gen")

// LeafPrefix is leaves prefix list key.
var LeafPrefix = []byte("leaf")

// ChildrenPrefix is children prefix list key.
var ChildrenPrefix = []byte("children")
