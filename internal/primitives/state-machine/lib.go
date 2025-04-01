// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package state_machine

import "github.com/ChainSafe/gossamer/internal/primitives/state-machine/backend"

// A key value state at any storage level.
type KeyValueStorageLevel struct {
	/// State root of the level, for
	/// top trie it is as an empty byte array.
	StateRoot []byte
	// Storage of parents, empty for top root or
	// when exporting (building proof).
	ParentStorageKeys [][]byte
	// Pair of key and values from this state.
	KeyValues []backend.StorageKeyValue
}

// Multiple key value state.
// States are ordered by root storage key.
type KeyValueStates []KeyValueStorageLevel
