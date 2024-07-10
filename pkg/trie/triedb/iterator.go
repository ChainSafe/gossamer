// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package triedb

// Entries returns all the key-value pairs in the trie as a map of keys to values
// where the keys are encoded in Little Endian.
func (t *TrieDB) Entries() (keyValueMap map[string][]byte) {
	entries := make(map[string][]byte)

	iter := NewTrieDBIterator(t)
	for entry := iter.NextEntry(); entry != nil; entry = iter.NextEntry() {
		entries[string(entry.Key)] = entry.Value
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
	iter := NewPrefixedTrieDBIterator(t, prefix)

	keys := make([][]byte, 0)

	for key := iter.NextKey(); key != nil; key = iter.NextKey() {
		keys = append(keys, key)
	}

	return keys
}
