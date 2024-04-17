// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package triedb

import "bytes"

// Entries returns all the key-value pairs in the trie as a map of keys to values
// where the keys are encoded in Little Endian.
func (t *TrieDB) Entries() (keyValueMap map[string][]byte) {
	entries := make(map[string][]byte)

	iter := NewTrieDBIterator(t)
	for entry := iter.NextEntry(); entry != nil; entry = iter.NextEntry() {
		entries[string(entry.key)] = entry.value
	}

	return entries
}

// NextKey returns the next key in the trie in lexicographic order.
// It returns nil if no next key is found.
func (t *TrieDB) NextKey(key []byte) []byte {
	iter := NewTrieDBIterator(t)

	// TODO: Seek will potentially skip a lot of keys, we need to find a way to
	// optimise it, maybe creating a lookupFor
	iter.Seek(key)
	return iter.NextKey()
}

// GetKeysWithPrefix returns all keys in little Endian
// format from nodes in the trie that have the given little
// Endian formatted prefix in their key.
func (t *TrieDB) GetKeysWithPrefix(prefix []byte) (keysLE [][]byte) {
	iter := NewTrieDBIterator(t)

	// TODO: this method could be expensive if we have to skip a big amount of keys
	// We could optimise it by traversing the trie following the targetKey path and
	// going directly to the key we are looking for, then visiting its children
	iter.Seek(prefix)

	//Since seek consumes the prefix, we need to add it in the keys list
	keys := [][]byte{prefix}

	for key := iter.NextKey(); bytes.HasPrefix(key, prefix); key = iter.NextKey() {
		keys = append(keys, key)
	}

	return keys
}
