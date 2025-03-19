// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package keys

import (
	"strings"
)

// List of all well known keys and prefixes in storage.
var (
	// DefaultChildStorageKeyPrefix is a prefix of the default child storage keys in the top trie.
	DefaultChildStorageKeyPrefix = []byte(":child_storage:default:")
)

// IsChildStorageKey returns whether a key is a child storage key.
//
// This is convenience function which basically checks if the given key starts
// with [DefaultChildStorageKeyPrefix].
func IsChildStorageKey(key []byte) bool {
	i := strings.Index(string(key), string(DefaultChildStorageKeyPrefix))
	return i == 0
}
