package triedb

// / Iterator for going through all nodes in the trie in pre-order traversal order.
// pub struct TrieDBRawIterator<L: TrieLayout> {
type TrieDBRawIterator[H any] struct {
	// trail: Vec<Crumb<L::Hash>>,
	// key_nibbles: NibbleVec,
}

func (TrieDBRawIterator[H]) NextKey(db TrieDB[H]) (*[]byte, error) {
	panic("unimpl")
}
