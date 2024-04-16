// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package triedb

// Entries returns all the key-value pairs in the trie as a map of keys to values
// where the keys are encoded in Little Endian.
func (t *TrieDB) Entries() (keyValueMap map[string][]byte) {
	panic("not implemented yet")
}

// NextKey returns the next key in the trie in lexicographic order.
// It returns nil if no next key is found.
func (t *TrieDB) NextKey(key []byte) []byte {
	panic("not implemented yet")
}
