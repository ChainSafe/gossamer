// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package triedb

// Entries returns all the key-value pairs in the trie as a map of keys to values
// where the keys are encoded in Little Endian.
func (tdb *TrieDB) Entries() (keyValueMap map[string][]byte) {
	panic("not implemented yet")
}

// NextKey returns the next key in the trie in lexicographic order.
// It returns nil if no next key is found.
func (tdb *TrieDB) NextKey(key []byte) []byte {
	iter, err := NewTrieDBIterator(tdb)
	if err != nil {
		panic("Unexpected error creating trie iterator")
	}

	iter.Seek(key)
	nextKey := iter.NextKey()
	return nextKey
}
