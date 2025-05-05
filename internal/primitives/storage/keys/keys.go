// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package keys

import (
	"bytes"
	"strings"
)

// List of all well known keys and prefixes in storage.
var (
	Code = []byte(":code")
	// DefaultChildStorageKeyPrefix is a prefix of the default child storage keys in the top trie.
	DefaultChildStorageKeyPrefix = []byte(":child_storage:default:")
	// Prefix of child storage keys
	ChildStorageKeyPrefix = []byte(":child_storage:")
	// Current extrinsic index (u32) is stored under this key.
	// Encodes to `0x3a65787472696e7369635f696e646578`.
	ExtrinsicIndexKey = []byte(":extrinsic_index")
)

// IsChildStorageKey returns whether a key is a child storage key.
//
// This is convenience function which basically checks if the given `key` starts
// with `ChildStorageKeyPrefix` and doesn't do anything apart from that.
func IsChildStorageKey(key []byte) bool {
	return bytes.HasPrefix(key, ChildStorageKeyPrefix)
}

// IsDefaultChildStorageKey returns whether a key is a child storage key.
//
// This is convenience function which basically checks if the given key starts
// with [DefaultChildStorageKeyPrefix].
func IsDefaultChildStorageKey(key []byte) bool {
	i := strings.Index(string(key), string(DefaultChildStorageKeyPrefix))
	return i == 0
}

// Returns if the given key starts with [ChildStorageKeyPrefix] or collides with it.
func StartsWithChildStorageKey(key []byte) bool {
	if len(key) > len(ChildStorageKeyPrefix) {
		return bytes.HasPrefix(key, ChildStorageKeyPrefix)
	}
	return bytes.HasPrefix(ChildStorageKeyPrefix, key)
}
