// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package triedb

import "bytes"

// Entries returns all the key-value pairs in the trie as a map of keys to values
// where the keys are encoded in Little Endian.
func (t *TrieDB) Entries() (keyValueMap map[string][]byte) {
	entries := make(map[string][]byte)

	iter := NewTrieDBIterator(t)
	for key, value := iter.NextEntry(); key != nil; key, value = iter.NextEntry() {
		entries[string(key)] = value
	}

	return entries
}

// NextKey returns the next key in the trie in lexicographic order.
// It returns nil if no next key is found.
func (t *TrieDB) NextKey(key []byte) []byte {
	iter := NewTrieDBIterator(t)

	iter.Seek(key)
	nextKey := iter.NextKey()
	return nextKey
}

// GetKeysWithPrefix returns all keys in little Endian
// format from nodes in the trie that have the given little
// Endian formatted prefix in their key.
func (t *TrieDB) GetKeysWithPrefix(prefix []byte) (keysLE [][]byte) {
	iter := NewTrieDBIterator(t)
	iter.Seek(prefix)

	//Since seek consumes the prefix, we need to add it in the keys list
	keys := [][]byte{prefix}

	for key := iter.NextKey(); bytes.HasPrefix(key, prefix); key = iter.NextKey() {
		keys = append(keys, key)
	}

	return keys
}
