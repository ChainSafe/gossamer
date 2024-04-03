package triedb

// Entries returns all the key-value pairs in the trie as a map of keys to values
// where the keys are encoded in Little Endian.
func (tdb *TrieDB) Entries() (keyValueMap map[string][]byte) {
	panic("not implemented yet")
}

// NextKey returns the next key in the trie in lexicographic order.
// It returns nil if no next key is found.
func (tdb *TrieDB) NextKey(key []byte) []byte {
	panic("not implemented yet")
}
